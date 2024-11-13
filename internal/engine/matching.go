package engine

import (
    "fmt"
    "matching-engine/internal/engine/liquiditypool"
    "sort"
    
)

type MatchingEngine struct {
    orderBook     OrderBook
    liquidityPool liquiditypool.LiquidityPoolClient
}

func NewMatchingEngine(lp liquiditypool.LiquidityPoolClient) *MatchingEngine {
    return &MatchingEngine{
        orderBook: OrderBook{
            BuyOrders:  make([]Order, 0),
            SellOrders: make([]Order, 0),
        },
        liquidityPool: lp,
    }
}

func (e *MatchingEngine) ProcessOrder(order Order) MatchResult {
    order.InitialAmount = order.Amount
    order.FilledAmount = 0

    currentPrice := e.liquidityPool.GetCurrentPrice(order.Asset)

    if e.checkStopLossAndTakeProfit(order, currentPrice) {
        return MatchResult{
            Success:         true,
            FilledAmount:    order.Amount,
            RemainingAmount: 0,
            ExecutedPrice:   currentPrice,
            Message:         fmt.Sprintf("Order executed by stop loss/take profit at price %.2f", currentPrice),
        }
    }

    if order.Type == Market {
        return e.processMarketOrder(order)
    }
    return e.processLimitOrder(order)
}

func (e *MatchingEngine) processMarketOrder(order Order) MatchResult {
    var matchingOrders *[]Order
    if order.IsBuyOrder {
        matchingOrders = &e.orderBook.SellOrders
    } else {
        matchingOrders = &e.orderBook.BuyOrders
    }

    result := e.matchOrders(order, matchingOrders)
    
    if result.RemainingAmount > 0 {
        if lpFilled, err := e.tryLiquidityPool(order, result.RemainingAmount); err == nil {
            result.FilledAmount += lpFilled
            result.RemainingAmount -= lpFilled
            result.Success = result.RemainingAmount == 0
        }
    }

    return result
}

func (e *MatchingEngine) processLimitOrder(order Order) MatchResult {
    var matchingOrders *[]Order
    if order.IsBuyOrder {
        matchingOrders = &e.orderBook.SellOrders
    } else {
        matchingOrders = &e.orderBook.BuyOrders
    }

    result := e.matchOrders(order, matchingOrders)

    if result.RemainingAmount > 0 {
        if lpFilled, err := e.tryLiquidityPool(order, result.RemainingAmount); err == nil {
            result.FilledAmount += lpFilled
            result.RemainingAmount -= lpFilled
            result.Success = result.RemainingAmount == 0
        }

        if result.RemainingAmount > 0 {
            remainingOrder := order
            remainingOrder.Amount = result.RemainingAmount
            if order.IsBuyOrder {
                e.orderBook.BuyOrders = append(e.orderBook.BuyOrders, remainingOrder)
            } else {
                e.orderBook.SellOrders = append(e.orderBook.SellOrders, remainingOrder)
            }
        }
    }

    return result
}

// 1. Change in matchOrders function - replace the current function with this version
func (e *MatchingEngine) matchOrders(order Order, matchingOrders *[]Order) MatchResult {
    remainingAmount := order.Amount
    filledAmount := 0.0
    weightedSum := 0.0
    executedPrice := 0.0
    orderbookFilled := 0.0
    lpFilled := 0.0

    // Sort orders by price
    if order.IsBuyOrder {
        sort.Slice(*matchingOrders, func(i, j int) bool {
            return (*matchingOrders)[i].Price < (*matchingOrders)[j].Price
        })
    } else {
        sort.Slice(*matchingOrders, func(i, j int) bool {
            return (*matchingOrders)[i].Price > (*matchingOrders)[j].Price
        })
    }

    for i := 0; i < len(*matchingOrders) && remainingAmount > 0; i++ {
        matched := &(*matchingOrders)[i]
        
        if matched.FilledAmount >= matched.Amount {
            continue
        }

        if order.Type == Limit {
            if (order.IsBuyOrder && matched.Price > order.Price) ||
               (!order.IsBuyOrder && matched.Price < order.Price) {
                break
            }
        }

        matchAmount := min(remainingAmount, matched.Amount-matched.FilledAmount)
        if matchAmount > 0 {
            remainingAmount -= matchAmount
            filledAmount += matchAmount
            orderbookFilled += matchAmount
            weightedSum += matchAmount * matched.Price
            matched.FilledAmount += matchAmount

            // Add log details
            // fmt.Printf("Matched %.2f units between %s and %s at price %.2f\n",
            //    matchAmount, order.ID, matched.ID, matched.Price)
        }
    }

    e.cleanupOrders(matchingOrders)

    if filledAmount > 0 {
        executedPrice = weightedSum / filledAmount
    }

    // Correct message format
    message := fmt.Sprintf("Order %s: Initial: %.2f, Filled: %.2f (%.2f from orderbook, %.2f from LP)",
        order.ID, order.InitialAmount, filledAmount, orderbookFilled, lpFilled)
    
    return MatchResult{
        Success:         remainingAmount == 0,
        FilledAmount:    filledAmount,
        RemainingAmount: remainingAmount,
        ExecutedPrice:   executedPrice,
        Message:         message,
    }
}

// 2. Change in formatMessage function
func (e *MatchingEngine) formatMessage(order Order, orderbookFill float64, lpFill float64) string {
    return fmt.Sprintf("Order %s: Initial: %.2f, Filled: %.2f (%.2f from orderbook, %.2f from LP)",
        order.ID, order.InitialAmount, orderbookFill + lpFill, orderbookFill, lpFill)
}

func (e *MatchingEngine) cleanupOrders(orders *[]Order) {
    newOrders := make([]Order, 0)
    for _, order := range *orders {
        if order.FilledAmount < order.Amount {
            newOrders = append(newOrders, order)
        }
    }
    *orders = newOrders
}

func min(a, b float64) float64 {
    if a < b {
        return a
    }
    return b
}

func (e *MatchingEngine) checkStopLossAndTakeProfit(order Order, currentPrice float64) bool {
    if order.IsBuyOrder {
        if order.StopLossPrice > 0 && currentPrice <= order.StopLossPrice {
            return true // Stop loss triggered
        }
        if order.TakeProfitPrice > 0 && currentPrice >= order.TakeProfitPrice {
            return true // Take profit triggered
        }
    } else {
        if order.StopLossPrice > 0 && currentPrice >= order.StopLossPrice {
            return true // Stop loss triggered
        }
        if order.TakeProfitPrice > 0 && currentPrice >= order.TakeProfitPrice {
            return true // Take profit triggered for sell order
        }
    }
    return false
}

func (e *MatchingEngine) isPriceAcceptable(order Order, currentPrice float64) bool {
    if order.Type != Limit {
        return true
    }
    if order.IsBuyOrder {
        return currentPrice <= order.Price
    }
    return currentPrice >= order.Price
}

func (e *MatchingEngine) tryLiquidityPool(order Order, amount float64) (float64, error) {
    if e.liquidityPool == nil {
        return 0, fmt.Errorf("no liquidity pool available")
    }

    available, ok := e.liquidityPool.GetAvailableLiquidity(order.IsBuyOrder)
    if !ok || available <= 0 {
        return 0, fmt.Errorf("insufficient liquidity in pool")
    }

    lpAmount := min(amount, available)
    filled, err := e.liquidityPool.TradeWithPool(order.ID, lpAmount, order.IsBuyOrder)
    return filled, err
}
