package main

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"pob_api/pricer"
	"pob_api/translator"
)

// ---------------------------------------------------------------------------
// POB Code → XML decoding (same logic as pob_decode tool)
// ---------------------------------------------------------------------------

func decodePOBCode(pobCode string) ([]byte, error) {
	pobCode = strings.TrimSpace(pobCode)
	if len(pobCode) == 0 {
		return nil, fmt.Errorf("empty POB code")
	}

	// Base64url → standard Base64
	b64Std := strings.NewReplacer("-", "+", "_", "/").Replace(pobCode)

	// Decode Base64
	decoded, err := base64.StdEncoding.WithPadding(base64.NoPadding).DecodeString(b64Std)
	if err != nil {
		decoded, err = base64.StdEncoding.DecodeString(b64Std)
		if err != nil {
			return nil, fmt.Errorf("base64 decode failed: %w", err)
		}
	}

	// zlib decompress
	reader, err := zlib.NewReader(bytes.NewReader(decoded))
	if err != nil {
		return nil, fmt.Errorf("zlib init failed: %w", err)
	}
	defer reader.Close()

	xmlBytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("zlib decompress failed: %w", err)
	}

	return xmlBytes, nil
}

// ---------------------------------------------------------------------------
// LuaJIT Worker Process
// ---------------------------------------------------------------------------

type Worker struct {
	id     int
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	stderr io.ReadCloser
	mu     sync.Mutex // protects stdin/stdout access
	alive  bool
}

func newWorker(id int, srcDir, luaPath, luaCPath, workerScript string) (*Worker, error) {
	cmd := exec.Command("luajit", workerScript)
	cmd.Dir = srcDir
	cmd.Env = append(os.Environ(),
		"LUA_PATH="+luaPath,
		"LUA_CPATH="+luaCPath,
	)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	w := &Worker{
		id:     id,
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewReaderSize(stdout, 256*1024), // 256KB buffer
		stderr: stderr,
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start worker: %w", err)
	}

	// Forward stderr to log in background
	go w.drainStderr()

	// Wait for READY signal
	readyCh := make(chan error, 1)
	go func() {
		line, err := w.stdout.ReadString('\n')
		if err != nil {
			readyCh <- fmt.Errorf("waiting for READY: %w", err)
			return
		}
		line = strings.TrimSpace(line)
		if line != "READY" {
			readyCh <- fmt.Errorf("expected READY, got: %q", line)
			return
		}
		readyCh <- nil
	}()

	select {
	case err := <-readyCh:
		if err != nil {
			cmd.Process.Kill()
			return nil, err
		}
	case <-time.After(120 * time.Second): // HeadlessWrapper can be slow
		cmd.Process.Kill()
		return nil, fmt.Errorf("worker %d timed out during initialization", id)
	}

	w.alive = true
	log.Printf("[pool] Worker %d initialized and ready", id)
	return w, nil
}

func (w *Worker) drainStderr() {
	scanner := bufio.NewScanner(w.stderr)
	scanner.Buffer(make([]byte, 64*1024), 64*1024)
	for scanner.Scan() {
		log.Printf("[worker-%d] %s", w.id, scanner.Text())
	}
}

// Recalc sends XML to the worker and returns recalculated XML.
func (w *Worker) Recalc(xmlInput []byte) ([]byte, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.alive {
		return nil, fmt.Errorf("worker %d is dead", w.id)
	}

	// Send: "RECALC <len>\n" + data
	header := fmt.Sprintf("RECALC %d\n", len(xmlInput))
	if _, err := io.WriteString(w.stdin, header); err != nil {
		w.alive = false
		return nil, fmt.Errorf("write header: %w", err)
	}
	if _, err := w.stdin.Write(xmlInput); err != nil {
		w.alive = false
		return nil, fmt.Errorf("write payload: %w", err)
	}

	// Read response header: "OK <len>\n" or "ERR <len>\n"
	respLine, err := w.stdout.ReadString('\n')
	if err != nil {
		w.alive = false
		return nil, fmt.Errorf("read response header: %w", err)
	}
	respLine = strings.TrimSpace(respLine)

	parts := strings.SplitN(respLine, " ", 2)
	if len(parts) != 2 {
		w.alive = false
		return nil, fmt.Errorf("invalid response header: %q", respLine)
	}

	respLen, err := strconv.Atoi(parts[1])
	if err != nil {
		w.alive = false
		return nil, fmt.Errorf("invalid response length: %q", parts[1])
	}

	// Read response body
	body := make([]byte, respLen)
	if _, err := io.ReadFull(w.stdout, body); err != nil {
		w.alive = false
		return nil, fmt.Errorf("read response body: %w", err)
	}

	switch parts[0] {
	case "OK":
		return body, nil
	case "ERR":
		return nil, fmt.Errorf("worker error: %s", string(body))
	default:
		w.alive = false
		return nil, fmt.Errorf("unexpected response type: %q", parts[0])
	}
}

