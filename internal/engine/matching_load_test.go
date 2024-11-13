package engine

import (
	"fmt"
	"runtime"
	"testing"
	"time"
)

func TestHighVolumeComplexOrderMatching(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	mockLP := &MockLiquidityPool{shouldFail: false}
	engine := NewMatchingEngine(mockLP)

	// Track initial memory stats
	var initialMemStats runtime.MemStats
	runtime.ReadMemStats(&initialMemStats)

	// Create 400 orders with different configurations
	createTestOrders(t, engine)

	// Process large market orders and verify results
	processAndVerifyLargeOrders(t, engine)

	// Test stop loss and take profit triggers
	testStopLossAndTakeProfit(t, engine)

	// Report memory usage
	reportMemoryStats(t, initialMemStats)
}

func createTestOrders(t *testing.T, engine *MatchingEngine) {
	assets := []string{"BTC", "ETH", "SOL", "AVAX"}
	basePrice := 40000.0
	
	// Define SL/TP percentages
	const (
		slPercentage = 0.05  // 5% for Stop Loss
		tpPercentage = 0.05  // 5% for Take Profit
	)
	
	for i := 0; i < 5000; i++ {
		// Sell orders
		sellPrice := basePrice + float64(i%10)
		sellOrder := Order{
			ID:              fmt.Sprintf("sell-limit-%d", i),
			Price:           sellPrice,
			Amount:          1.0 + float64(i%10),
			InitialAmount:   1.0 + float64(i%10),
			Type:           Limit,
			IsBuyOrder:     false,
			Asset:          assets[i%len(assets)],
			Leverage:       int64(1 + i%3),
			MarginType:     MarginType(i % 2),
			Trader:         fmt.Sprintf("trader-%d", i%20),
			// Add SL/TP for sell orders
			StopLossPrice:   sellPrice * (1 - slPercentage),  // 5% below sell price
			TakeProfitPrice: sellPrice * (1 + tpPercentage),  // 5% above sell price
		}
		engine.orderBook.SellOrders = append(engine.orderBook.SellOrders, sellOrder)

		// Market sell orders
		if i%5 == 0 {
			marketSellOrder := Order{
				ID:            fmt.Sprintf("market-sell-%d", i),
				Amount:        2.0 + float64(i%10),
				InitialAmount: 2.0 + float64(i%10),
				Type:         Market,
				IsBuyOrder:   false,
				Asset:        assets[i%len(assets)],
				Leverage:     int64(1 + i%3),
				MarginType:   MarginType(i % 2),
				Trader:       fmt.Sprintf("trader-%d", i%20),
				// Add SL/TP for market sell orders
				StopLossPrice:   basePrice * (1 - slPercentage),
				TakeProfitPrice: basePrice * (1 + tpPercentage),
			}
			engine.orderBook.SellOrders = append(engine.orderBook.SellOrders, marketSellOrder)
		}

		// Buy orders
		buyPrice := basePrice - float64(i%10)
		buyOrder := Order{
			ID:              fmt.Sprintf("buy-limit-%d", i),
			Price:           buyPrice,
			Amount:          1.0 + float64(i%10),
			InitialAmount:   1.0 + float64(i%10),
			Type:           Limit,
			IsBuyOrder:     true,
			Asset:          assets[i%len(assets)],
			Leverage:       int64(1 + i%3),
			MarginType:     MarginType(i % 2),
			Trader:         fmt.Sprintf("trader-%d", i%20),
			// Add SL/TP for buy orders
			StopLossPrice:   buyPrice * (1 + slPercentage),  // 5% above buy price
			TakeProfitPrice: buyPrice * (1 - tpPercentage),  // 5% below buy price
		}
		engine.orderBook.BuyOrders = append(engine.orderBook.BuyOrders, buyOrder)
	}

	t.Logf("Created %d buy orders and %d sell orders", 
		len(engine.orderBook.BuyOrders), 
		len(engine.orderBook.SellOrders))
}

