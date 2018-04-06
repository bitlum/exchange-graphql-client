package client

import (
	"encoding/json"
	"errors"

	"github.com/bitlum/macaroon-application-auth"
	"github.com/shopspring/decimal"
	gomacaroon "gopkg.in/macaroon.v2"
)

// Client is the http://exchange.bitlum.io exchange client which wraps the raw GraphQL API.
type Client struct {
	core
}

// NewClient creates new client for bitlum exchange on specified URL
// with either JWT token or hex encoded binary macaroon.
// It returns an error if the macaroon can not be decoded.
func NewClient(url string, macaroon string, jwt string) (*Client, error) {
	var m *gomacaroon.Macaroon

	if macaroon != "" {
		var err error
		m, err = auth.DecodeMacaroon(macaroon)
		if err != nil {
			return nil, err
		}
	}
	return &Client{
		core: &graphQLCore{
			url:      url,
			macaroon: m,
			jwt:      jwt,
		},
	}, nil
}

// Markets return markets supported by exchange
// TODO: the list should be requested from the backend
func (c *Client) SupportedMarkets() []string {
	return []string{
		"BTCETH",
		"BTCBCH",
		"BTCDASH",
		"BTCLTC",
	}
}

// Me is a structure to hold the result of Me query
type Me struct {
	// ID is User's ID
	ID string
	// Email is an email used during the registration of the user
	// or an email of the user who requested macaroon token
	Email string
}

// Me returns user info on behalf which all
// exchange operations are performing.
func (c *Client) Me() (Me, error) {
	var req request

	req.Query = `
		query Me {
			me {
			  id
			  email
			}
		}
	`

	resp := struct {
		responseBase
		Data struct {
			Me Me
		}
	}{}

	respJSON, err := c.do(true, req)
	if err != nil {
		return Me{}, errors.New("failed to do request: " + err.Error())
	}

	if err := json.Unmarshal(respJSON, &resp); err != nil {
		return Me{}, errors.New("failed to json.Unmarshal resp: " +
			err.Error())
	}

	if err := resp.Error(); err != nil {
		return Me{}, errors.New("exchange error: " + err.Error())
	}

	return resp.Data.Me, nil
}

// UserID returns exchange user ID on behalf which all
// exchange operations are performing.
func (c *Client) UserID() (string, error) {
	var req request

	req.Query = `
		query Me {
			me {
			  id
			}
		}
	`

	resp := struct {
		responseBase
		Data struct {
			User struct {
				ID string
			} `json:"me"`
		}
	}{}

	respJSON, err := c.do(true, req)
	if err != nil {
		return "", errors.New("failed to do request: " + err.Error())
	}

	if err := json.Unmarshal(respJSON, &resp); err != nil {
		return "", errors.New("failed to json.Unmarshal resp: " +
			err.Error())
	}

	if err := resp.Error(); err != nil {
		return "", errors.New("exchange error: " + err.Error())
	}

	return resp.Data.User.ID, nil
}

// Ticker is stock exchange market ticker with information about prices.
// TODO: obsolete, should be removed as soon as ZigZag switch to use Market instead
type Ticker struct {
	// Market is a stock exchange market, e.g. BTCETH.
	Market string
	// Last is the price of last order proceed by the
	// exchange engine.
	Last decimal.Decimal
	// ChangeLast is a change of the last price.
	ChangeLast decimal.Decimal
}

// tickersRequestVariables is a query variables used in request
// in client Tickers method.
// TODO: obsolete, should be removed as soon as ZigZag switch to use Market instead
type tickersRequestVariables struct {
	Markets []string `json:"markets"`
}