func (w *Worker) Ping() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.alive {
		return fmt.Errorf("worker %d is dead", w.id)
	}

	if _, err := io.WriteString(w.stdin, "PING\n"); err != nil {
		w.alive = false
		return err
	}

	line, err := w.stdout.ReadString('\n')
	if err != nil {
		w.alive = false
		return err
	}
	if strings.TrimSpace(line) != "PONG" {
		w.alive = false
		return fmt.Errorf("expected PONG, got: %q", line)
	}
	return nil
}

// ReplaceItem sends XML + replacement info to the worker and returns comparison JSON.
// Protocol: "REPLACE_ITEM <xml_len> <slot_len> <item_len>\n" + xml + slot + item
func (w *Worker) ReplaceItem(xmlInput []byte, slot string, itemText string) ([]byte, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.alive {
		return nil, fmt.Errorf("worker %d is dead", w.id)
	}

	slotBytes := []byte(slot)
	itemBytes := []byte(itemText)

	header := fmt.Sprintf("REPLACE_ITEM %d %d %d\n", len(xmlInput), len(slotBytes), len(itemBytes))
	if _, err := io.WriteString(w.stdin, header); err != nil {
		w.alive = false
		return nil, fmt.Errorf("write header: %w", err)
	}
	if _, err := w.stdin.Write(xmlInput); err != nil {
		w.alive = false
		return nil, fmt.Errorf("write xml payload: %w", err)
	}
	if _, err := w.stdin.Write(slotBytes); err != nil {
		w.alive = false
		return nil, fmt.Errorf("write slot: %w", err)
	}
	if _, err := w.stdin.Write(itemBytes); err != nil {
		w.alive = false
		return nil, fmt.Errorf("write item: %w", err)
	}

	// Read response header: "OK <len>\n" or "ERR <len>\n"
	respLine, err := w.stdout.ReadString('\n')
	if err != nil {
		w.alive = false
		return nil, fmt.Errorf("read response header: %w", err)
	}
	respLine = strings.TrimSpace(respLine)

	parts := strings.SplitN(respLine, " ", 2)
	if len(parts) != 2 {
		w.alive = false
		return nil, fmt.Errorf("invalid response header: %q", respLine)
	}

	respLen, err := strconv.Atoi(parts[1])
	if err != nil {
		w.alive = false
		return nil, fmt.Errorf("invalid response length: %q", parts[1])
	}

	body := make([]byte, respLen)
	if _, err := io.ReadFull(w.stdout, body); err != nil {
		w.alive = false
		return nil, fmt.Errorf("read response body: %w", err)
	}

	switch parts[0] {
	case "OK":
		return body, nil
	case "ERR":
		return nil, fmt.Errorf("worker error: %s", string(body))
	default:
		w.alive = false
		return nil, fmt.Errorf("unexpected response type: %q", parts[0])
	}
}

// GenerateWeights sends XML + slot + options to the worker and returns mod weight analysis JSON.
// Protocol: "GENERATE_WEIGHTS <xml_len> <slot_len> <opts_len>\n" + xml + slot + opts
func (w *Worker) GenerateWeights(xmlInput []byte, slot string, optionsJSON string) ([]byte, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.alive {
		return nil, fmt.Errorf("worker %d is dead", w.id)
	}

	slotBytes := []byte(slot)
	optsBytes := []byte(optionsJSON)

	header := fmt.Sprintf("GENERATE_WEIGHTS %d %d %d\n", len(xmlInput), len(slotBytes), len(optsBytes))
	if _, err := io.WriteString(w.stdin, header); err != nil {
		w.alive = false
		return nil, fmt.Errorf("write header: %w", err)
	}
	if _, err := w.stdin.Write(xmlInput); err != nil {
		w.alive = false
		return nil, fmt.Errorf("write xml payload: %w", err)
	}
	if _, err := w.stdin.Write(slotBytes); err != nil {
		w.alive = false
		return nil, fmt.Errorf("write slot: %w", err)
	}
	if _, err := w.stdin.Write(optsBytes); err != nil {
		w.alive = false
		return nil, fmt.Errorf("write options: %w", err)
	}

	// Read response header: "OK <len>\n" or "ERR <len>\n"
	respLine, err := w.stdout.ReadString('\n')
	if err != nil {
		w.alive = false
		return nil, fmt.Errorf("read response header: %w", err)
	}
	respLine = strings.TrimSpace(respLine)

	parts := strings.SplitN(respLine, " ", 2)
	if len(parts) != 2 {
		w.alive = false
		return nil, fmt.Errorf("invalid response header: %q", respLine)
	}

	respLen, err := strconv.Atoi(parts[1])
	if err != nil {
		w.alive = false
		return nil, fmt.Errorf("invalid response length: %q", parts[1])
	}

	body := make([]byte, respLen)
	if _, err := io.ReadFull(w.stdout, body); err != nil {
		w.alive = false
		return nil, fmt.Errorf("read response body: %w", err)
	}

	switch parts[0] {
	case "OK":
		return body, nil
	case "ERR":
		return nil, fmt.Errorf("worker error: %s", string(body))
	default:
		w.alive = false
		return nil, fmt.Errorf("unexpected response type: %q", parts[0])
	}
}

