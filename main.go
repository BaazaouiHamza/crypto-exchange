package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/Baazaouihamza/crypto-exchange/orderbook"
	"github.com/labstack/echo/v4"
)

func main() {
	e := echo.New()
	ex := NewExchange()

	e.GET("/book/:market", ex.handleGetBook)
	e.POST("/order", ex.handlePlaceOrder)
	e.DELETE("/order/:id", ex.cancelOrder)

	e.Start(":3000")

	fmt.Println("working")
}

type OrderType string

const (
	MarketOrder OrderType = "MARKET"
	LimitOrder  OrderType = "LIMIT"
)

type Market string

const (
	MarketEth Market = "ETH"
)

type Exchange struct {
	orderbooks map[Market]*orderbook.Orderbook
}

func NewExchange() *Exchange {
	orderbooks := make(map[Market]*orderbook.Orderbook)
	orderbooks[MarketEth] = orderbook.NewOrderbook()

	return &Exchange{
		orderbooks: orderbooks,
	}
}

type PlaceOrderRequest struct {
	Type   OrderType // limit or market
	Bid    bool
	Size   float64
	Price  float64
	Market Market
}
type Order struct {
	ID        int64
	Price     float64
	Size      float64
	Bid       bool
	Timestamp int64
}

type OrderBookData struct {
	TotalBidVolume float64
	TotalAskVolume float64
	Asks           []*Order
	Bids           []*Order
}

func (ex *Exchange) handleGetBook(c echo.Context) error {
	market := Market(c.Param("market"))
	ob, ok := ex.orderbooks[market]

	if !ok {
		return c.JSON(http.StatusBadRequest, map[string]any{"msg": "market not found"})
	}

	orderBookData := OrderBookData{
		TotalBidVolume: ob.BidTotalVolume(),
		TotalAskVolume: ob.AskTotalVolume(),
		Asks:           []*Order{},
		Bids:           []*Order{},
	}
	for _, limits := range ob.Asks() {
		for _, order := range limits.Orders {
			o := Order{
				ID:        order.ID,
				Price:     limits.Price,
				Size:      order.Size,
				Bid:       order.Bid,
				Timestamp: order.Timestamp,
			}
			orderBookData.Asks = append(orderBookData.Asks, &o)
		}
	}

	for _, limits := range ob.Bids() {
		for _, order := range limits.Orders {
			o := Order{
				ID:        order.ID,
				Price:     limits.Price,
				Size:      order.Size,
				Bid:       order.Bid,
				Timestamp: order.Timestamp,
			}
			orderBookData.Bids = append(orderBookData.Bids, &o)
		}
	}
	return c.JSON(http.StatusOK, orderBookData)
}

func (ex *Exchange) cancelOrder(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	ob := ex.orderbooks[MarketEth]
	orderCancelled := false
	for _, limit := range ob.Asks() {
		for _, order := range limit.Orders {
			if order.ID == int64(id) {
				ob.CancelOrder(order)
				orderCancelled = true
			}
			if orderCancelled {
				return c.JSON(http.StatusOK, map[string]any{"msg": "order canceled"})
			}
		}
	}

	for _, limit := range ob.Bids() {
		for _, order := range limit.Orders {
			if order.ID == int64(id) {
				ob.CancelOrder(order)
				orderCancelled = true
			}

			if orderCancelled {
				return c.JSON(http.StatusOK, map[string]any{"msg": "order canceled"})
			}
		}
	}

	return nil
}

func (ex *Exchange) handlePlaceOrder(c echo.Context) error {
	var placeOrderData PlaceOrderRequest

	if err := json.NewDecoder(c.Request().Body).Decode(&placeOrderData); err != nil {
		return err
	}

	market := Market(placeOrderData.Market)
	ob := ex.orderbooks[market]
	order := orderbook.NewOrder(placeOrderData.Bid, placeOrderData.Size)

	if placeOrderData.Type == LimitOrder {
		ob.PlaceLimitOrder(placeOrderData.Price, order)
		return c.JSON(200, map[string]any{"msg": "limit order placed"})
	}

	if placeOrderData.Type == MarketOrder {
		matches := ob.PlaceMarketOrder(order)
		return c.JSON(200, map[string]any{"matches": len(matches)})
	}

	return nil
}
