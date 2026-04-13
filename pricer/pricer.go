package pricer

import (
	"fmt"
	"log"
	"strings"

	"pob_api/translator"

	"github.com/cn-poe-community/cn-poe-utils/go/data/poe"
)

// BuildCostCalculator calculates the total cost of a build from a POB code.
type BuildCostCalculator struct {
	// en→zh reverse maps (built once at init, shared across requests)
	uniqueEnToZh   map[string]string // "Hyrri's Truth" -> "海日真言"
	baseTypeEnToZh map[string]string // "Jade Amulet" -> "翡翠护身符"
	gemEnToZh      map[string]string // "Tornado Shot" -> "龙卷射击"
}

// NewBuildCostCalculator creates a new calculator.
// translator.Init() must have been called before this.
func NewBuildCostCalculator() (*BuildCostCalculator, error) {
	data := translator.GetPoeData()
	if data == nil {
		return nil, fmt.Errorf("translator not initialized, call translator.Init() first")
	}

	calc := &BuildCostCalculator{
		uniqueEnToZh:   make(map[string]string),
		baseTypeEnToZh: make(map[string]string),
		gemEnToZh:      make(map[string]string),
	}

	calc.buildMaps(data)
	return calc, nil
}

// buildMaps constructs en→zh reverse lookup maps from the poe.Data.
func (c *BuildCostCalculator) buildMaps(data *poe.Data) {
	allBaseTypes := [][]poe.BaseType{
		data.Amulets, data.Belts, data.BodyArmours, data.Boots,
		data.Flasks, data.Gloves, data.Helmets, data.Jewels,
		data.Quivers, data.Rings, data.Shields, data.Weapons,
		data.Tinctures,
	}

	for _, group := range allBaseTypes {
		for _, bt := range group {
			if bt.En != "" && bt.Zh != "" {
				c.baseTypeEnToZh[bt.En] = bt.Zh
			}
			for _, u := range bt.Uniques {
				if u.En != "" && u.Zh != "" {
					c.uniqueEnToZh[u.En] = u.Zh
				}
			}
		}
	}

	allSkills := [][]poe.Skill{
		data.GemSkills, data.HybridSkills, data.IndexableSupports, data.TransfiguredSkills,
	}
	for _, group := range allSkills {
		for _, s := range group {
			if s.En != "" && s.Zh != "" {
				c.gemEnToZh[s.En] = s.Zh
			}
		}
	}

	log.Printf("[pricer] Built reverse maps: %d uniques, %d base types, %d gems",
		len(c.uniqueEnToZh), len(c.baseTypeEnToZh), len(c.gemEnToZh))
}

// Calculate performs the full build cost calculation.
// It decodes the POB code, parses XML, and queries CN trade for prices.
func (c *BuildCostCalculator) Calculate(req *BuildCostRequest, xmlData []byte) (*BuildCostResponse, error) {
	cnLeague := req.CNLeague
	if cnLeague == "" {
		cnLeague = defaultCNLeague
	}

	// Step 1: Parse POB XML
	items, gems, buildInfo, err := ParsePOBXML(xmlData)
	if err != nil {
		return nil, fmt.Errorf("parse POB XML: %w", err)
	}

	gems = DeduplicateGems(gems)

	resp := &BuildCostResponse{
		CNLeague: cnLeague,
	}
	if buildInfo != nil {
		resp.Level = buildInfo.Level
		resp.AscClass = buildInfo.AscendClass
		resp.Character = buildInfo.ClassName
	}

	if len(items) == 0 && len(gems) == 0 {
		return nil, fmt.Errorf("no items or gems found in POB code")
	}

	log.Printf("[pricer] Found %d items and %d unique gems", len(items), len(gems))

	// Step 2: Create trade client with POESESSID
	trade := NewTradeClient(req.POESESSID)

	// Step 3: Fetch divine rate
	divineRate, err := trade.FetchDivineRate(cnLeague)
	if err != nil {
		log.Printf("[pricer] Warning: failed to fetch divine rate: %v, using default", err)
		divineRate = 150
	}
	resp.DivineRate = divineRate
	log.Printf("[pricer] Divine rate: %.0f chaos", divineRate)

	// Step 4: Price each item
	var totalChaos float64

	for _, item := range items {
		result := c.priceEquipment(item, trade, cnLeague, divineRate)
		resp.Items = append(resp.Items, result)
		totalChaos += result.PriceChaos
	}

	// Step 5: Price each gem
	for _, gem := range gems {
		result := c.priceGem(gem, trade, cnLeague, divineRate)
		resp.Gems = append(resp.Gems, result)
		totalChaos += result.PriceChaos
	}

	resp.TotalChaos = round2(totalChaos)
	if divineRate > 0 {
		resp.TotalDivine = round2(totalChaos / divineRate)
	}

	return resp, nil
}