// FindBestAnoint sends XML + options to the worker and returns anoint ranking JSON.
// Protocol: "FIND_BEST_ANOINT <xml_len> <opts_len>\n" + xml + opts
func (w *Worker) FindBestAnoint(xmlInput []byte, optionsJSON string) ([]byte, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.alive {
		return nil, fmt.Errorf("worker %d is dead", w.id)
	}

	optsBytes := []byte(optionsJSON)

	header := fmt.Sprintf("FIND_BEST_ANOINT %d %d\n", len(xmlInput), len(optsBytes))
	if _, err := io.WriteString(w.stdin, header); err != nil {
		w.alive = false
		return nil, fmt.Errorf("write header: %w", err)
	}
	if _, err := w.stdin.Write(xmlInput); err != nil {
		w.alive = false
		return nil, fmt.Errorf("write xml payload: %w", err)
	}
	if _, err := w.stdin.Write(optsBytes); err != nil {
		w.alive = false
		return nil, fmt.Errorf("write options: %w", err)
	}

	// Read response header: "OK <len>\n" or "ERR <len>\n"
	respLine, err := w.stdout.ReadString('\n')
	if err != nil {
		w.alive = false
		return nil, fmt.Errorf("read response header: %w", err)
	}
	respLine = strings.TrimSpace(respLine)

	parts := strings.SplitN(respLine, " ", 2)
	if len(parts) != 2 {
		w.alive = false
		return nil, fmt.Errorf("invalid response header: %q", respLine)
	}

	respLen, err := strconv.Atoi(parts[1])
	if err != nil {
		w.alive = false
		return nil, fmt.Errorf("invalid response length: %q", parts[1])
	}

	body := make([]byte, respLen)
	if _, err := io.ReadFull(w.stdout, body); err != nil {
		w.alive = false
		return nil, fmt.Errorf("read response body: %w", err)
	}

	switch parts[0] {
	case "OK":
		return body, nil
	case "ERR":
		return nil, fmt.Errorf("worker error: %s", string(body))
	default:
		w.alive = false
		return nil, fmt.Errorf("unexpected response type: %q", parts[0])
	}
}

func (w *Worker) Shutdown() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.alive {
		io.WriteString(w.stdin, "QUIT\n")
		w.alive = false
	}
	// Give it a moment to exit gracefully, then kill
	done := make(chan struct{})
	go func() {
		w.cmd.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		w.cmd.Process.Kill()
	}
}

// ---------------------------------------------------------------------------
// Worker Pool
// ---------------------------------------------------------------------------

type WorkerPool struct {
	workers    chan *Worker
	allWorkers []*Worker
	mu         sync.Mutex

	// Config for spawning new workers
	srcDir       string
	luaPath      string
	luaCPath     string
	workerScript string
	poolSize     int
	nextID       int
}

func newWorkerPool(poolSize int, srcDir, luaPath, luaCPath, workerScript string) (*WorkerPool, error) {
	pool := &WorkerPool{
		workers:      make(chan *Worker, poolSize),
		srcDir:       srcDir,
		luaPath:      luaPath,
		luaCPath:     luaCPath,
		workerScript: workerScript,
		poolSize:     poolSize,
	}

	log.Printf("[pool] Starting %d worker(s)...", poolSize)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for i := 0; i < poolSize; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			w, err := newWorker(id, srcDir, luaPath, luaCPath, workerScript)
			if err != nil {
				log.Printf("[pool] Failed to start worker %d: %v", id, err)
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				return
			}
			mu.Lock()
			pool.allWorkers = append(pool.allWorkers, w)
			mu.Unlock()
			pool.workers <- w
		}(i)
	}

	wg.Wait()
	pool.nextID = poolSize

	if len(pool.allWorkers) == 0 {
		return nil, fmt.Errorf("no workers started: %w", firstErr)
	}

	log.Printf("[pool] %d/%d workers ready", len(pool.allWorkers), poolSize)
	return pool, nil
}

// Acquire gets a worker from the pool (blocks until one is available).
func (p *WorkerPool) Acquire(timeout time.Duration) (*Worker, error) {
	select {
	case w := <-p.workers:
		if w.alive {
			return w, nil
		}
		// Worker died — try to replace it
		log.Printf("[pool] Worker %d found dead, replacing...", w.id)
		return p.replaceWorker(w)
	case <-time.After(timeout):
		return nil, fmt.Errorf("no worker available within %v", timeout)
	}
}

