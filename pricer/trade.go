package pricer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	cnTradeBaseURL   = "https://poe.game.qq.com/api/trade"
	tradeRequestGap  = 6 * time.Second // CN trade API rate limit: ~8-10 req/min
	maxFetchPerBatch = 10
	defaultCNLeague  = "S29赛季"
)

// TradeClient handles interaction with the CN Trade API.
type TradeClient struct {
	client    *http.Client
	mu        sync.Mutex
	lastReq   time.Time
	poesessid string // POESESSID cookie value
}

// NewTradeClient creates a new trade API client with the given POESESSID.
func NewTradeClient(poesessid string) *TradeClient {
	return &TradeClient{
		client: &http.Client{
			Timeout: 20 * time.Second,
		},
		poesessid: poesessid,
	}
}

const userAgent = "POB-Recalc-API/1.0 (Build Cost Calculator)"

// rateLimit ensures we don't exceed the trade API rate limit.
func (tc *TradeClient) rateLimit() {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	elapsed := time.Since(tc.lastReq)
	if elapsed < tradeRequestGap {
		time.Sleep(tradeRequestGap - elapsed)
	}
	tc.lastReq = time.Now()
}

// addAuth adds POESESSID cookie and common headers to a request.
func (tc *TradeClient) addAuth(req *http.Request) {
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/json")
	if tc.poesessid != "" {
		req.AddCookie(&http.Cookie{
			Name:  "POESESSID",
			Value: tc.poesessid,
		})
	}
}

// SearchAndPrice searches for an item on CN trade and returns the median price.
// Returns (priceChaos, originalCurrency, originalPrice, confidence, error).
func (tc *TradeClient) SearchAndPrice(league string, searchReq *TradeSearchRequest) (float64, string, float64, string, error) {
	tc.rateLimit()

	// Step 1: Search
	searchURL := fmt.Sprintf("%s/search/%s", cnTradeBaseURL, league)
	reqBody, err := json.Marshal(searchReq)
	if err != nil {
		return 0, "", 0, "none", fmt.Errorf("marshal search request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", searchURL, bytes.NewReader(reqBody))
	if err != nil {
		return 0, "", 0, "none", fmt.Errorf("create search request: %w", err)
	}
	tc.addAuth(httpReq)

	resp, err := tc.client.Do(httpReq)
	if err != nil {
		return 0, "", 0, "none", fmt.Errorf("trade search: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1*1024*1024))
	if err != nil {
		return 0, "", 0, "none", fmt.Errorf("read search response: %w", err)
	}

	if resp.StatusCode != 200 {
		return 0, "", 0, "none", fmt.Errorf("trade search returned %d: %s", resp.StatusCode, truncate(string(body), 200))
	}

	var searchResp TradeSearchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return 0, "", 0, "none", fmt.Errorf("parse search response: %w", err)
	}

	if searchResp.Total == 0 || len(searchResp.Result) == 0 {
		return 0, "", 0, "none", nil
	}

	// Step 2: Fetch top results (up to 10 cheapest)
	fetchCount := len(searchResp.Result)
	if fetchCount > maxFetchPerBatch {
		fetchCount = maxFetchPerBatch
	}

	tc.rateLimit()

	ids := strings.Join(searchResp.Result[:fetchCount], ",")
	fetchURL := fmt.Sprintf("%s/fetch/%s?query=%s", cnTradeBaseURL, ids, searchResp.ID)

	httpReq, err = http.NewRequest("GET", fetchURL, nil)
	if err != nil {
		return 0, "", 0, "none", fmt.Errorf("create fetch request: %w", err)
	}
	tc.addAuth(httpReq)

	resp, err = tc.client.Do(httpReq)
	if err != nil {
		return 0, "", 0, "none", fmt.Errorf("trade fetch: %w", err)
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return 0, "", 0, "none", fmt.Errorf("read fetch response: %w", err)
	}

	if resp.StatusCode != 200 {
		return 0, "", 0, "none", fmt.Errorf("trade fetch returned %d: %s", resp.StatusCode, truncate(string(body), 200))
	}

	var fetchResp TradeFetchResponse
	if err := json.Unmarshal(body, &fetchResp); err != nil {
		return 0, "", 0, "none", fmt.Errorf("parse fetch response: %w", err)
	}

	if len(fetchResp.Result) == 0 {
		return 0, "", 0, "none", nil
	}

	return computeMedianPrice(fetchResp.Result, searchResp.Total)
}

// FetchDivineRate queries the CN trade for the current divine/chaos exchange rate.
func (tc *TradeClient) FetchDivineRate(league string) (float64, error) {
	searchReq := &TradeSearchRequest{
		Query: TradeQuery{
			Status: TradeStatus{Option: "any"},
			Type:   "崇高石",
		},
		Sort: TradeSort{Price: "asc"},
	}

	tc.rateLimit()

	searchURL := fmt.Sprintf("%s/search/%s", cnTradeBaseURL, league)
	reqBody, _ := json.Marshal(searchReq)

	httpReq, err := http.NewRequest("POST", searchURL, bytes.NewReader(reqBody))
	if err != nil {
		return 0, err
	}
	tc.addAuth(httpReq)

	resp, err := tc.client.Do(httpReq)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1*1024*1024))
	if resp.StatusCode != 200 {
		log.Printf("[pricer] Failed to fetch divine rate (status %d), using default 150", resp.StatusCode)
		return 150, nil
	}

	var searchResp TradeSearchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return 150, nil
	}

	if len(searchResp.Result) == 0 {
		return 150, nil
	}

	fetchCount := 5
	if fetchCount > len(searchResp.Result) {
		fetchCount = len(searchResp.Result)
	}

	tc.rateLimit()

	ids := strings.Join(searchResp.Result[:fetchCount], ",")
	fetchURL := fmt.Sprintf("%s/fetch/%s?query=%s", cnTradeBaseURL, ids, searchResp.ID)

	httpReq, _ = http.NewRequest("GET", fetchURL, nil)
	tc.addAuth(httpReq)

	resp, err = tc.client.Do(httpReq)
	if err != nil {
		return 150, nil
	}
	defer resp.Body.Close()

	body, _ = io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if resp.StatusCode != 200 {
		return 150, nil
	}

	var fetchResp TradeFetchResponse
	if err := json.Unmarshal(body, &fetchResp); err != nil {
		return 150, nil
	}

	var chaosPrices []float64
	for _, r := range fetchResp.Result {
		curr := r.Listing.Price.Currency
		if curr == "chaos" || curr == "混沌石" {
			chaosPrices = append(chaosPrices, r.Listing.Price.Amount)
		}
	}

	if len(chaosPrices) > 0 {
		sort.Float64s(chaosPrices)
		return chaosPrices[len(chaosPrices)/2], nil
	}

	return 150, nil
}

