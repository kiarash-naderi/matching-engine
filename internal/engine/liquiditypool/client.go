package liquiditypool

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
)

type Client struct {
    baseURL string
    client  *http.Client
}

func NewClient(baseURL string) LiquidityPoolClient {
    return &Client{
        baseURL: baseURL,
        client:  &http.Client{},
    }
}

func (c *Client) GetAvailableLiquidity(isBuyOrder bool) (float64, bool) {
    resp, err := c.client.Get(fmt.Sprintf("%s/liquidity?isBuyOrder=%v", c.baseURL, isBuyOrder))
    if err != nil {
        return 0, false
    }
    defer resp.Body.Close()

    var result struct {
        Amount float64 `json:"amount"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return 0, false
    }
    return result.Amount, true
}

func (c *Client) TradeWithPool(orderId string, amount float64, isBuy bool) (float64, error) {
    payload := struct {
        OrderID string  `json:"order_id"`
        Amount  float64 `json:"amount"`
        IsBuy   bool    `json:"is_buy"`
    }{
        OrderID: orderId,
        Amount:  amount,
        IsBuy:   isBuy,
    }

    jsonData, err := json.Marshal(payload)
    if err != nil {
        return 0, err
    }

    resp, err := c.client.Post(
        fmt.Sprintf("%s/trade", c.baseURL),
        "application/json",
        bytes.NewBuffer(jsonData),
    )
    if err != nil {
        return 0, err
    }
    defer resp.Body.Close()

    var result struct {
        FilledAmount float64 `json:"filled_amount"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return 0, err
    }
    return result.FilledAmount, nil
}

func (c *Client) GetCurrentPrice(asset string) float64 {
    resp, err := c.client.Get(fmt.Sprintf("%s/price/%s", c.baseURL, asset))
    if err != nil {
        return 0
    }
    defer resp.Body.Close()

    var result struct {
        Price float64 `json:"price"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return 0
    }
    return result.Price
}