// Release returns a worker to the pool.
func (p *WorkerPool) Release(w *Worker) {
	if !w.alive {
		// Replace dead worker in background
		go func() {
			replacement, err := p.replaceWorker(w)
			if err != nil {
				log.Printf("[pool] Failed to replace worker %d: %v", w.id, err)
				return
			}
			p.workers <- replacement
		}()
		return
	}
	p.workers <- w
}

func (p *WorkerPool) replaceWorker(_ *Worker) (*Worker, error) {
	p.mu.Lock()
	id := p.nextID
	p.nextID++
	p.mu.Unlock()

	log.Printf("[pool] Spawning replacement worker %d", id)
	w, err := newWorker(id, p.srcDir, p.luaPath, p.luaCPath, p.workerScript)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	p.allWorkers = append(p.allWorkers, w)
	p.mu.Unlock()

	return w, nil
}

func (p *WorkerPool) Shutdown() {
	log.Println("[pool] Shutting down all workers...")
	for _, w := range p.allWorkers {
		w.Shutdown()
	}
}

// ---------------------------------------------------------------------------
// HTTP Server
// ---------------------------------------------------------------------------

type Server struct {
	pool   *WorkerPool
	pricer *pricer.BuildCostCalculator
}

func (s *Server) handleRecalc(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed. Use POST.", http.StatusMethodNotAllowed)
		return
	}

	// Read POB code from request body
	body, err := io.ReadAll(io.LimitReader(r.Body, 1*1024*1024)) // 1MB limit
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	pobCode := strings.TrimSpace(string(body))
	if len(pobCode) == 0 {
		http.Error(w, "Empty request body. Send a POB code as the POST body.", http.StatusBadRequest)
		return
	}

	start := time.Now()

	// Step 1: Decode POB code → XML
	xmlInput, err := decodePOBCode(pobCode)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode POB code: %v", err), http.StatusBadRequest)
		return
	}

	decodeTime := time.Since(start)
	log.Printf("[http] Decoded POB code: %d bytes compressed → %d bytes XML (%.1fms)",
		len(pobCode), len(xmlInput), float64(decodeTime.Microseconds())/1000)

	// Step 2: Acquire a worker
	worker, err := s.pool.Acquire(30 * time.Second)
	if err != nil {
		http.Error(w, fmt.Sprintf("No available worker: %v", err), http.StatusServiceUnavailable)
		return
	}
	defer s.pool.Release(worker)

	// Step 3: Recalculate
	recalcStart := time.Now()
	xmlOutput, err := worker.Recalc(xmlInput)
	if err != nil {
		http.Error(w, fmt.Sprintf("Recalculation failed: %v", err), http.StatusInternalServerError)
		return
	}

	recalcTime := time.Since(recalcStart)
	totalTime := time.Since(start)
	log.Printf("[http] Recalculation complete: %d → %d bytes (decode=%.1fms, recalc=%.1fms, total=%.1fms)",
		len(xmlInput), len(xmlOutput),
		float64(decodeTime.Microseconds())/1000,
		float64(recalcTime.Microseconds())/1000,
		float64(totalTime.Microseconds())/1000)

	// Step 4: Return result
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("X-Decode-Time-Ms", fmt.Sprintf("%.1f", float64(decodeTime.Microseconds())/1000))
	w.Header().Set("X-Recalc-Time-Ms", fmt.Sprintf("%.1f", float64(recalcTime.Microseconds())/1000))
	w.Header().Set("X-Total-Time-Ms", fmt.Sprintf("%.1f", float64(totalTime.Microseconds())/1000))
	w.WriteHeader(http.StatusOK)
	w.Write(xmlOutput)
}

