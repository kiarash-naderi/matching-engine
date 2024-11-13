package liquiditypool

type LiquidityPoolClient interface {
    GetAvailableLiquidity(isBuyOrder bool) (float64, bool)
    TradeWithPool(orderId string, amount float64, isBuy bool) (float64, error)
    GetCurrentPrice(asset string) float64
} 