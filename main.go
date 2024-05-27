package main

import (
	"time"

	"github.com/Baazaouihamza/crypto-exchange/client"
	"github.com/Baazaouihamza/crypto-exchange/server"
)

func main() {
	go server.StartServer()

	time.Sleep(1 * time.Second)

	c := client.NewClient()

	for {
		limitOrderParams := &client.PlaceOrderParams{
			UserID: 8,
			Bid:    false,
			Price:  10_000,
			Size:   500_000,
		}

		_, err := c.PlaceLimitOrder(limitOrderParams)
		if err != nil {
			panic(err)
		}

		otherLimitOrderParams := &client.PlaceOrderParams{
			UserID: 666,
			Bid:    false,
			Price:  9_000,
			Size:   500_000,
		}

		_, err = c.PlaceLimitOrder(otherLimitOrderParams)
		if err != nil {
			panic(err)
		}

		marketOrderParams := &client.PlaceOrderParams{
			UserID: 7,
			Bid:    true,
			Size:   1_000_000,
		}

		_, err = c.PlaceMArketOrder(marketOrderParams)
		if err != nil {
			panic(err)
		}

		time.Sleep(1 * time.Second)
	}

	select {}
}