func (s *Server) handleReplaceItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed. Use POST.", http.StatusMethodNotAllowed)
		return
	}

	// Parse JSON request body
	var req struct {
		PobCode  string `json:"pob_code"`
		Slot     string `json:"slot"`
		ItemText string `json:"item_text"`
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 2*1024*1024)) // 2MB limit
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	if req.PobCode == "" {
		http.Error(w, "Missing required field: pob_code", http.StatusBadRequest)
		return
	}
	if req.Slot == "" {
		http.Error(w, "Missing required field: slot (e.g. \"Helmet\", \"Body Armour\", \"Weapon 1\", \"Ring 1\")", http.StatusBadRequest)
		return
	}
	if req.ItemText == "" {
		http.Error(w, "Missing required field: item_text (POB item text format)", http.StatusBadRequest)
		return
	}

	start := time.Now()

	// Step 1: Decode POB code → XML
	xmlInput, err := decodePOBCode(req.PobCode)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode POB code: %v", err), http.StatusBadRequest)
		return
	}

	decodeTime := time.Since(start)
	log.Printf("[http] /replace-item: Decoded POB code: %d bytes → %d bytes XML (%.1fms), slot=%q",
		len(req.PobCode), len(xmlInput), float64(decodeTime.Microseconds())/1000, req.Slot)

	// Step 2: Acquire a worker
	worker, err := s.pool.Acquire(30 * time.Second)
	if err != nil {
		http.Error(w, fmt.Sprintf("No available worker: %v", err), http.StatusServiceUnavailable)
		return
	}
	defer s.pool.Release(worker)

	// Step 3: Send REPLACE_ITEM command
	recalcStart := time.Now()
	result, err := worker.ReplaceItem(xmlInput, req.Slot, req.ItemText)
	if err != nil {
		http.Error(w, fmt.Sprintf("Replace item failed: %v", err), http.StatusInternalServerError)
		return
	}

	recalcTime := time.Since(recalcStart)
	totalTime := time.Since(start)
	log.Printf("[http] /replace-item: Complete (decode=%.1fms, recalc=%.1fms, total=%.1fms)",
		float64(decodeTime.Microseconds())/1000,
		float64(recalcTime.Microseconds())/1000,
		float64(totalTime.Microseconds())/1000)

	// Step 4: Return JSON result
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Decode-Time-Ms", fmt.Sprintf("%.1f", float64(decodeTime.Microseconds())/1000))
	w.Header().Set("X-Recalc-Time-Ms", fmt.Sprintf("%.1f", float64(recalcTime.Microseconds())/1000))
	w.Header().Set("X-Total-Time-Ms", fmt.Sprintf("%.1f", float64(totalTime.Microseconds())/1000))
	w.WriteHeader(http.StatusOK)
	w.Write(result)
}

func (s *Server) handleGenerateWeights(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed. Use POST.", http.StatusMethodNotAllowed)
		return
	}

	// Parse JSON request body
	var req struct {
		PobCode     string `json:"pob_code"`
		Slot        string `json:"slot"`
		StatWeights []struct {
			Stat       string  `json:"stat"`
			WeightMult float64 `json:"weightMult"`
		} `json:"stat_weights"`
		IncludeCorrupted bool `json:"include_corrupted"`
		IncludeEldritch  bool `json:"include_eldritch"`
		IncludeScourge   bool `json:"include_scourge"`
		IncludeSynthesis bool `json:"include_synthesis"`
		IncludeTalisman  bool `json:"include_talisman"`
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 2*1024*1024)) // 2MB limit
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	if req.PobCode == "" {
		http.Error(w, "Missing required field: pob_code", http.StatusBadRequest)
		return
	}
	if req.Slot == "" {
		http.Error(w, "Missing required field: slot (e.g. \"Helmet\", \"Body Armour\", \"Weapon 1\", \"Ring 1\")", http.StatusBadRequest)
		return
	}

	start := time.Now()

	// Step 1: Decode POB code → XML
	xmlInput, err := decodePOBCode(req.PobCode)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode POB code: %v", err), http.StatusBadRequest)
		return
	}

	decodeTime := time.Since(start)
	log.Printf("[http] /generate-weights: Decoded POB code: %d bytes → %d bytes XML (%.1fms), slot=%q",
		len(req.PobCode), len(xmlInput), float64(decodeTime.Microseconds())/1000, req.Slot)

	// Step 2: Build options JSON to pass to worker
	optionsJSON, _ := json.Marshal(map[string]interface{}{
		"stat_weights":      req.StatWeights,
		"include_corrupted": req.IncludeCorrupted,
		"include_eldritch":  req.IncludeEldritch,
		"include_scourge":   req.IncludeScourge,
		"include_synthesis": req.IncludeSynthesis,
		"include_talisman":  req.IncludeTalisman,
	})

	// Step 3: Acquire a worker (longer timeout for weight calculation)
	worker, err := s.pool.Acquire(60 * time.Second)
	if err != nil {
		http.Error(w, fmt.Sprintf("No available worker: %v", err), http.StatusServiceUnavailable)
		return
	}
	defer s.pool.Release(worker)

	// Step 4: Send GENERATE_WEIGHTS command
	calcStart := time.Now()
	result, err := worker.GenerateWeights(xmlInput, req.Slot, string(optionsJSON))
	if err != nil {
		http.Error(w, fmt.Sprintf("Generate weights failed: %v", err), http.StatusInternalServerError)
		return
	}

	calcTime := time.Since(calcStart)
	totalTime := time.Since(start)
	log.Printf("[http] /generate-weights: Complete (decode=%.1fms, calc=%.1fms, total=%.1fms)",
		float64(decodeTime.Microseconds())/1000,
		float64(calcTime.Microseconds())/1000,
		float64(totalTime.Microseconds())/1000)

	// Step 5: Return JSON result
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Decode-Time-Ms", fmt.Sprintf("%.1f", float64(decodeTime.Microseconds())/1000))
	w.Header().Set("X-Calc-Time-Ms", fmt.Sprintf("%.1f", float64(calcTime.Microseconds())/1000))
	w.Header().Set("X-Total-Time-Ms", fmt.Sprintf("%.1f", float64(totalTime.Microseconds())/1000))
	w.WriteHeader(http.StatusOK)
	w.Write(result)
}

