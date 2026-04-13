package pricer

// BuildCostRequest is the input for the /build-cost endpoint.
type BuildCostRequest struct {
	// POB code (base64url + zlib compressed XML) — the main input.
	// This is the same code that poe.ninja builds pages use.
	PobCode string `json:"pob_code"`

	// CN server league name (optional, defaults to current league)
	CNLeague string `json:"cn_league,omitempty"`

	// POESESSID cookie value for CN trade API authentication.
	// Required for querying poe.game.qq.com trade API.
	POESESSID string `json:"poesessid"`
}

// BuildCostResponse is the output of the /build-cost endpoint.
type BuildCostResponse struct {
	Character   string            `json:"character,omitempty"`
	AscClass    string            `json:"asc_class,omitempty"`
	Level       int               `json:"level,omitempty"`
	League      string            `json:"league,omitempty"`
	CNLeague    string            `json:"cn_league"`
	Items       []ItemPriceResult `json:"items"`
	Gems        []ItemPriceResult `json:"gems"`
	TotalChaos  float64           `json:"total_chaos"`
	TotalDivine float64           `json:"total_divine"`
	DivineRate  float64           `json:"divine_rate"` // 1 divine = ? chaos
	Errors      []string          `json:"errors,omitempty"`
}

// ItemPriceResult represents the pricing result for a single item.
type ItemPriceResult struct {
	Name       string  `json:"name"`                    // Display name (unique name or base type)
	BaseType   string  `json:"base_type,omitempty"`     // Base type
	NameZh     string  `json:"name_zh,omitempty"`       // Chinese name
	BaseTypeZh string  `json:"base_type_zh,omitempty"`  // Chinese base type
	Slot       string  `json:"slot,omitempty"`          // Equipment slot (Helmet, Weapon 1, etc.)
	Rarity     string  `json:"rarity"`                  // UNIQUE, RARE, MAGIC, NORMAL
	GemLevel   int     `json:"gem_level,omitempty"`
	GemQuality int     `json:"gem_quality,omitempty"`
	PriceChaos float64 `json:"price_chaos"`
	Currency   string  `json:"currency,omitempty"`
	PriceRaw   float64 `json:"price_raw,omitempty"`
	Confidence string  `json:"confidence"` // high/medium/low/none
	Error      string  `json:"error,omitempty"`
}

// POBItem represents an item parsed from POB XML text format.
type POBItem struct {
	ID       int
	Rarity   string // UNIQUE, RARE, MAGIC, NORMAL
	Name     string // For uniques: the unique name (e.g. "Hyrri's Truth")
	BaseType string // The base type (e.g. "Jade Amulet")
	Slot     string // Equipment slot from ItemSet
	Mods     []string
}

// POBGem represents a gem parsed from POB XML <Gem> elements.
type POBGem struct {
	NameSpec string // English name (e.g. "Tornado Shot")
	SkillID  string
	Level    int
	Quality  int
	Slot     string // Which equipment slot this gem is socketed in
	Enabled  bool
}

// Trade API types for CN server

// TradeSearchRequest is the body for POST /api/trade/search/{league}.
type TradeSearchRequest struct {
	Query TradeQuery `json:"query"`
	Sort  TradeSort  `json:"sort"`
}

// TradeQuery is the query portion of a trade search request.
type TradeQuery struct {
	Name    string        `json:"name,omitempty"`
	Type    string        `json:"type,omitempty"`
	Status  TradeStatus   `json:"status"`
	Stats   []TradeStat   `json:"stats,omitempty"`
	Filters *TradeFilters `json:"filters,omitempty"`
}

// TradeStatus specifies item listing status.
type TradeStatus struct {
	Option string `json:"option"` // "online" or "any"
}

// TradeStat is a stat filter group.
type TradeStat struct {
	Type    string            `json:"type"` // "and", "or", etc.
	Filters []TradeStatFilter `json:"filters"`
}

// TradeStatFilter is an individual stat filter.
type TradeStatFilter struct {
	ID    string      `json:"id"`
	Value interface{} `json:"value,omitempty"`
}

// TradeFilters contains filter groups for trade search.
type TradeFilters struct {
	MiscFilters *MiscFilters `json:"misc_filters,omitempty"`
	TypeFilters *TypeFilters `json:"type_filters,omitempty"`
}

// MiscFilters for gem level/quality etc.
type MiscFilters struct {
	Filters map[string]FilterMinMax `json:"filters"`
}

// TypeFilters for item category.
type TypeFilters struct {
	Filters map[string]FilterOption `json:"filters"`
}

// FilterMinMax represents a min/max range filter.
type FilterMinMax struct {
	Min *int `json:"min,omitempty"`
	Max *int `json:"max,omitempty"`
}

// FilterOption represents an option filter.
type FilterOption struct {
	Option string `json:"option"`
}

// TradeSort specifies result ordering.
type TradeSort struct {
	Price string `json:"price"` // "asc" or "desc"
}

// TradeSearchResponse is the response from trade search.
type TradeSearchResponse struct {
	ID     string   `json:"id"`
	Result []string `json:"result"`
	Total  int      `json:"total"`
}

// TradeFetchResponse is the response from trade fetch.
type TradeFetchResponse struct {
	Result []TradeFetchResult `json:"result"`
}

// TradeFetchResult is a single item result from trade fetch.
type TradeFetchResult struct {
	ID      string `json:"id"`
	Listing struct {
		Price struct {
			Type     string  `json:"type"`
			Amount   float64 `json:"amount"`
			Currency string  `json:"currency"`
		} `json:"price"`
	} `json:"listing"`
}
