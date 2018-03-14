package client

import (
	"encoding/json"
	"errors"

	"github.com/shopspring/decimal"
)

const exchangeID = "bitlum"

// Client is the http://exchange.bitlum.io exchange
type Client struct {
	backend
}

// NewExchangeClient create new bitlum exchange with specified URL and
// authorization token.
func NewClient(url string, authToken string) *Client {
	return &Client{
		backend: &graphqlBackend{
			url:       url,
			authToken: authToken,
		},
	}
}

// ID returns exchange ID
func (c *Client) ID() string {
	return exchangeID
}

// Markets return markets supported by exchange
func (c *Client) Markets() []string {
	return []string{
		"BTCETH",
		"BTCBCH",
		"BTCDASH",
		"BTCLTC",
	}
}

// UserID returns exchange user ID, user on behalf which all
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

	respJSON, err := c.do(req)
	if err != nil {
		return "", errors.New("failed to do request: " + err.Error())
	}

	if err := json.Unmarshal(respJSON, &resp); err != nil {
		return "", errors.New("failed to json.Unmarshal resp: " +
			err.Error())
	}

	if err := resp.Error(); err != nil {
		return "", errors.New("bitlum server-side error: " +
			err.Error())
	}

	return resp.Data.User.ID, nil
}

type Ticker struct {
	Market     string
	Last       decimal.Decimal
	ChangeLast decimal.Decimal
}

// tickersRequestVariables is a query variables used in request
// in bitlum Tickers method.
type tickersRequestVariables struct {
	Markets []string `json:"markets"`
}

// Ticker returns summary information about last 24 hours of each market
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

	respJSON, err := c.do(req)
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
		return nil, errors.New("bitlum server-side error: " +
			err.Error())
	}

	return resp.Data.Markets, nil
}

type Ask struct {
	Price  decimal.Decimal
	Volume decimal.Decimal
}

type Bid struct {
	Price  decimal.Decimal
	Volume decimal.Decimal
}

// Depth is limited lists of asks and bids in benefit order.
// Limit depends on the specific exchange
type Depth struct {
	Asks []Ask
	Bids []Bid
}

// depthRequestVariables is a query variables used in request
// in bitlum Depth method.
type depthRequestVariables struct {
	Market string `json:"market"`
}