func (s *Server) handleFindBestAnoint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed. Use POST.", http.StatusMethodNotAllowed)
		return
	}

	// Parse JSON request body
	var req struct {
		PobCode    string `json:"pob_code"`
		Stat       string `json:"stat"`
		MaxResults int    `json:"max_results"`
		Search     string `json:"search"`
		SlotName   string `json:"slot_name"`
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 2*1024*1024)) // 2MB limit
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	if req.PobCode == "" {
		http.Error(w, "Missing required field: pob_code", http.StatusBadRequest)
		return
	}

	// Defaults
	if req.Stat == "" {
		req.Stat = "CombinedDPS"
	}
	if req.MaxResults <= 0 {
		req.MaxResults = 30
	}
	if req.SlotName == "" {
		req.SlotName = "Amulet"
	}

	start := time.Now()

	// Step 1: Decode POB code → XML
	xmlInput, err := decodePOBCode(req.PobCode)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode POB code: %v", err), http.StatusBadRequest)
		return
	}

	decodeTime := time.Since(start)
	log.Printf("[http] /find-best-anoint: Decoded POB code: %d bytes → %d bytes XML (%.1fms), stat=%q",
		len(req.PobCode), len(xmlInput), float64(decodeTime.Microseconds())/1000, req.Stat)

	// Step 2: Build options JSON
	optionsJSON, _ := json.Marshal(map[string]interface{}{
		"stat":        req.Stat,
		"max_results": req.MaxResults,
		"search":      req.Search,
		"slot_name":   req.SlotName,
	})

	// Step 3: Acquire a worker (longer timeout for anoint calculation — can be slow)
	worker, err := s.pool.Acquire(60 * time.Second)
	if err != nil {
		http.Error(w, fmt.Sprintf("No available worker: %v", err), http.StatusServiceUnavailable)
		return
	}
	defer s.pool.Release(worker)

	// Step 4: Send FIND_BEST_ANOINT command
	calcStart := time.Now()
	result, err := worker.FindBestAnoint(xmlInput, string(optionsJSON))
	if err != nil {
		http.Error(w, fmt.Sprintf("Find best anoint failed: %v", err), http.StatusInternalServerError)
		return
	}

	calcTime := time.Since(calcStart)
	totalTime := time.Since(start)
	log.Printf("[http] /find-best-anoint: Complete (decode=%.1fms, calc=%.1fms, total=%.1fms)",
		float64(decodeTime.Microseconds())/1000,
		float64(calcTime.Microseconds())/1000,
		float64(totalTime.Microseconds())/1000)

	// Step 5: Return JSON result
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Decode-Time-Ms", fmt.Sprintf("%.1f", float64(decodeTime.Microseconds())/1000))
	w.Header().Set("X-Calc-Time-Ms", fmt.Sprintf("%.1f", float64(calcTime.Microseconds())/1000))
	w.Header().Set("X-Total-Time-Ms", fmt.Sprintf("%.1f", float64(totalTime.Microseconds())/1000))
	w.WriteHeader(http.StatusOK)
	w.Write(result)
}

func (s *Server) handleTranslateItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed. Use POST.", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 2*1024*1024)) // 2MB limit
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if len(body) == 0 {
		http.Error(w, "Empty request body. Send a single item JSON object from the trade API.", http.StatusBadRequest)
		return
	}

	start := time.Now()

	result, err := translator.TranslateItem(body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Translation failed: %v", err), http.StatusBadRequest)
		return
	}

	translateTime := time.Since(start)
	log.Printf("[http] /translate-item: %d bytes JSON → slot=%q, %d bytes item text (%.1fms)",
		len(body), result.Slot, len(result.ItemText), float64(translateTime.Microseconds())/1000)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Translate-Time-Ms", fmt.Sprintf("%.1f", float64(translateTime.Microseconds())/1000))
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

func (s *Server) handleConvertItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed. Use POST.", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 2*1024*1024)) // 2MB limit
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if len(body) == 0 {
		http.Error(w, "Empty request body. Send a single item JSON object from the trade API.", http.StatusBadRequest)
		return
	}

	start := time.Now()

	result, err := translator.ConvertItem(body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Conversion failed: %v", err), http.StatusBadRequest)
		return
	}

	convertTime := time.Since(start)
	log.Printf("[http] /convert-item: %d bytes JSON → slot=%q, %d bytes item text (%.1fms)",
		len(body), result.Slot, len(result.ItemText), float64(convertTime.Microseconds())/1000)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Convert-Time-Ms", fmt.Sprintf("%.1f", float64(convertTime.Microseconds())/1000))
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

