package main

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/Baazaouihamza/crypto-exchange/client"
	"github.com/Baazaouihamza/crypto-exchange/server"
)

const (
	maxOrders = 3
)

var (
	tick   = 2 * time.Second
	myAsks = make(map[float64]int64)
	myBids = make(map[float64]int64)
)

func marketOrderPlacer(c *client.Client) {
	ticker := time.NewTicker(5 * time.Second)

	for {
		marketSellOrder := &client.PlaceOrderParams{
			UserID: 666,
			Bid:    true,
			Size:   1000,
		}
		askOrderResp, err := c.PlaceMArketOrder(marketSellOrder)
		if err != nil {
			log.Println(askOrderResp.OrderID)
		}

		marketBuyOrder := &client.PlaceOrderParams{
			UserID: 666,
			Bid:    true,
			Size:   1000,
		}
		bidOrderResp, err := c.PlaceMArketOrder(marketBuyOrder)
		if err != nil {
			log.Println(bidOrderResp.OrderID)
		}
		<-ticker.C
	}
}

func makeMarketSimple(c *client.Client) {
	ticker := time.NewTicker(tick)

	for {
		bestAsk, err := c.GetBestAsk()
		if err != nil {
			log.Println(err)
		}

		bestbid, err := c.GetBestBid()
		if err != nil {
			log.Println(err)
		}

		spread := math.Abs(bestbid - bestAsk)
		fmt.Println("exchange spread", spread)
		// place the bid
		if len(myBids) < 3 {
			bidLimit := &client.PlaceOrderParams{
				UserID: 7,
				Bid:    true,
				Price:  bestbid + 100,
				Size:   1000,
			}
			bidOrderResp, err := c.PlaceLimitOrder(bidLimit)
			if err != nil {
				log.Println(bidOrderResp.OrderID)
			}

			myBids[bidLimit.Price] = bidOrderResp.OrderID
		}
		// place the ask
		if len(myAsks) < 3 {
			askLimit := &client.PlaceOrderParams{
				UserID: 7,
				Bid:    false,
				Price:  bestAsk - 100,
				Size:   1000,
			}
			askOrderResp, err := c.PlaceLimitOrder(askLimit)
			if err != nil {
				log.Println(askOrderResp.OrderID)
			}

			myAsks[askLimit.Price] = askOrderResp.OrderID
		}

		fmt.Println("best ask price", bestAsk)
		fmt.Println("best bid price", bestbid)

		<-ticker.C
	}
}

func seedMarket(c *client.Client) error {
	ask := &client.PlaceOrderParams{
		UserID: 8,
		Bid:    false,
		Price:  10_000,
		Size:   1_000_000,
	}

	bid := &client.PlaceOrderParams{
		UserID: 8,
		Bid:    true,
		Price:  9_000,
		Size:   1_000_000,
	}

	_, err := c.PlaceLimitOrder(ask)
	if err != nil {
		return err
	}

	_, err = c.PlaceLimitOrder(bid)
	if err != nil {
		return err
	}

	return nil

}

func main() {
	go server.StartServer()

	time.Sleep(1 * time.Second)

	c := client.NewClient()

	if err := seedMarket(c); err != nil {
		panic(err)
	}

	go makeMarketSimple(c)
	time.Sleep(1 * time.Second)
	marketOrderPlacer(c)
	// limitOrderParams := &client.PlaceOrderParams{
	// 	UserID: 8,
	// 	Bid:    false,
	// 	Price:  10_000,
	// 	Size:   5_000_000,
	// }

	// _, err := c.PlaceLimitOrder(limitOrderParams)
	// if err != nil {
	// 	panic(err)
	// }

	// otherLimitOrderParams := &client.PlaceOrderParams{
	// 	UserID: 666,
	// 	Bid:    false,
	// 	Price:  9_000,
	// 	Size:   500_000,
	// }

	// _, err = c.PlaceLimitOrder(otherLimitOrderParams)
	// if err != nil {
	// 	panic(err)
	// }

	// buyLimitOrder := &client.PlaceOrderParams{
	// 	UserID: 666,
	// 	Bid:    true,
	// 	Price:  11_000,
	// 	Size:   500_000,
	// }

	// _, err = c.PlaceLimitOrder(buyLimitOrder)
	// if err != nil {
	// 	panic(err)
	// }

	// marketOrderParams := &client.PlaceOrderParams{
	// 	UserID: 7,
	// 	Bid:    true,
	// 	Size:   1_000_000,
	// }

	// _, err = c.PlaceMArketOrder(marketOrderParams)
	// if err != nil {
	// 	panic(err)
	// }

	// bestBidPrice, err := c.GetBestBid()
	// if err != nil {
	// 	panic(err)
	// }

	// fmt.Println("best bid price", bestBidPrice)

	// bestAskPrice, err := c.GetBestAsk()
	// if err != nil {
	// 	panic(err)
	// }

	// fmt.Println("best bid price", bestAskPrice)

	// time.Sleep(1 * time.Second)

	select {}
}