// Ticker returns summary information about last 24 hours of each market
// TODO: obsolete, should be removed as soon as ZigZag switch to use Market instead
func (c *Client) Tickers(markets []string) ([]Ticker, error) {

	if len(markets) == 0 {
		return nil, errors.New("not empty markets expected")
	}

	var req request

	req.Query = `
		query GetMarketInfo($markets: [Market!]) {
			markets(markets: $markets) {
				market
				last
				changeLast
			}
		}
	`

	req.Variables = tickersRequestVariables{markets}

	respJSON, err := c.do(false, req)
	if err != nil {
		return nil, errors.New("failed to do request: " + err.Error())
	}

	resp := struct {
		responseBase
		Data struct {
			Markets []Ticker
		}
	}{}

	if err := json.Unmarshal(respJSON, &resp); err != nil {
		return nil, errors.New("failed to json.Unmarshal resp: " +
			err.Error())
	}

	if err := resp.Error(); err != nil {
		return nil, errors.New("exchange error: " + err.Error())
	}

	return resp.Data.Markets, nil
}

// Ask is an order to sell stock (right asset in market) with given
// maximum price and within given volume. Price is given in amount of
// money (left asset in market). E.g. ask in BTCETH market is an
// order to sell ETH for BTC.
// https://www.investopedia.com/terms/b/bid-and-asked.asp
type Ask struct {
	Price  decimal.Decimal
	Volume decimal.Decimal
}

// Bid is an order to buy stock (right asset in market) with given
// maximum price and within given volume. Price is given in amount of
// money (left asset in market). E.g. bid in BTCETH market is an
// order to buy ETH using BTC.
// https://www.investopedia.com/terms/b/bid-and-asked.asp
type Bid struct {
	Price  decimal.Decimal
	Volume decimal.Decimal
}

// Depth is limited lists of asks and bids in benefit order.
type Depth struct {
	// Top asks by increasing price
	Asks []Ask
	// Top bids by decreasing price.
	Bids []Bid
}

// depthRequestVariables is a query variables used in request
// in client Depth method.
type depthRequestVariables struct {
	Market   string  `json:"market"`
	Limit    uint    `json:"limit"`
	Interval float64 `json:"interval"`
}

// Depth returns limited lists of asks and bids in benefit order.
func (c *Client) Depth(market string, limit uint, interval float64) (Depth, error) {

	var (
		depth Depth
		req   request
	)

	req.Query = `
	query GetBestAskBid($market: Market!, $limit: Int, $interval: Float) {
  			depth(market: $market, limit: $limit, interval: $interval) {
    			asks {
      				price
      				volume
    			}
				bids {
					price
      				volume
    			}
			}
		}
	`

	req.Variables = depthRequestVariables{market, limit, interval}

	resp := struct {
		responseBase
		Data struct {
			Depth Depth
		}
	}{}

	respJSON, err := c.do(false, req)
	if err != nil {
		return depth, errors.New("failed to do request: " + err.Error())
	}

	if err := json.Unmarshal(respJSON, &resp); err != nil {
		return depth, errors.New("failed to json.Unmarshal resp: " +
			err.Error())
	}

	if err := resp.Error(); err != nil {
		return depth, errors.New("exchange error: " + err.Error())
	}

	return resp.Data.Depth, nil
}

// depositRequestVariables is a query variables used in request
// in client Deposits method.
type depositRequestVariables struct {
	Assets []string `json:"assets"`
	Offset int64    `json:"offset"`
	Limit  int64    `json:"limit"`
}

// Deposit represents an account deposit.
type Deposit struct {
	// PaymentID is system specific withdraw operation ID.
	// In blockchain it is transaction ID, in lightning network
	// it is payment hash.
	PaymentID string

	// PaymentSystem is a payment system in which deposit payment was
	// occurred,
	PaymentType string

	// Change is an amount on which balance has been changed.
	Change decimal.Decimal

	// Time when deposit was registered.
	Time float64
}

// Deposits returns account deposits in given offset and limit
// from account change history.
func (c *Client) Deposits(asset string, offset,
limit int64) ([]Deposit, error) {

	var req request

	req.Query = `
		query GetBalanceUpdates($assets: [Asset!]!, $offset: Int!,
$limit: Int!) {
  			balanceUpdateRecords(assets: $assets, offset: $offset,
				recordTypes: deposit, limit: $limit) {
    			... on Deposit {
      				change
      				time
      				paymentID
      				paymentType
    			}
  			}
		}
	`

	req.Variables = depositRequestVariables{
		Assets: []string{asset},
		Offset: offset,
		Limit:  limit,
	}

	resp := struct {
		responseBase
		Data struct {
			Deposits []Deposit `json:"balanceUpdateRecords"`
		}
	}{}

	respJSON, err := c.do(true, req)
	if err != nil {
		return nil, errors.New("failed to do request: " + err.Error())
	}

	if err := json.Unmarshal(respJSON, &resp); err != nil {
		return nil, errors.New("failed to json.Unmarshal resp: " +
			err.Error())
	}

	if err := resp.Error(); err != nil {
		return nil, errors.New("exchange error: " + err.Error())
	}

	return resp.Data.Deposits, nil
}