func (s *Server) handleTranslate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed. Use POST.", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 5*1024*1024)) // 5MB limit
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if len(body) == 0 {
		http.Error(w, "Empty request body. Send JSON with 'items' and 'passiveSkills' fields.", http.StatusBadRequest)
		return
	}

	start := time.Now()

	xmlStr, err := translator.TranslateItemsJSON(body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Translation failed: %v", err), http.StatusBadRequest)
		return
	}

	translateTime := time.Since(start)
	log.Printf("[http] /translate: %d bytes JSON → %d bytes XML (%.1fms)",
		len(body), len(xmlStr), float64(translateTime.Microseconds())/1000)

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("X-Translate-Time-Ms", fmt.Sprintf("%.1f", float64(translateTime.Microseconds())/1000))
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(xmlStr))
}

func (s *Server) handleTranslateAndRecalc(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed. Use POST.", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 5*1024*1024)) // 5MB limit
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if len(body) == 0 {
		http.Error(w, "Empty request body. Send JSON with 'items' and 'passiveSkills' fields.", http.StatusBadRequest)
		return
	}

	start := time.Now()

	// Step 1: Translate Chinese → English POB XML
	xmlStr, err := translator.TranslateItemsJSON(body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Translation failed: %v", err), http.StatusBadRequest)
		return
	}

	translateTime := time.Since(start)
	xmlInput := []byte(xmlStr)
	log.Printf("[http] /translate-and-recalc: Translated %d bytes JSON → %d bytes XML (%.1fms)",
		len(body), len(xmlInput), float64(translateTime.Microseconds())/1000)

	// Step 2: Acquire a worker for recalculation
	worker, err := s.pool.Acquire(30 * time.Second)
	if err != nil {
		http.Error(w, fmt.Sprintf("No available worker: %v", err), http.StatusServiceUnavailable)
		return
	}
	defer s.pool.Release(worker)

	// Step 3: Recalculate
	recalcStart := time.Now()
	xmlOutput, err := worker.Recalc(xmlInput)
	if err != nil {
		http.Error(w, fmt.Sprintf("Recalculation failed: %v", err), http.StatusInternalServerError)
		return
	}

	recalcTime := time.Since(recalcStart)
	totalTime := time.Since(start)
	log.Printf("[http] /translate-and-recalc: Complete (translate=%.1fms, recalc=%.1fms, total=%.1fms)",
		float64(translateTime.Microseconds())/1000,
		float64(recalcTime.Microseconds())/1000,
		float64(totalTime.Microseconds())/1000)

	// Step 4: Return recalculated XML
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("X-Translate-Time-Ms", fmt.Sprintf("%.1f", float64(translateTime.Microseconds())/1000))
	w.Header().Set("X-Recalc-Time-Ms", fmt.Sprintf("%.1f", float64(recalcTime.Microseconds())/1000))
	w.Header().Set("X-Total-Time-Ms", fmt.Sprintf("%.1f", float64(totalTime.Microseconds())/1000))
	w.WriteHeader(http.StatusOK)
	w.Write(xmlOutput)
}

func (s *Server) handleBuildCost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed. Use POST.", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 2*1024*1024))
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req pricer.BuildCostRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	if req.PobCode == "" {
		http.Error(w, "Missing required field: pob_code", http.StatusBadRequest)
		return
	}
	if req.POESESSID == "" {
		http.Error(w, "Missing required field: poesessid (needed for CN trade API)", http.StatusBadRequest)
		return
	}

	start := time.Now()
	log.Printf("[http] /build-cost: Decoding POB code (%d chars)", len(req.PobCode))

	// Decode POB code → XML
	xmlData, err := decodePOBCode(req.PobCode)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode POB code: %v", err), http.StatusBadRequest)
		return
	}

	decodeTime := time.Since(start)
	log.Printf("[http] /build-cost: Decoded POB code → %d bytes XML (%.1fms)",
		len(xmlData), float64(decodeTime.Microseconds())/1000)

	result, err := s.pricer.Calculate(&req, xmlData)
	if err != nil {
		http.Error(w, fmt.Sprintf("Build cost calculation failed: %v", err), http.StatusInternalServerError)
		return
	}

	totalTime := time.Since(start)
	log.Printf("[http] /build-cost: Complete — %d items, %d gems, total=%.0f chaos (%.1f divine), took %.1fs",
		len(result.Items), len(result.Gems), result.TotalChaos, result.TotalDivine, totalTime.Seconds())

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Total-Time-Ms", fmt.Sprintf("%.1f", float64(totalTime.Microseconds())/1000))
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "ok\nworkers_available: %d\n", len(s.pool.workers))
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	// Read POB version info embedded at build time
	data, err := os.ReadFile("pob_version")
	if err != nil {
		fmt.Fprintf(w, "pob_version=unknown (file not found)\n")
		return
	}
	w.Write(data)
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

