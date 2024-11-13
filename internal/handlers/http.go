package handlers

import (
    "encoding/json"
    "github.com/gorilla/mux"
    "matching-engine/internal/engine"
    "matching-engine/pkg/utils"
    "net/http"
)

type Handler struct {
    engine *engine.MatchingEngine
}

func NewHandler(e *engine.MatchingEngine) *Handler {
    return &Handler{engine: e}
}

func (h *Handler) SetupRoutes(r *mux.Router) {
    r.HandleFunc("/api/health", h.healthCheck).Methods("GET")
    r.HandleFunc("/api/order", h.createOrder).Methods("POST")
}

func (h *Handler) healthCheck(w http.ResponseWriter, r *http.Request) {
    json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *Handler) createOrder(w http.ResponseWriter, r *http.Request) {
    var orderReq struct {
        Price      float64 `json:"price"`
        Amount     float64 `json:"amount"`
        IsBuyOrder bool    `json:"is_buy_order"`
        Type       string  `json:"type"`
        Asset      string  `json:"asset"`
        Trader     string  `json:"trader"`
        Leverage   int64   `json:"leverage"`
        MarginType string  `json:"margin_type"`
    }

    if err := json.NewDecoder(r.Body).Decode(&orderReq); err != nil {
        utils.Logger.Error("Failed to decode request", err)
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    order := engine.Order{
        Price:      orderReq.Price,
        Amount:     orderReq.Amount,
        IsBuyOrder: orderReq.IsBuyOrder,
        Asset:      orderReq.Asset,
        Trader:     orderReq.Trader,
        Leverage:   orderReq.Leverage,
    }

    if orderReq.Type == "market" {
        order.Type = engine.Market
    } else {
        order.Type = engine.Limit
    }

    result := h.engine.ProcessOrder(order)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(result)

    if !result.Success {
        utils.LogMatchResult(order.ID, result.Message)
    }
}