// Order is an exchange order to buy or sell stock. Market contains
// two currencies: left one is money and right - stock. For example
// Market{BTC,LTC} means that BTC is a money and LTC - stock.
type Order struct {
	// ID is a exchange specific order ID
	ID int64

	// Status is order status: pending, finished or canceled
	Status string

	// Either amount of money or stock depending on direction
	// of order: buy or sell. Now only buy direction used so this is
	// always should be stock amount.
	Amount decimal.Decimal

	// Price of 1 stock in money currency.
	Price decimal.Decimal

	// DealMoney is the amount of money which were involved in the
	// order.
	DealMoney decimal.Decimal

	// DealStock is the amount of stock which were involved in the
	// order.
	DealStock decimal.Decimal

	// Left is the amount of funds left in the market without being
	// handled.
	Left decimal.Decimal
}

// orderRequestVariables is a query variables used in request
// in client Order method.
type orderRequestVariables struct {
	ID int64 `json:"id"`
}

// Order returns order with specified id
func (c *Client) Order(id int64) (Order, error) {

	var req request

	req.Query = `
		query GetOrder($id: Int!) {
  			order(id: $id) {
				id
    			status
				dealStock
				dealMoney
				amount
				price
  			}
		}
	`

	req.Variables = orderRequestVariables{id}

	resp := struct {
		responseBase
		Data struct {
			Order Order
		}
	}{}

	respJSON, err := c.do(true, req)
	if err != nil {
		return Order{}, errors.New("failed to do request: " + err.
			Error())
	}

	if err := json.Unmarshal(respJSON, &resp); err != nil {
		return Order{}, errors.New("failed to json.Unmarshal resp: " +
			err.Error())
	}

	if err := resp.Error(); err != nil {
		return Order{}, errors.New("exchange error: " + err.Error())
	}

	return resp.Data.Order, nil
}

// createOrderRequestVariables is a query variables used in request
// in client CreateOrder method.
type createOrderRequestVariables struct {
	Market string          `json:"market"`
	Amount decimal.Decimal `json:"amount"`
	Side   string          `json:"side"`
}

func (c *Client) createOrder(market string, amount decimal.Decimal, side string) (Order, error) {

	var req request

	req.Query = `
	mutation CreateMarketOrder($market: Market!, $amount: String!, $side: MarketSide!) {
  			createMarketOrder(amount: $amount, market: $market, side: $side) {
    			id
    			status
    			amount
				price
    			dealStock
				dealMoney
    			left
  			}
		}
	`

	req.Variables = createOrderRequestVariables{
		Market: market,
		Amount: amount,
		Side:   side,
	}

	resp := struct {
		responseBase
		Data struct {
			Order Order `json:"createMarketOrder"`
		}
	}{}

	respJSON, err := c.do(true, req)
	if err != nil {
		return Order{}, errors.New("failed to do request: " + err.
			Error())
	}

	if err := json.Unmarshal(respJSON, &resp); err != nil {
		return Order{}, errors.New("failed to json.Unmarshal resp: " +
			err.Error())
	}

	if err := resp.Error(); err != nil {
		return Order{}, errors.New("exchange error: " + err.Error())
	}

	return resp.Data.Order, nil
}

// CreateOrder is an alias of CreateOrderBid
func (c *Client) CreateOrder(market string,
	amount decimal.Decimal) (Order, error) {
	return c.CreateOrderBid(market, amount)
}

