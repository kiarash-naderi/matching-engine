package engine

type OrderType int
type MarginType int

const (
	Market OrderType = iota
	Limit
)

const (
	Cross MarginType = iota
	Isolated
)

// Order represents an order in the orderbook
type Order struct {
	ID            string
	Price         float64
	Amount        float64
	InitialAmount float64 // Initial amount of the order
	FilledAmount  float64 // Amount of the order that has been filled
	Type          OrderType
	IsBuyOrder    bool
	Trader        string
	Asset         string
	Leverage      int64
	MarginType    MarginType
	Expiration    int64
	StopLossPrice float64
	TakeProfitPrice float64
}

// MatchResult represents the result of order matching
type MatchResult struct {
	Success         bool
	FilledAmount    float64
	RemainingAmount float64
	ExecutedPrice   float64
	Message         string
}

// OrderBook maintains the buy and sell orders
type OrderBook struct {
	BuyOrders  []Order
	SellOrders []Order
}