func processAndVerifyLargeOrders(t *testing.T, engine *MatchingEngine) {
	// Define large orders for testing
	largeOrders := []Order{
		{
			ID:            "limit-sell-large-1",
			Amount:        250.0,
			InitialAmount: 250.0,
			Type:         Limit,
			IsBuyOrder:   false,
			Price:        40005.0,
			Asset:        "BTC",
		},
		{
			ID:            "limit-buy-super-huge",
			Amount:        50000.0,
			InitialAmount: 50000.0,
			Type:         Limit,
			IsBuyOrder:   true,
			Price:        40000.0,
			Asset:        "BTC",
		},
	}

	// Add diverse orders
	basePrice := 40000.0
	priceSpread := 50.0

	for i := 1; i <= 25; i++ {
		// Market Buy Orders
		marketBuyAmount := 100.0 + float64(i%10)*10
		largeOrders = append(largeOrders, Order{
			ID:            fmt.Sprintf("market-buy-%d", i),
			Amount:        marketBuyAmount,
			InitialAmount: marketBuyAmount,
			Type:         Market,
			IsBuyOrder:   true,
			Asset:        "BTC",
		})

		// Limit Buy Orders
		limitBuyAmount := 50.0 + float64(i%10)*10
		buyPrice := basePrice + float64(i)*priceSpread/2
		largeOrders = append(largeOrders, Order{
			ID:            fmt.Sprintf("limit-buy-%d", i),
			Amount:        limitBuyAmount,
			InitialAmount: limitBuyAmount,
			Type:         Limit,
			IsBuyOrder:   true,
			Price:        buyPrice,
			Asset:        "BTC",
		})

		// Limit Sell Orders
		limitSellAmount := 75.0 + float64(i%10)*10
		sellPrice := basePrice - float64(i)*priceSpread/2
		largeOrders = append(largeOrders, Order{
			ID:            fmt.Sprintf("limit-sell-%d", i),
			Amount:        limitSellAmount,
			InitialAmount: limitSellAmount,
			Type:         Limit,
			IsBuyOrder:   false,
			Price:        sellPrice,
			Asset:        "BTC",
		})
	}

	t.Log("=== Performance Test Started ===")
	t.Logf("Total test orders: %d", len(largeOrders))

	var (
		totalExecutionTime time.Duration
		fastestOrder      time.Duration = time.Hour
		slowestOrder      time.Duration
		totalVolume       float64
		unfilledOrders    int
		totalLPVolume     float64
	)

	// Process orders
	for _, order := range largeOrders {
		startTime := time.Now()
		result := engine.ProcessOrder(order)
		executionTime := time.Since(startTime)

		totalExecutionTime += executionTime
		if executionTime < fastestOrder {
			fastestOrder = executionTime
		}
		if executionTime > slowestOrder {
			slowestOrder = executionTime
		}

		// Calculate various order metrics
		var orderID string
		var initialAmount, totalFilled, orderbookFill, lpFill float64
		_, err := fmt.Sscanf(result.Message, 
			"Order %s: Initial: %f, Filled: %f (%f from orderbook, %f from LP)",
			&orderID, &initialAmount, &totalFilled, &orderbookFill, &lpFill)
		
		if err != nil {
			orderbookFill = result.FilledAmount
			lpFill = 0
			if result.RemainingAmount > 0 {
				lpFill = result.RemainingAmount
			}
		}

		// Log unfilled orders with new format
		if result.RemainingAmount > 0 {
			unfilledOrders++
			orderTypeStr := "Market"
			if order.Type == Limit {
				orderTypeStr = fmt.Sprintf("Limit(%.2f)", order.Price)
			}
			sideStr := "Buy"
			if !order.IsBuyOrder {
				sideStr = "Sell"
			}
			
			t.Logf("Order [%s-%s %s] >> Initial Amount: %.2f, Not Filled: %.2f => Sending to Liquidity Pool", 
				orderTypeStr, 
				sideStr,
				order.ID,
				order.InitialAmount,
				result.RemainingAmount)
		}

		totalVolume += orderbookFill + lpFill
		totalLPVolume += lpFill
	}

	// Final report - only important stats
	t.Log("\n=== Performance Summary ===")
	t.Logf("Total orders: %d", len(largeOrders))
	t.Logf("Average processing time: %v", totalExecutionTime/time.Duration(len(largeOrders)))
	t.Logf("Orders per second: %.2f", float64(len(largeOrders))/totalExecutionTime.Seconds())
	t.Logf("Unfilled orders: %d (%.1f%%)", unfilledOrders, 
		float64(unfilledOrders)/float64(len(largeOrders))*100)
	t.Logf("Total volume sent to LP: %.2f units", totalLPVolume)
	t.Log("=========================")
}

func testStopLossAndTakeProfit(t *testing.T, engine *MatchingEngine) {
	basePrice := 40000.0
	
	// Define test prices based on percentage change from base price
	prices := []struct {
		price       float64
		description string
	}{
		{basePrice * 0.94, "6% below base (should trigger sell SL)"},       // For testing sell SL
		{basePrice * 1.06, "6% above base (should trigger buy SL)"},        // For testing buy SL
		{basePrice * 1.06, "6% above base (should trigger sell TP)"},       // For testing sell TP
		{basePrice * 0.94, "6% below base (should trigger buy TP)"},        // For testing buy TP
	}
	
	for _, priceTest := range prices {
		triggered := 0
		start := time.Now()
		
		t.Logf("\nTesting price %.2f (%s)", priceTest.price, priceTest.description)
		
		// Test buy orders
		for _, order := range engine.orderBook.BuyOrders {
			if engine.checkStopLossAndTakeProfit(order, priceTest.price) {
				triggered++
			   // t.Logf("Buy Order %d triggered at %.2f (SL: %.2f, TP: %.2f)", 
				  //  i, priceTest.price, order.StopLossPrice, order.TakeProfitPrice)
			}
		}
		
		// Test sell orders
		for _, order := range engine.orderBook.SellOrders {
			if engine.checkStopLossAndTakeProfit(order, priceTest.price) {
				triggered++
			  //  t.Logf("Sell Order %d triggered at %.2f (SL: %.2f, TP: %.2f)", 
				   // i, priceTest.price, order.StopLossPrice, order.TakeProfitPrice)
			}
		}
		
		duration := time.Since(start)
		t.Logf("SL/TP check at price %.2f took: %v, triggered: %d orders", 
			priceTest.price, duration, triggered)
		
		// Add more detailed stats
	  //  if triggered > 0 {
	  //      t.Logf("%.2f%% of orders were triggered at this price point",
	   //         float64(triggered) / float64(len(engine.orderBook.BuyOrders) + len(engine.orderBook.SellOrders)) * 100)
	  //  }
	}

	// Overall stats
	t.Log("\n=== SL/TP Test Summary ===")
	t.Logf("Total Buy Orders: %d", len(engine.orderBook.BuyOrders))
	t.Logf("Total Sell Orders: %d", len(engine.orderBook.SellOrders))
	t.Log("=========================")
}

func reportMemoryStats(t *testing.T, initialStats runtime.MemStats) {
	var currentStats runtime.MemStats
	runtime.ReadMemStats(&currentStats)
	
	memoryDiff := float64(currentStats.Alloc - initialStats.Alloc) / 1024 / 1024
	t.Logf("Memory usage difference: %.2f MB", memoryDiff)
	t.Logf("Number of garbage collections: %d", currentStats.NumGC)
}