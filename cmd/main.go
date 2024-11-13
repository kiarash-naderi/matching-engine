package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"matching-engine/internal/engine"
	"matching-engine/internal/engine/liquiditypool"
	"matching-engine/internal/config"
)

var matchingEngine *engine.MatchingEngine

type OrderRequest struct {
	Price           float64 `json:"price"`
	Amount          float64 `json:"amount"`
	IsBuyOrder      bool    `json:"is_buy_order"`
	Type            string  `json:"type"`
	Asset           string  `json:"asset"`
	Trader          string  `json:"trader"`
	Leverage        int64   `json:"leverage"`
	MarginType      string  `json:"margin_type"`
	StopLossPrice   float64 `json:"stop_loss_price"`
	TakeProfitPrice float64 `json:"take_profit_price"`
}

func handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var orderReq OrderRequest
	if err := json.NewDecoder(r.Body).Decode(&orderReq); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Convert OrderRequest to Order
	order := engine.Order{
		Price:           orderReq.Price,
		Amount:          orderReq.Amount,
		IsBuyOrder:      orderReq.IsBuyOrder,
		Asset:           orderReq.Asset,
		Trader:          orderReq.Trader,
		Leverage:        orderReq.Leverage,
		StopLossPrice:   orderReq.StopLossPrice,
		TakeProfitPrice: orderReq.TakeProfitPrice,
	}

	// Determine order type
	if orderReq.Type == "market" {
		order.Type = engine.Market
	} else {
		order.Type = engine.Limit
	}

	// Determine margin type
	if orderReq.MarginType == "cross" {
		order.MarginType = engine.Cross
	} else {
		order.MarginType = engine.Isolated
	}

	// Process order
	result := matchingEngine.ProcessOrder(order)

	// Convert result to JSON and send
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)

	// If the order is partially filled, log the message
	if !result.Success {
		log.Printf("Partial fill detected: %s", result.Message)
	}
}

func main() {
	// Initialize liquidity pool client
	lpClient := liquiditypool.NewClient(config.LiquidityPoolURL)
	
	// Initialize matching engine with liquidity pool
	matchingEngine = engine.NewMatchingEngine(lpClient)

	// Define routes
	http.HandleFunc("/api/order", handleCreateOrder)

	// Start server
	port := ":8080"
	fmt.Printf("Server starting on port %s\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
}