// priceEquipment prices a single equipment item from POB XML.
func (c *BuildCostCalculator) priceEquipment(item POBItem, trade *TradeClient, cnLeague string, divineRate float64) ItemPriceResult {
	result := ItemPriceResult{
		Name:     item.Name,
		BaseType: item.BaseType,
		Slot:     item.Slot,
		Rarity:   item.Rarity,
	}

	// Set display name
	if result.Name == "" {
		result.Name = item.BaseType
	}

	// Translate names — strip mutation prefixes like "Foulborn", "Blightborn" etc.
	if item.Name != "" {
		if zh, ok := c.uniqueEnToZh[item.Name]; ok {
			result.NameZh = zh
		} else {
			// Try stripping common mutation/corruption prefixes
			stripped := stripMutationPrefix(item.Name)
			if stripped != item.Name {
				if zh, ok := c.uniqueEnToZh[stripped]; ok {
					result.NameZh = zh
				}
			}
		}
	}
	if item.BaseType != "" {
		bt := item.BaseType
		if zh, ok := c.baseTypeEnToZh[bt]; ok {
			result.BaseTypeZh = zh
		}
	}

	rarity := strings.ToUpper(item.Rarity)
	switch rarity {
	case "UNIQUE":
		c.priceUniqueItem(&result, item, trade, cnLeague, divineRate)
	case "RARE":
		result.Confidence = "none"
		result.Error = "rare items cannot be auto-priced"
	case "MAGIC":
		// Try to price magic flasks by base type
		if isFlaskSlot(item.Slot) || isFlaskBaseType(item.BaseType) {
			result.Confidence = "none"
			result.Error = "magic flask, skipped"
		} else {
			result.Confidence = "none"
			result.Error = "magic item, skipped"
		}
	default:
		result.Confidence = "none"
		result.Error = "skipped: " + item.Rarity
	}

	return result
}

// priceUniqueItem queries CN trade for a unique item.
func (c *BuildCostCalculator) priceUniqueItem(result *ItemPriceResult, item POBItem, trade *TradeClient, league string, divineRate float64) {
	nameZh := result.NameZh
	baseZh := result.BaseTypeZh

	if nameZh == "" {
		result.Confidence = "none"
		result.Error = fmt.Sprintf("no Chinese name found for unique: %s", item.Name)
		return
	}

	searchReq := &TradeSearchRequest{
		Query: TradeQuery{
			Name:   nameZh,
			Type:   baseZh,
			Status: TradeStatus{Option: "any"},
		},
		Sort: TradeSort{Price: "asc"},
	}

	chaosPrice, currency, rawPrice, confidence, err := trade.SearchAndPrice(league, searchReq)
	if err != nil {
		result.Confidence = "none"
		result.Error = fmt.Sprintf("trade search failed: %v", err)
		return
	}

	if confidence == "none" {
		result.Confidence = "none"
		result.Error = "no listings found"
		return
	}

	priceChaos := CurrencyToChaos(chaosPrice, currency, divineRate)
	result.PriceChaos = round2(priceChaos)
	result.Currency = currency
	result.PriceRaw = rawPrice
	result.Confidence = confidence
}

