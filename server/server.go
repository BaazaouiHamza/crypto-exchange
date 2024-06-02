package server

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strconv"

	"github.com/Baazaouihamza/crypto-exchange/orderbook"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/labstack/echo/v4"
)

const (
	MarketOrder OrderType = "MARKET"

	LimitOrder OrderType = "LIMIT"

	MarketEth Market = "ETH"

	exchangePrivateKey = "4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d"
)

type (
	OrderType string

	Market string

	PlaceOrderRequest struct {
		UserID int64
		Type   OrderType // limit or market
		Bid    bool
		Size   float64
		Price  float64
		Market Market
	}
	Order struct {
		UserID    int64
		ID        int64
		Price     float64
		Size      float64
		Bid       bool
		Timestamp int64
	}

	OrderBookData struct {
		TotalBidVolume float64
		TotalAskVolume float64
		Asks           []*Order
		Bids           []*Order
	}

	MatchedOrders struct {
		UserID int64
		Price  float64
		Size   float64
		ID     int64
	}
)

func StartServer() {
	e := echo.New()
	e.HTTPErrorHandler = httpErrorHandler

	client, err := ethclient.Dial("http://localhost:8545")
	if err != nil {
		log.Fatal(err)
	}

	ex, err := NewExchange(exchangePrivateKey, client)
	if err != nil {
		log.Fatal(err)
	}

	buyerBalance, err := ex.Client.BalanceAt(context.Background(),
		common.HexToAddress("0x28a8746e75304c0780E011BEd21C72cD78cd535E"), nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("buyer balance:", buyerBalance)

	sellerBalance, err := ex.Client.BalanceAt(context.Background(),
		common.HexToAddress("0xACa94ef8bD5ffEE41947b4585a84BdA5a3d3DA6E"), nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("seller balance:", sellerBalance)

	pkStr8 := "829e924fdf021ba3dbbc4225edfece9aca04b929d6e75613329ca6f1d31c0bb4"
	user8 := NewUser(pkStr8, 8)
	ex.Users[user8.ID] = user8

	pkstr7 := "a453611d9419d0e56f499079478fd72c37b251a94bfde4d19872c44cf65386e3"
	user7 := NewUser(pkstr7, 7)
	ex.Users[user7.ID] = user7

	johnPk := "e485d098507f54e7733a205420dfddbe58db035fa577fc294ebd14db90767a52"
	john := NewUser(johnPk, 666)
	ex.Users[john.ID] = john

	johnBalance, err := client.BalanceAt(context.Background(),
		common.HexToAddress("0x3E5e9111Ae8eB78Fe1CC3bb8915d5D461F3Ef9A9"), nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("john balance:", johnBalance)

	e.POST("/order", ex.handlePlaceOrder)

	e.GET("/order/:userID", ex.handleGetOrders)
	e.GET("/book/:market/ask", ex.handleGetBook)
	e.GET("/book/:market", ex.handleGetBook)
	e.GET("/book/:market/bid", ex.handleGetBestBid)
	e.GET("/book/:market/ask", ex.handleGetBestAsk)

	e.DELETE("/order/:id", ex.cancelOrder)

	e.Start(":3000")

	fmt.Println("working")
}

type User struct {
	ID         int64
	PrivateKey *ecdsa.PrivateKey
}

func NewUser(privateKey string, id int64) *User {
	pk, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		panic(err)
	}

	return &User{
		ID:         id,
		PrivateKey: pk,
	}
}

func httpErrorHandler(err error, c echo.Context) {
	fmt.Println(err)
}

type Exchange struct {
	Client *ethclient.Client
	Users  map[int64]*User
	// Orders maps a user to his orders.
	Orders     map[int64][]*orderbook.Order
	PrivateKey *ecdsa.PrivateKey
	orderbooks map[Market]*orderbook.Orderbook
}

func NewExchange(privateKey string, client *ethclient.Client) (*Exchange, error) {
	orderbooks := make(map[Market]*orderbook.Orderbook)
	orderbooks[MarketEth] = orderbook.NewOrderbook()

	pk, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		return nil, err
	}

	return &Exchange{
		Client:     client,
		Users:      make(map[int64]*User),
		Orders:     make(map[int64][]*orderbook.Order),
		PrivateKey: pk,
		orderbooks: orderbooks,
	}, nil
}

func (ex *Exchange) handleGetOrders(c echo.Context) error {
	userID, err := strconv.Atoi(c.Param("userID"))
	if err != nil {
		return err
	}

	orderbooksOrders := ex.Orders[int64(userID)]
	orders := make([]Order, len(orderbooksOrders))

	for i := 0; i < len(orderbooksOrders); i++ {
		order := Order{
			ID:     orderbooksOrders[i].ID,
			UserID: orderbooksOrders[i].UserID,
			// Price:     orderbooksOrders[i].Limit.Price,
			Size:      orderbooksOrders[i].Size,
			Timestamp: orderbooksOrders[i].Timestamp,
			Bid:       orderbooksOrders[i].Bid,
		}
		orders[i] = order
	}
	return c.JSON(http.StatusOK, orders)
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
				UserID:    order.UserID,
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
				UserID:    order.UserID,
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

type PriceResponse struct {
	Price float64
}

func (ex *Exchange) handleGetBestBid(c echo.Context) error {
	market := Market(c.Param("market"))
	ob := ex.orderbooks[market]
	if len(ob.Bids()) == 0 {
		return fmt.Errorf("the bids are empty")
	}
	bestBidPrice := ob.Bids()[0].Price
	pr := PriceResponse{
		Price: bestBidPrice,
	}

	return c.JSON(http.StatusOK, pr)
}

func (ex *Exchange) handleGetBestAsk(c echo.Context) error {
	market := Market(c.Param("market"))
	ob := ex.orderbooks[market]
	if len(ob.Asks()) == 0 {
		return fmt.Errorf("the asks are empty")
	}
	bestAskPrice := ob.Asks()[0].Price
	pr := PriceResponse{
		Price: bestAskPrice,
	}

	return c.JSON(http.StatusOK, pr)
}

func (ex *Exchange) cancelOrder(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	ob := ex.orderbooks[MarketEth]
	ob.CancelOrder(ob.Orders[int64(id)])

	log.Println("order canceled id =>", id)

	return c.JSON(http.StatusOK, map[string]any{"msg": "order deleted"})
}

func (ex *Exchange) handlePlaceMarketOrder(market Market, order *orderbook.Order) ([]orderbook.Match, []*MatchedOrders) {
	ob := ex.orderbooks[market]
	matches := ob.PlaceMarketOrder(order)
	matchedOrders := make([]*MatchedOrders, len(matches))

	isBid := false
	if order.Bid {
		isBid = true
	}

	totalSizeFilled := 0.0
	sumPrice := 0.0
	for i := 0; i < len(matchedOrders); i++ {
		id := matches[i].Bid.ID
		limitUserID := matches[i].Bid.UserID
		if isBid {
			limitUserID = matches[i].Ask.UserID
			id = matches[i].Ask.ID
		}
		matchedOrders[i] = &MatchedOrders{
			UserID: limitUserID,
			Size:   matches[i].SizeFilled,
			Price:  matches[i].Price,
			ID:     id,
		}
		totalSizeFilled += matches[i].SizeFilled
		sumPrice += matches[i].Price
	}

	avgPrice := sumPrice / float64(len(matches))

	log.Printf("filled market order => %d | size: [%.2f] avgPrice: [%.2f]", order.ID, totalSizeFilled, avgPrice)

	return matches, matchedOrders
}

func (ex *Exchange) handlePlaceLimitOrder(market Market, price float64, order *orderbook.Order) error {
	ob := ex.orderbooks[market]
	ob.PlaceLimitOrder(price, order)

	// keep track of the user orders
	ex.Orders[order.UserID] = append(ex.Orders[order.UserID], order)

	log.Printf("new LIMIT order => [%t] price [%.2f] | size [%.2f]", order.Bid, order.Limit.Price, order.Size)

	return nil
}

type PlaceOrderResponse struct {
	OrderID int64
}

func (ex *Exchange) handlePlaceOrder(c echo.Context) error {
	var placeOrderData PlaceOrderRequest

	if err := json.NewDecoder(c.Request().Body).Decode(&placeOrderData); err != nil {
		return err
	}

	market := Market(placeOrderData.Market)
	order := orderbook.NewOrder(placeOrderData.Bid, placeOrderData.Size, placeOrderData.UserID)

	// limit orders
	if placeOrderData.Type == LimitOrder {
		if err := ex.handlePlaceLimitOrder(market, placeOrderData.Price, order); err != nil {
			return err
		}
	}

	// market orders
	if placeOrderData.Type == MarketOrder {
		matches, _ := ex.handlePlaceMarketOrder(market, order)
		if err := ex.handleMatches(matches); err != nil {
			return err
		}
	}

	resp := &PlaceOrderResponse{
		OrderID: order.ID,
	}
	return c.JSON(200, resp)
}

func (ex *Exchange) handleMatches(matches []orderbook.Match) error {
	for _, match := range matches {
		fromUser, ok := ex.Users[match.Ask.UserID]
		if !ok {
			return fmt.Errorf("user not found %d", match.Ask.UserID)
		}

		toUser, ok := ex.Users[match.Bid.UserID]
		if !ok {
			return fmt.Errorf("user not found %d", match.Bid.UserID)
		}
		toAddress := crypto.PubkeyToAddress(toUser.PrivateKey.PublicKey)

		// this is only used for the fees
		// publicKeyECDSA, ok := ex.PrivateKey.Public().(*ecdsa.PublicKey)
		// if !ok {
		// 	return fmt.Errorf("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
		// }

		amount := big.NewInt(int64(match.SizeFilled))

		// transfer => user => exchange
		err := transferETH(ex.Client, fromUser.PrivateKey, toAddress, amount)
		if err != nil {
			return err
		}

	}
	return nil
}

func transferETH(client *ethclient.Client, fromPrivKey *ecdsa.PrivateKey, to common.Address, amount *big.Int) error {
	ctx := context.Background()
	publicKey := fromPrivKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	nonce, err := client.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		return err
	}

	gasLimit := uint64(21000) // in units
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return err
	}
	tx := types.NewTransaction(nonce, to, amount, gasLimit, gasPrice, nil)
	chainID := big.NewInt(1337)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), fromPrivKey)
	if err != nil {
		return err
	}

	err = client.SendTransaction(ctx, signedTx)
	if err != nil {
		return err
	}

	return nil
}