// computeMedianPrice computes median from fetched results.
func computeMedianPrice(results []TradeFetchResult, totalListings int) (float64, string, float64, string, error) {
	type priceEntry struct {
		amount   float64
		currency string
	}

	var prices []priceEntry
	for _, r := range results {
		p := r.Listing.Price
		if p.Amount > 0 {
			prices = append(prices, priceEntry{
				amount:   p.Amount,
				currency: p.Currency,
			})
		}
	}

	if len(prices) == 0 {
		return 0, "", 0, "none", nil
	}

	sort.Slice(prices, func(i, j int) bool {
		return prices[i].amount < prices[j].amount
	})

	median := prices[len(prices)/2]

	confidence := "low"
	if totalListings >= 10 {
		confidence = "high"
	} else if totalListings >= 3 {
		confidence = "medium"
	}

	return median.amount, median.currency, median.amount, confidence, nil
}

// CurrencyToChaos converts a price in a given currency to chaos orbs.
func CurrencyToChaos(amount float64, currency string, divineRate float64) float64 {
	switch currency {
	case "chaos", "混沌石", "Chaos Orb":
		return amount
	case "divine", "神圣石", "Divine Orb":
		return amount * divineRate
	case "exalted", "崇高石", "Exalted Orb":
		return amount * divineRate * 0.5
	case "alchemy", "点金石", "Orb of Alchemy":
		return amount * 0.3
	case "fusing", "链结石", "Orb of Fusing":
		return amount * 0.5
	default:
		if amount > 0 {
			return amount
		}
		return 0
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func round2(f float64) float64 {
	return math.Round(f*100) / 100
}