// priceGem queries CN trade for a skill gem.
func (c *BuildCostCalculator) priceGem(gem POBGem, trade *TradeClient, league string, divineRate float64) ItemPriceResult {
	result := ItemPriceResult{
		Name:       gem.NameSpec,
		Rarity:     "GEM",
		Slot:       gem.Slot,
		GemLevel:   gem.Level,
		GemQuality: gem.Quality,
	}

	gemNameZh := ""
	if zh, ok := c.gemEnToZh[gem.NameSpec]; ok {
		gemNameZh = zh
	}

	if gemNameZh == "" {
		// Try with "Vaal" prefix stripped
		stripped := strings.TrimPrefix(gem.NameSpec, "Vaal ")
		if zh, ok := c.gemEnToZh[stripped]; ok {
			gemNameZh = "瓦尔 " + zh
		}
	}

	// Try stripping " Support" suffix (some gems listed without it in translator data)
	if gemNameZh == "" {
		stripped := strings.TrimSuffix(gem.NameSpec, " Support")
		if stripped != gem.NameSpec {
			if zh, ok := c.gemEnToZh[stripped]; ok {
				gemNameZh = zh
			}
		}
	}

	// Fallback to extra gem names (Enlighten, Enhance, etc.)
	if gemNameZh == "" {
		if zh, ok := extraGemNames[gem.NameSpec]; ok {
			gemNameZh = zh
		} else if zh, ok := extraGemNames[strings.TrimSuffix(gem.NameSpec, " Support")]; ok {
			gemNameZh = zh
		}
	}

	if gemNameZh == "" {
		result.Confidence = "none"
		result.Error = fmt.Sprintf("no Chinese name for gem: %s", gem.NameSpec)
		return result
	}

	result.NameZh = gemNameZh

	// Only price notable gems (level >= 19 or quality >= 20)
	if gem.Level < 19 && gem.Quality < 20 {
		result.Confidence = "low"
		result.PriceChaos = 0
		result.Error = "low-level gem, assumed free"
		return result
	}

	searchReq := &TradeSearchRequest{
		Query: TradeQuery{
			Type:   gemNameZh,
			Status: TradeStatus{Option: "any"},
			Filters: &TradeFilters{
				MiscFilters: &MiscFilters{
					Filters: map[string]FilterMinMax{
						"gem_level": {Min: intPtr(gem.Level)},
					},
				},
			},
		},
		Sort: TradeSort{Price: "asc"},
	}

	if gem.Quality >= 20 {
		searchReq.Query.Filters.MiscFilters.Filters["quality"] = FilterMinMax{Min: intPtr(gem.Quality)}
	}

	chaosPrice, currency, rawPrice, confidence, err := trade.SearchAndPrice(league, searchReq)
	if err != nil {
		result.Confidence = "none"
		result.Error = fmt.Sprintf("trade search failed: %v", err)
		return result
	}

	if confidence == "none" {
		result.Confidence = "none"
		result.Error = "no listings found"
		return result
	}

	priceChaos := CurrencyToChaos(chaosPrice, currency, divineRate)
	result.PriceChaos = round2(priceChaos)
	result.Currency = currency
	result.PriceRaw = rawPrice
	result.Confidence = confidence

	return result
}

// --- Helpers ---

// mutationPrefixes are prefixes added by POB for mutated/anointed unique items.
// They need to be stripped before looking up Chinese names.
var mutationPrefixes = []string{
	"Foulborn ", "Blightborn ", "Soulwrest ", "Eldritch ",
	"Crucible ", "Scourged ", "Synthesised ",
}

// stripMutationPrefix removes known mutation prefixes from unique item names.
// e.g. "Foulborn Uul-Netol's Kiss" → "Uul-Netol's Kiss"
func stripMutationPrefix(name string) string {
	for _, prefix := range mutationPrefixes {
		if strings.HasPrefix(name, prefix) {
			return strings.TrimPrefix(name, prefix)
		}
	}
	return name
}

// extraGemNames maps English gem names that may not be in the translator data.
var extraGemNames = map[string]string{
	"Enlighten":         "启蒙",
	"Enhance":           "强化",
	"Empower":           "赋能",
	"Cooldown Recovery":  "冷却恢复",
	"Autoexertion":      "自动增助",
}

func isFlaskSlot(slot string) bool {
	return strings.HasPrefix(slot, "Flask")
}

func isFlaskBaseType(baseType string) bool {
	return strings.HasSuffix(baseType, "Flask")
}