func main() {
	// Configuration from environment variables with sensible defaults
	listenAddr := envOrDefault("POB_LISTEN", ":8080")
	poolSizeStr := envOrDefault("POB_POOL_SIZE", "2")
	srcDir := envOrDefault("POB_SRC_DIR", "")
	luaPath := envOrDefault("LUA_PATH", "../runtime/lua/?.lua;../runtime/lua/?/init.lua;;")
	luaCPath := envOrDefault("LUA_CPATH", "")
	workerScript := envOrDefault("POB_WORKER_SCRIPT", "")

	poolSize, err := strconv.Atoi(poolSizeStr)
	if err != nil || poolSize < 1 {
		poolSize = 2
	}

	// Auto-detect paths if not explicitly set
	if srcDir == "" {
		// Try to find src/ relative to the binary
		if _, err := os.Stat("src/HeadlessWrapper.lua"); err == nil {
			srcDir = "src"
		} else if _, err := os.Stat("../src/HeadlessWrapper.lua"); err == nil {
			srcDir = "../src"
		} else {
			log.Fatal("Cannot find src/HeadlessWrapper.lua. Set POB_SRC_DIR environment variable.")
		}
	}

	if workerScript == "" {
		// Look for worker.lua relative to src/
		candidates := []string{
			"../tools/pob_api/worker.lua",
			"tools/pob_api/worker.lua",
		}
		for _, c := range candidates {
			// Check relative to srcDir
			fullPath := srcDir + "/" + c
			if _, err := os.Stat(fullPath); err == nil {
				workerScript = c
				break
			}
		}
		if workerScript == "" {
			log.Fatal("Cannot find worker.lua. Set POB_WORKER_SCRIPT environment variable.")
		}
	}

	log.Printf("POB Recalc API Server")
	log.Printf("  Listen:        %s", listenAddr)
	log.Printf("  Pool size:     %d", poolSize)
	log.Printf("  Src dir:       %s", srcDir)
	log.Printf("  Worker script: %s", workerScript)
	log.Printf("  LUA_PATH:      %s", luaPath)
	log.Printf("  LUA_CPATH:     %s", luaCPath)

	// Print POB version info if available
	if versionData, err := os.ReadFile("pob_version"); err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(versionData)), "\n") {
			log.Printf("  [POB] %s", line)
		}
	}

	// Initialize translator (loads embedded translation data)
	log.Println("Initializing Chinese→English translator...")
	if err := translator.Init(); err != nil {
		log.Fatalf("Failed to initialize translator: %v", err)
	}
	log.Println("Translator initialized successfully")

	// Start worker pool
	pool, err := newWorkerPool(poolSize, srcDir, luaPath, luaCPath, workerScript)
	if err != nil {
		log.Fatalf("Failed to start worker pool: %v", err)
	}

	// Create server
	buildCostCalc, err := pricer.NewBuildCostCalculator()
	if err != nil {
		log.Fatalf("Failed to initialize build cost calculator: %v", err)
	}

	srv := &Server{pool: pool, pricer: buildCostCalc}

	mux := http.NewServeMux()
	mux.HandleFunc("/recalc", srv.handleRecalc)
	mux.HandleFunc("/replace-item", srv.handleReplaceItem)
	mux.HandleFunc("/generate-weights", srv.handleGenerateWeights)
	mux.HandleFunc("/find-best-anoint", srv.handleFindBestAnoint)
	mux.HandleFunc("/translate-item", srv.handleTranslateItem)
	mux.HandleFunc("/convert-item", srv.handleConvertItem)
	mux.HandleFunc("/translate", srv.handleTranslate)
	mux.HandleFunc("/translate-and-recalc", srv.handleTranslateAndRecalc)
	mux.HandleFunc("/build-cost", srv.handleBuildCost)
	mux.HandleFunc("/health", srv.handleHealth)
	mux.HandleFunc("/version", srv.handleVersion)

	httpSrv := &http.Server{
		Addr:         listenAddr,
		Handler:      mux,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 300 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		log.Printf("Received %v, shutting down...", sig)
		httpSrv.Close()
		pool.Shutdown()
		os.Exit(0)
	}()

	log.Printf("Server listening on %s", listenAddr)
	log.Printf("  POST /recalc              — Send POB code, get recalculated XML")
	log.Printf("  POST /replace-item        — Replace an item and compare DPS")
	log.Printf("  POST /generate-weights    — Generate stat weights for a slot")
	log.Printf("  POST /find-best-anoint    — Find best anoint for amulet")
	log.Printf("  POST /translate-item       — Translate single CN item JSON → EN POB text + slot")
	log.Printf("  POST /translate           — Translate CN items JSON → EN POB XML")
	log.Printf("  POST /translate-and-recalc — Translate + recalculate")
	log.Printf("  POST /build-cost          — Calculate build cost from poe.ninja URL")
	log.Printf("  GET  /health              — Health check")
	log.Printf("  GET  /version             — Show POB source version info")

	if err := httpSrv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("HTTP server error: %v", err)
	}
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