// Depth returns limited lists of asks and bids in benefit order.
func (c *Client) Depth(market string) (Depth, error) {

	var (
		depth Depth
		req   request
	)

	req.Query = `
		query GetBestAskBid($market: Market!) {
  			depth(market: $market, limit: 50, interval: 0.00000001) {
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

	req.Variables = depthRequestVariables{market}

	resp := struct {
		responseBase
		Data struct {
			Depth Depth
		}
	}{}

	respJSON, err := c.do(req)
	if err != nil {
		return depth, errors.New("failed to do request: " + err.Error())
	}

	if err := json.Unmarshal(respJSON, &resp); err != nil {
		return depth, errors.New("failed to json.Unmarshal resp: " +
			err.Error())
	}

	if err := resp.Error(); err != nil {
		return depth, errors.New("bitlum server-side error: " +
			err.Error())
	}

	return resp.Data.Depth, nil
}

// depositRequestVariables is a query variables used in request
// in bitlum Deposits method.
type depositRequestVariables struct {
	Assets []string `json:"assets"`
	Offset int64    `json:"offset"`
	Limit  int64    `json:"limit"`
}

type Deposit struct {
	PaymentID   string
	PaymentType string
	Change      decimal.Decimal
	Time        float64
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

	respJSON, err := c.do(req)
	if err != nil {
		return nil, errors.New("failed to do request: " + err.Error())
	}

	if err := json.Unmarshal(respJSON, &resp); err != nil {
		return nil, errors.New("failed to json.Unmarshal resp: " +
			err.Error())
	}

	if err := resp.Error(); err != nil {
		return nil, errors.New("bitlum server-side error: " +
			err.Error())
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
// in bitlum Order method.
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

	respJSON, err := c.do(req)
	if err != nil {
		return Order{}, errors.New("failed to do request: " + err.
			Error())
	}

	if err := json.Unmarshal(respJSON, &resp); err != nil {
		return Order{}, errors.New("failed to json.Unmarshal resp: " +
			err.Error())
	}

	if err := resp.Error(); err != nil {
		return Order{}, errors.New("bitlum server-side error: " +
			err.Error())
	}

	return resp.Data.Order, nil
}

// createOrderRequestVariables is a query variables used in request
// in bitlum CreateOrder method.
type createOrderRequestVariables struct {
	Market string          `json:"market"`
	Amount decimal.Decimal `json:"amount"`
}

// CreateBidOrder creates bid order on market. Bid order means that
// market[0] asset is used to buy market[1] asset
func (c *Client) CreateOrder(market string,
	amount decimal.Decimal) (Order, error) {

	var req request

	req.Query = `
		mutation CreateMarketOrder($market: Market!, $amount: String!) {
  			createMarketOrder(amount: $amount, market: $market, 
side: bid) {
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
	}

	resp := struct {
		responseBase
		Data struct {
			Order Order `json:"createMarketOrder"`
		}
	}{}

	respJSON, err := c.do(req)
	if err != nil {
		return Order{}, errors.New("failed to do request: " + err.
			Error())
	}

	if err := json.Unmarshal(respJSON, &resp); err != nil {
		return Order{}, errors.New("failed to json.Unmarshal resp: " +
			err.Error())
	}

	if err := resp.Error(); err != nil {
		return Order{}, errors.New("bitlum server-side error: " +
			err.Error())
	}

	return resp.Data.Order, nil

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
// in bitlum Withdraw method.
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

	respJSON, err := c.do(req)
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
			errors.New("bitlum server-side error: " + err.Error())
	}

	return resp.Data.Withdrawal, nil
}

// reachableRequestVariables is a query variables used in request
// in bitlum LightningNodeReachable method.
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

	respJSON, err := c.do(req)
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
			errors.New("bitlum server-side error: " + err.Error())
	}

	return resp.Data.Reachable, nil
}

// LightningNodeInfo is a lightning network node info
type LightningNodeInfo struct {
	Host      string
	Port      string
	MinAmount decimal.Decimal
	MaxAmount decimal.Decimal

	// The identity pubkey of the current node.
	IdentityPubkey string

	// If applicable, the alias of the current node, e.g. "bob"
	Alias string

	// Number of pending channels
	NumPendingChannels uint32

	// Number of active channels
	NumActiveChannels uint32

	// Number of peers
	NumPeers uint32

	// The node's current view of the height of the best block
	BlockHeight uint32

	// The node's current view of the hash of the best block
	BlockHash string

	// Whether the wallet's view is synced to the main chain
	SyncedToChain bool

	// Whether the current node is connected to testnet
	Testnet bool

	// A list of active chains the node is connected to
	Chains []string
}

// nodeInfoRequestVariables is a query variables used in request
// in bitlum LightningNodeInfo method.
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

	respJSON, err := c.do(req)
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
			errors.New("bitlum server-side error: " + err.Error())
	}

	return resp.Data.Info, nil
}

// lightningCreateRequestVariables is a query variables used in request
// in bitlum LightningCreateInvoice method.
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

	respJSON, err := c.do(req)
	if err != nil {
		return "", errors.New("failed to do request: " + err.Error())
	}

	if err := json.Unmarshal(respJSON, &resp); err != nil {
		return "", errors.New("failed to json.Unmarshal resp: " +
			err.Error())
	}

	if err := resp.Error(); err != nil {
		return "", errors.New("bitlum server-side error: " +
			err.Error())
	}

	return resp.Data.Invoice, nil
}

// lightningWithdrawRequestError is a query variables used in request
// in bitlum LightningWithdraw method.
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

	respJSON, err := c.do(req)
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
			errors.New("bitlum server-side error: " + err.Error())
	}

	return resp.Data.Withdrawal, nil
}