// CreateOrderAsk creates ask order on market. Asc order means that
// left asset of the market is used to sell right asset. E.g. in
// market BTCETH this method creates an order to sell ETH for BTC.
func (c *Client) CreateOrderAsk(market string,
	amount decimal.Decimal) (Order, error) {
	return c.createOrder(market, amount, "ask")
}

// CreateOrderBid creates bid order on market. Bid order means that
// left asset of the market is used to buy right asset.
// E.g. in market BTCETH this method creates an order to buy ETH using BTC.
func (c *Client) CreateOrderBid(market string,
	amount decimal.Decimal) (Order, error) {
	return c.createOrder(market, amount, "bid")

}

// Withdrawal represents an account withdraw.
type Withdrawal struct {
	// PaymentID is system specific withdraw operation ID.
	// In blockchain it is transaction ID, in lightning network
	// it is payment hash.
	PaymentID string

	// PaymentAddr is the address of the payment receiver in
	// blockchain system. Meaningless in lightning network.
	PaymentAddr string

	// Change is an amount on which balance has been changed.
	Change decimal.Decimal
}

// withdrawRequestVariables is a query variables used in request
// in client Withdraw method.
type withdrawRequestVariables struct {
	Asset   string          `json:"asset"`
	Amount  decimal.Decimal `json:"amount"`
	Address string          `json:"address"`
}

// Withdraw withdraws funds from exchange using blockchain to the
// specified address.
func (c *Client) Withdraw(asset string, amount decimal.Decimal,
	address string) (Withdrawal, error) {

	var req request

	req.Query = `
		mutation Withdraw($asset: Asset!, $amount: String!,
$address: String!) {
  			withdrawWithBlockchain(
    			asset: $asset,
    			amount: $amount,
    			address: $address) {
    				...on Withdrawal {
      					paymentID
      					paymentAddr
						change
    				}
  			}
		}
	`

	req.Variables = withdrawRequestVariables{
		Asset:   asset,
		Amount:  amount,
		Address: address,
	}

	resp := struct {
		responseBase
		Data struct {
			Withdrawal Withdrawal `json:"withdrawWithBlockchain"`
		}
	}{}

	respJSON, err := c.do(true, req)
	if err != nil {
		return Withdrawal{},
			errors.New("failed to do request: " + err.Error())
	}

	if err := json.Unmarshal(respJSON, &resp); err != nil {
		return Withdrawal{},
			errors.New("failed to json.Unmarshal resp: " + err.Error())
	}

	if err := resp.Error(); err != nil {
		return Withdrawal{},
			errors.New("exchange error: " + err.Error())
	}

	return resp.Data.Withdrawal, nil
}

// reachableRequestVariables is a query variables used in request
// in client LightningNodeReachable method.
type reachableRequestVariables struct {
	Asset          string `json:"asset"`
	IdentityPubKey string `json:"identityKey"`
}

// LightningNodeReachable checks that lightning network node with
// specified identity public key can be reached from exchange
// lightning node
func (c *Client) LightningNodeReachable(asset string,
	identityPubKey string) (bool, error) {

	var req request

	req.Query = `
		query CheckReachable($asset: Asset!, $identityKey: String!) {
  			checkReachable(asset: $asset, identityKey: $identityKey)
		}
	`

	req.Variables = reachableRequestVariables{
		Asset:          asset,
		IdentityPubKey: identityPubKey,
	}

	resp := struct {
		responseBase
		Data struct {
			Reachable bool `json:"checkReachable"`
		}
	}{}

	respJSON, err := c.do(false, req)
	if err != nil {
		return false,
			errors.New("failed to do request: " + err.Error())
	}

	if err := json.Unmarshal(respJSON, &resp); err != nil {
		return false,
			errors.New("failed to json.Unmarshal resp: " + err.Error())
	}

	if err := resp.Error(); err != nil {
		return false,
			errors.New("exchange error: " + err.Error())
	}

	return resp.Data.Reachable, nil
}

