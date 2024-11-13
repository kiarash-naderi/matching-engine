package engine

import (
	"fmt"
	"testing"
	"time"
)

type MockLiquidityPool struct {
	shouldFail bool
}

func (m *MockLiquidityPool) GetAvailableLiquidity(isBuyOrder bool) (float64, bool) {
	if m.shouldFail {
		return 0.0, false
	}
	return 1000.0, true
}

func (m *MockLiquidityPool) TradeWithPool(orderId string, amount float64, isBuy bool) (float64, error) {
	if m.shouldFail {
		return 0.0, nil
	}
	// We only fill half of the requested amount
	return amount / 2, nil
}

func (m *MockLiquidityPool) GetCurrentPrice(asset string) float64 {
	return 100.0 // Fixed price for testing
}

func TestMarketOrderMatching(t *testing.T) {
	mockLP := &MockLiquidityPool{}
	engine := NewMatchingEngine(mockLP)

	// Create a sell order in the order book
	sellOrder := Order{
		ID:            "sell-1",
		Price:         100.0,
		Amount:        10.0,
		InitialAmount: 10.0,
		Type:          Limit,
		IsBuyOrder:    false,
	}
	engine.orderBook.SellOrders = append(engine.orderBook.SellOrders, sellOrder)

	// Create a market buy order
	buyOrder := Order{
		ID:            "buy-1",
		Amount:        5.0,
		InitialAmount: 5.0,
		Type:          Market,
		IsBuyOrder:    true,
	}

	result := engine.ProcessOrder(buyOrder)

	if !result.Success {
		t.Errorf("Expected successful match, got failure")
	}
	if result.FilledAmount != 5.0 {
		t.Errorf("Expected filled amount 5.0, got %f", result.FilledAmount)
	}
	if result.ExecutedPrice != 100.0 {
		t.Errorf("Expected executed price 100.0, got %f", result.ExecutedPrice)
	}
}

// TestPartialFillWithLiquidityPool tests the scenario where a buy order is partially filled
// using both the order book and a liquidity pool. The test sets up a large buy order and a 
// smaller sell order in the order book. It then processes the buy order and verifies that 
// the order is partially filled with the expected amounts from the order book and the 
// liquidity pool. The test checks the filled amount, the remaining amount, and the 
// resulting message to ensure they match the expected values.
func TestPartialFillWithLiquidityPool(t *testing.T) {
	mockLP := &MockLiquidityPool{shouldFail: false}
	engine := NewMatchingEngine(mockLP)

	// Create a large buy order
	buyOrder := Order{
		ID:            "buy-2",
		Amount:        20.0,
		InitialAmount: 20.0,
		Type:          Market,
		IsBuyOrder:    true,
	}

	// Create a smaller sell order in the order book
	sellOrder := Order{
		ID:            "sell-2",
		Price:         100.0,
		Amount:        5.0,
		InitialAmount: 5.0,
		Type:          Limit,
		IsBuyOrder:    false,
	}
	engine.orderBook.SellOrders = append(engine.orderBook.SellOrders, sellOrder)

	result := engine.ProcessOrder(buyOrder)

	if result.Success {
		t.Errorf("Expected partial fill")
	}

	// 5.0 from order book and 7.5 from liquidity pool (half of the remaining 15.0)
	expectedFilled := 12.5
	expectedRemaining := 7.5

	if result.FilledAmount != expectedFilled {
		t.Errorf("Expected %f to be filled, but %f was filled", expectedFilled, result.FilledAmount)
	}

	if result.RemainingAmount != expectedRemaining {
		t.Errorf("Expected %f to remain, but %f remained", expectedRemaining, result.RemainingAmount)
	}

	expectedMsg := fmt.Sprintf("Order partially filled. Initial amount: %.2f, Filled: %.2f, Unfilled: %.2f",
		buyOrder.InitialAmount, expectedFilled, expectedRemaining)
	if result.Message != expectedMsg {
		t.Errorf("Expected message: '%s', got: '%s'", expectedMsg, result.Message)
	}
}
func TestLimitOrderMatching(t *testing.T) {
	mockLP := &MockLiquidityPool{shouldFail: false}
	engine := NewMatchingEngine(mockLP)

	// Create a limit sell order
	sellOrder := Order{
		ID:            "sell-3",
		Price:         100.0,
		Amount:        10.0,
		InitialAmount: 10.0,
		Type:          Limit,
		IsBuyOrder:    false,
	}
	engine.orderBook.SellOrders = append(engine.orderBook.SellOrders, sellOrder)

	// Create a limit buy order with a matching price
	buyOrder := Order{
		ID:            "buy-3",
		Price:         100.0,
		Amount:        5.0,
		InitialAmount: 5.0,
		Type:          Limit,
		IsBuyOrder:    true,
	}

	result := engine.ProcessOrder(buyOrder)

	if !result.Success {
		t.Errorf("Expected successful match for limit order")
	}
	if result.FilledAmount != 5.0 {
		t.Errorf("Expected filled amount 5.0, got %f", result.FilledAmount)
	}
}