// LightningNodeInfo is a lightning network node info.
type LightningNodeInfo struct {
	Host      string
	Port      string
	MinAmount decimal.Decimal
	MaxAmount decimal.Decimal

	// IdentityPubkey is the identity pubkey of the current node.
	IdentityPubkey string

	// Alias if applicable, the alias of the current node, e.g. "bob".
	Alias string

	// NumPendingChannels is the number of pending channels.
	NumPendingChannels uint32

	// NumActiveChannels is the number of active channels.
	NumActiveChannels uint32

	// NumPeers is the number of peers.
	NumPeers uint32

	// BlockHeight is the node's current view of the height of the best
	// block.
	BlockHeight uint32

	// BlockHash is the node's current view of the hash of the best
	// block.
	BlockHash string

	// SyncedToChain means whether the wallet's view is synced to the
	// main chain.
	SyncedToChain bool

	// Testnet means whether the current node is connected to testnet
	Testnet bool

	// Chains is a list of active chains the node is connected to
	Chains []string
}

// nodeInfoRequestVariables is a query variables used in request
// in client LightningNodeInfo method.
type nodeInfoRequestVariables struct {
	Asset string `json:"asset"`
}

// LightningNodeInfo returns exchange lightning network node for
// specified asset info
func (c *Client) LightningNodeInfo(asset string) (LightningNodeInfo,
	error) {

	var req request
	req.Query = `
		query LightningNodeInfo($asset: Asset!) {
			lightningInfo(asset: $asset)   {
    		host
			port
			minAmount
    		maxAmount
    		identityPubkey
    		alias
    		numPendingChannels
    		numActiveChannels
    		numPeers
    		blockHeight
    		blockHash
    		syncedToChain
			testnet
			chains
		  }
		}
	`

	req.Variables = nodeInfoRequestVariables{asset}

	resp := struct {
		responseBase
		Data struct {
			Info LightningNodeInfo `json:"lightningInfo"`
		}
	}{}

	respJSON, err := c.do(false, req)
	if err != nil {
		return LightningNodeInfo{},
			errors.New("failed to do request: " + err.Error())
	}

	if err := json.Unmarshal(respJSON, &resp); err != nil {
		return LightningNodeInfo{},
			errors.New("failed to json.Unmarshal resp: " + err.Error())
	}

	if err := resp.Error(); err != nil {
		return LightningNodeInfo{},
			errors.New("exchange error: " + err.Error())
	}

	return resp.Data.Info, nil
}

// lightningCreateRequestVariables is a query variables used in request
// in client LightningCreateInvoice method.
type lightningCreateRequestVariables struct {
	Asset  string          `json:"asset"`
	Amount decimal.Decimal `json:"amount"`
}

// LightningCreateInvoice creates lightning network invoice to pay
// for deposit funds in exchange.
func (c *Client) LightningCreateInvoice(asset string,
	amount decimal.Decimal) (string, error) {

	var req request

	req.Query = `
		mutation GenerateLightningInvoice($asset: Asset!, 
$amount: String!) {
  			generateLightningInvoice(asset: $asset, amount: $amount)
		}
	`

	req.Variables = lightningCreateRequestVariables{
		Asset:  asset,
		Amount: amount,
	}

	resp := struct {
		responseBase
		Data struct {
			Invoice string `json:"generateLightningInvoice"`
		}
	}{}

	respJSON, err := c.do(true, req)
	if err != nil {
		return "", errors.New("failed to do request: " + err.Error())
	}

	if err := json.Unmarshal(respJSON, &resp); err != nil {
		return "", errors.New("failed to json.Unmarshal resp: " +
			err.Error())
	}

	if err := resp.Error(); err != nil {
		return "", errors.New("exchange error: " + err.Error())
	}

	return resp.Data.Invoice, nil
}

// lightningWithdrawRequestError is a query variables used in request
// in client LightningWithdraw method.
type lightningWithdrawRequestError struct {
	Asset   string `json:"asset"`
	Invoice string `json:"invoice"`
}

// LightningWithdraw withdraws funds from exchange with lightning network
// using specified invoice.
func (c *Client) LightningWithdraw(asset string,
	invoice string) (Withdrawal, error) {

	var req request

	req.Query = `
		mutation Withdraw($asset: Asset!, $invoice: String!) {
  			withdrawWithLightning(
    			asset: $asset,
    			invoice: $invoice) {
    				...on Withdrawal {
      					paymentID
    				}
  			}
		}
	`

	req.Variables = lightningWithdrawRequestError{
		Asset:   asset,
		Invoice: invoice,
	}

	resp := struct {
		responseBase
		Data struct {
			Withdrawal Withdrawal `json:"withdrawWithLightning"`
		}
	}{}

	respJSON, err := c.do(true, req)
	if err != nil {
		return Withdrawal{},
			errors.New("failed to do request: " + err.Error())
	}

	if err := json.Unmarshal(respJSON, &resp); err != nil {
		return Withdrawal{},
			errors.New("failed to json.Unmarshal resp: " + err.Error())
	}

	if err := resp.Error(); err != nil {
		return Withdrawal{},
			errors.New("exchange error: " + err.Error())
	}

	return resp.Data.Withdrawal, nil
}

// AccountsRequest is a query variables used in request user's account
// balances
type AccountsRequest struct {
	Assets []string `json:"assets"`
}

// Account struct describes the current balance of the exchange user by
// every asset he owns
type Account struct {
	// A name of asset. Currently: BTC, BCH, ETH, LTC, DASH
	Asset string

	// Address on which funds has to be sent to be deposited on the account.
	// If returned null than address has to be created first
	Address string

	// Available is the funds which can be used in trading.
	Available decimal.Decimal

	// Estimation is the estimated number of dollars which corresponds to this asset.
	Estimation decimal.Decimal

	// Freezed is the funds which currently occupied in trades.
	Freezed decimal.Decimal

	// Pending is the information which related to the blockchain, for example: how much funds are waiting to be approved, before to be enrolled in the account.
	Pending decimal.Decimal
}

// Accounts shows balances for the assets owned by loggedin user
// using specified invoice.
func (c *Client) Accounts(assets []string) ([]Account, error) {

	var req request

	req.Query = `
		query Accounts($assets: [Asset!]!) {
  			accounts( assets: $assets) {
				asset, address, available, estimation, freezed
  			}
		}
	`

	req.Variables = AccountsRequest{
		Assets: assets,
	}

	resp := struct {
		responseBase
		Data struct {
			Accounts []Account `json:"accounts"`
		}
	}{}

	respJSON, err := c.do(true, req)
	if err != nil {
		return []Account{},
			errors.New("failed to do request: " + err.Error())
	}

	if err := json.Unmarshal(respJSON, &resp); err != nil {
		return []Account{},
			errors.New("failed to json.Unmarshal resp: " + err.Error())
	}

	if err := resp.Error(); err != nil {
		return resp.Data.Accounts,
			errors.New("exchange error: " + err.Error())
	}

	return resp.Data.Accounts, nil
}

// IssueApiToken exchanges JWT User's token with the third-party applciations token.
// The application token is hex-encoded binary representation of the Macaroon bearer
func (c *Client) IssueApiToken() (string, error) {

	var req request

	req.Query = `
		query { issueApiToken }
	`

	resp := struct {
		responseBase
		Data struct {
			IssueApiToken string `json:"issueApiToken"`
		}
	}{}

	respJSON, err := c.do(true, req)
	if err != nil {
		return "",
			errors.New("unable to do request: " + err.Error())
	}

	if err := json.Unmarshal(respJSON, &resp); err != nil {
		return "",
			errors.New("unable to json.Unmarshal resp: " + err.Error())
	}

	if err := resp.Error(); err != nil {
		return resp.Data.IssueApiToken,
			errors.New("exchange error: " + err.Error())
	}

	return resp.Data.IssueApiToken, nil
}

// MarketsRequest is a query variables used in request
// markets statuses
type MarketsRequest struct {
	Markets []string `json:"markets"`
	Period  int32    `json:"period"`
}