func TestLargeMarketOrderWithMultipleMatches(t *testing.T) {
	mockLP := &MockLiquidityPool{shouldFail: true}
	engine := NewMatchingEngine(mockLP)

	// Create multiple sell orders in the order book
	sellOrders := []Order{
		{
			ID:            "sell-4",
			Price:         100.0,
			Amount:        5.0,
			InitialAmount: 5.0,
			Type:          Limit,
			IsBuyOrder:    false,
		},
		{
			ID:            "sell-5",
			Price:         101.0,
			Amount:        7.0,
			InitialAmount: 7.0,
			Type:          Limit,
			IsBuyOrder:    false,
		},
	}

	engine.orderBook.SellOrders = append(engine.orderBook.SellOrders, sellOrders...)

	// Create a large market buy order
	buyOrder := Order{
		ID:            "buy-4",
		Amount:        15.0,
		InitialAmount: 15.0,
		Type:          Market,
		IsBuyOrder:    true,
	}

	result := engine.ProcessOrder(buyOrder)

	expectedFilled := 12.0 // 5.0 + 7.0 from the order book
	if result.FilledAmount != expectedFilled {
		t.Errorf("Expected %f to be filled, got %f", expectedFilled, result.FilledAmount)
	}

	expectedMsg := fmt.Sprintf("Order partially filled. Initial amount: %.2f, Filled: %.2f, Unfilled: %.2f",
		buyOrder.InitialAmount, expectedFilled, buyOrder.Amount-expectedFilled)
	if result.Message != expectedMsg {
		t.Errorf("Expected message: '%s', got: '%s'", expectedMsg, result.Message)
	}
}

func TestComplexOrderScenario(t *testing.T) {
	mockLP := &MockLiquidityPool{shouldFail: false}
	engine := NewMatchingEngine(mockLP)

	// 1. Create several sell orders with different prices
	sellOrders := []Order{
		{
			ID:            "sell-1",
			Price:         100.0,
			Amount:        5.0,
			InitialAmount: 5.0,
			Type:          Limit,
			IsBuyOrder:    false,
			Asset:         "BTC",
			Leverage:      2,
			MarginType:    Cross,
			Expiration:    time.Now().Add(1 * time.Hour).Unix(),
		},
		{
			ID:            "sell-2",
			Price:         102.0,
			Amount:        7.0,
			InitialAmount: 7.0,
			Type:          Limit,
			IsBuyOrder:    false,
			Asset:         "BTC",
			Leverage:      1,
			MarginType:    Isolated,
			Expiration:    time.Now().Add(1 * time.Hour).Unix(),
		},
	}

	for _, order := range sellOrders {
		engine.orderBook.SellOrders = append(engine.orderBook.SellOrders, order)
	}

	// 2. Create a large buy order
	buyOrder := Order{
		ID:            "buy-1",
		Amount:        15.0,
		InitialAmount: 15.0,
		Type:          Market,
		IsBuyOrder:    true,
		Asset:         "BTC",
		Leverage:      2,
		MarginType:    Cross,
	}

	result := engine.ProcessOrder(buyOrder)

	// 3. Check results
	// Should fill 13.5 units (5.0 from first sell order + 7.0 from second sell order + 1.5 from liquidity pool)
	expectedFilled := 13.5
	if result.FilledAmount != expectedFilled {
		t.Errorf("Expected %f to be filled, got %f", expectedFilled, result.FilledAmount)
	}

	// Check remaining amount
	expectedRemaining := 1.5 // 15.0 - 13.5
	if result.RemainingAmount != expectedRemaining {
		t.Errorf("Expected remaining amount %f, got %f", expectedRemaining, result.RemainingAmount)
	}

	// Get current price from mock liquidity pool
	currentPrice := mockLP.GetCurrentPrice("BTC")

	// Calculate weighted average price for verification
	executedPrice := (5.0*100.0 + 7.0*102.0 + 1.5*currentPrice) / 13.5

	// Clean up filled orders
	engine.cleanupOrders(&engine.orderBook.SellOrders)

	// Verify executed price
	if result.ExecutedPrice != executedPrice {
		t.Errorf("Expected executed price %.2f, got %.2f", executedPrice, result.ExecutedPrice)
	}

	expectedMsg := fmt.Sprintf("Order partially filled. Initial amount: %.2f, Filled: %.2f, Unfilled: %.2f",
		buyOrder.InitialAmount, expectedFilled, expectedRemaining)
	if result.Message != expectedMsg {
		t.Errorf("Expected message: '%s', got: '%s'", expectedMsg, result.Message)
	}

	// 4. Check order book status
	if len(engine.orderBook.SellOrders) != 0 {
		t.Error("Expected empty sell orders after matching")
	}
}

func TestStopLossAndTakeProfit(t *testing.T) {
	mockLP := &MockLiquidityPool{shouldFail: false}
	engine := NewMatchingEngine(mockLP)

	// Create a buy order with stop loss
	buyOrder := Order{
		ID:            "buy-sl-1",
		Price:         100.0,
		Amount:        5.0,
		InitialAmount: 5.0,
		Type:          Limit,
		IsBuyOrder:    true,
		StopLossPrice: 95.0,
		Asset:         "BTC",
	}

	// Test stop loss trigger
	currentPrice := 94.0
	triggered := engine.checkStopLossAndTakeProfit(buyOrder, currentPrice)
	if !triggered {
		t.Errorf("Expected stop loss to trigger at price %.2f", currentPrice)
	}

	// Create a sell order with take profit
	sellOrder := Order{
		ID:              "sell-tp-1",
		Price:           100.0,
		Amount:          5.0,
		InitialAmount:   5.0,
		Type:           Limit,
		IsBuyOrder:     false,
		TakeProfitPrice: 105.0,
		Asset:          "BTC",
	}

	// Test take profit trigger
	currentPrice = 106.0
	triggered = engine.checkStopLossAndTakeProfit(sellOrder, currentPrice)
	if !triggered {
		t.Errorf("Expected take profit to trigger at price %.2f", currentPrice)
	}
}