// MarketStatus represent the information about market the market by the given period of time.
type MarketStatus struct {
	// Market is a pair of assets to be exchanged with each other
	Market string

	// Stock is a right pair of a market. It is an asset to be sold in case of ask
	// order and to be bought if a bid order
	Stock string

	// Money is a left pair of a market. It is an asset to be bought in case of ask
	// order and to be sold if a bid order
	Money string

	// The opening price is the price at which a stok first trades upon the opening
	// of an exchange on a given period
	Open decimal.Decimal

	// The closing price is the final price at which a stok is traded on a given period
	Close decimal.Decimal

	// The high price is the highest price at which a stok was traded within a given period
	High decimal.Decimal

	// The last price is the price at which a most recent order was executed upon the given period
	Last decimal.Decimal

	// The low price is the lowest price at which a stok was traded within a given period
	Low decimal.Decimal

	// Volume is the amount of stock traded during a given period of time. The volume is estimated in market money.
	Volume decimal.Decimal

	// ChangeLast is the differnce between Open and Last values measured in percents
	ChangeLast decimal.Decimal

	// ChangeHigh is the differnce between Open and Last values measured in percents
	ChangeHigh decimal.Decimal

	// ChangeLow is the differnce between Open and Last values measured in percents
	ChangeLow decimal.Decimal

	// BestAsk is the lowest price the stock may be bought right now
	BestAsk decimal.Decimal

	// BestBid is the highes price the stock may be sold right now
	BestBid decimal.Decimal
}

// Markets reporst the statuses (see MarketStatus) of the markets for the given period
func (c *Client) Markets(markets []string, period int32) ([]MarketStatus, error) {

	var req request

	req.Query = `
	query Markets($markets: [Market!]!, $period: Int) {
		markets (markets: $markets, period: $period){
				market
				stock
				money
				open
				close
				high
				last
				low
				volume
				changeLast
				changeHigh
				changeLow
				bestAsk
				bestBid
  			}
		}
	`

	req.Variables = MarketsRequest{
		markets,
		period,
	}

	respJSON, err := c.do(false, req)
	if err != nil {
		return []MarketStatus{},
			errors.New("failed to do request: " + err.Error())
	}

	resp := struct {
		responseBase
		Data struct {
			Markets []MarketStatus `json:"markets"`
		}
	}{}
	if err := json.Unmarshal(respJSON, &resp); err != nil {
		return []MarketStatus{},
			errors.New("failed to json.Unmarshal resp: " + err.Error())
	}

	if err := resp.Error(); err != nil {
		return resp.Data.Markets,
			errors.New("exchange error: " + err.Error())
	}

	return resp.Data.Markets, nil
}

// DealsRequest is a query variable used in the Deal query
type DealsRequest struct {
	// List of markets for which to return completed deals
	Markets []string `json:"markets"`
	// A number of the deals to return
	Limit int32 `json:"limit"`
}

// MarketDeal is a structure to hold result of the Deal query
type MarketDeal struct {
	// ID of a deal
	ID int32

	// Market the deal was closed on
	Market string

	// A time of a deal
	Time float32

	// Total amount of money spent to close the deal
	Amount decimal.Decimal

	// A price of stocks used to close the deal
	Price decimal.Decimal

	// Type is may be "ask" or "bid"
	Type string
}

// Deals returns the result of orders matching with other users's orders. When users opposite orders have the same ask and bid prices their orderders considired to be appropriate for matching , the result of this matching is called deal.
func (c *Client) Deals(markets []string, limit int32) ([]MarketDeal, error) {
	var req request

	req.Query = `
	query Deals ($markets: [Market!]!, $limit: Int) {
		deals (markets: $markets, limit: $limit){
			    id
				market
				time
				amount
				price
				type
  			}
		}
	`

	req.Variables = DealsRequest{
		markets,
		limit,
	}

	respJSON, err := c.do(false, req)
	if err != nil {
		return []MarketDeal{},
			errors.New("failed to do request: " + err.Error())
	}

	resp := struct {
		responseBase
		Data struct {
			Deals []MarketDeal `json:"deals"`
		}
	}{}
	if err := json.Unmarshal(respJSON, &resp); err != nil {
		return []MarketDeal{},
			errors.New("failed to json.Unmarshal resp: " + err.Error())
	}

	if err := resp.Error(); err != nil {
		return resp.Data.Deals,
			errors.New("exchange error: " + err.Error())
	}

	return resp.Data.Deals, nil
}
