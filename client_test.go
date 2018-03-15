package client

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/kr/pretty"
	"github.com/shopspring/decimal"
)

func TestNewExchange(t *testing.T) {
	const (
		wantURL = "http://test.wantURL"
	)

	client, err := NewClient(wantURL, macaroonHexEncoded)
	if err != nil {
		t.Fatalf("want NewClient no error but got `%v`", err)
	}
	if client.core == nil {
		t.Fatal("want not nil core")
	}
	core, isGraphQLCore := client.core.(*graphQLCore)
	if !isGraphQLCore {
		t.Fatal("want client.core is graphQLCore")
	}
	if core.url != wantURL {
		t.Fatalf("want client.core.wantURL is `%s` but got `%s`",
			wantURL, core.url)
	}
	if core.macaroon == nil {
		t.Fatal("want not nil client.core.macaroon")
	}
}

func TestClient_Markets(t *testing.T) {
	want := []string{
		"BTCETH",
		"BTCBCH",
		"BTCDASH",
		"BTCLTC",
	}
	got := (&Client{}).Markets()
	if !reflect.DeepEqual(want, got) {
		t.Errorf("want `%v` markets but got `%v`", want, got)
	}
}

func TestClient_UserID(t *testing.T) {
	checkRequest := func(t *testing.T, got request) {
		// TODO (dimuls): validate request.Query
		if got.Variables != nil {
			t.Fatalf("want nil request variables but got %#v", got.Variables)
		}
	}
	t.Run("when core error", func(t *testing.T) {
		backend := &mockCore{
			error: errors.New("fail"),
		}
		client := &Client{core: backend}
		_, err := client.UserID()
		if err == nil {
			t.Fatal("want error but got no error")
		}
		if !strings.Contains(err.Error(), "failed to do request") {
			t.Fatalf("want json.Unmarshal error but got `%s`", err.Error())
		}
		checkRequest(t, backend.request)
	})
	t.Run("when invalid response json", func(t *testing.T) {
		backend := &mockCore{
			respJSON: `
				{ "errors": 123, "data": "qwerty" }
			`,
		}
		client := &Client{core: backend}
		_, err := client.UserID()
		if err == nil {
			t.Fatal("want error but got no error")
		}
		if !strings.Contains(err.Error(), "failed to json.Unmarshal") {
			t.Fatalf("want json.Unmarshal error but got `%s`", err.Error())
		}
		checkRequest(t, backend.request)
	})
	t.Run("when exchange error", func(t *testing.T) {
		backend := &mockCore{
			respJSON: `
				{ "errors": [{ "message": "some error" }] }
			`,
		}
		client := &Client{core: backend}
		_, err := client.UserID()
		if err == nil {
			t.Fatal("want error but got no error")
		}
		if !strings.Contains(err.Error(), "exchange error") {
			t.Fatalf("want exchange error but got `%s`", err.Error())
		}
		checkRequest(t, backend.request)
	})
	t.Run("when valid response without errors", func(t *testing.T) {
		const wantUserID = "some-id"
		backend := &mockCore{
			respJSON: `
				{ "data": { "me": { "id": "` + wantUserID + `" } } }
			`,
		}
		client := &Client{core: backend}
		gotUserID, err := client.UserID()
		if err != nil {
			t.Fatalf("want no error but got `%s", err.Error())
		}
		if gotUserID != wantUserID {
			t.Fatalf("want userID `%s` but got `%s`", wantUserID, gotUserID)
		}
		checkRequest(t, backend.request)
	})
}

func TestClient_Tickers(t *testing.T) {
	wantMarkets := []string{"BTCETH", "BTCDASH"}
	checkRequest := func(t *testing.T, got request) {
		// TODO (dimuls): validate request.Query
		wantVariables := tickersRequestVariables{
			Markets: wantMarkets}
		if !reflect.DeepEqual(wantVariables, got.Variables) {
			t.Errorf("want variables `%#v` but got `%#v`",
				wantVariables, got.Variables)
		}
	}
	t.Run("when empty markets", func(t *testing.T) {
		client := &Client{core: nil}
		_, err := client.Tickers(nil)
		if err == nil {
			t.Fatal("want error but got no error")
		}
		if !strings.Contains(err.Error(), "not empty markets expected") {
			t.Fatalf("want `not empty markets expected` error but got `%s`", err.Error())
		}
	})
	t.Run("when core error", func(t *testing.T) {
		backend := &mockCore{
			error: errors.New("fail"),
		}
		client := &Client{core: backend}
		_, err := client.Tickers(wantMarkets)
		if err == nil {
			t.Fatal("want error but got no error")
		}
		if !strings.Contains(err.Error(), "failed to do request") {
			t.Fatalf("want json.Unmarshal error but got `%s`", err.Error())
		}
		checkRequest(t, backend.request)
	})
	t.Run("when invalid response json", func(t *testing.T) {
		backend := &mockCore{
			respJSON: `
				{ "errors": 123, "data": "qwerty" }
			`,
		}
		client := &Client{core: backend}
		_, err := client.Tickers(wantMarkets)
		if err == nil {
			t.Fatal("want error but got no error")
		}
		if !strings.Contains(err.Error(), "failed to json.Unmarshal") {
			t.Fatalf("want json.Unmarshal error but got `%s`", err.Error())
		}
		checkRequest(t, backend.request)
	})
	t.Run("when exchange error", func(t *testing.T) {
		backend := &mockCore{
			respJSON: `
				{ "errors": [{ "message": "some error" }] }
			`,
		}
		client := &Client{core: backend}
		_, err := client.Tickers(wantMarkets)
		if err == nil {
			t.Fatal("want error but got no error")
		}
		if !strings.Contains(err.Error(), "exchange error") {
			t.Fatalf("want exchange error but got `%s`", err.Error())
		}
		checkRequest(t, backend.request)
	})
	t.Run("when valid response without errors", func(t *testing.T) {
		wantTickers := []Ticker{{
			Market:     "BTCETH",
			Last:       dec(10),
			ChangeLast: dec(20),
		}, {
			Market:     "BTCDASH",
			Last:       dec(15),
			ChangeLast: dec(25),
		}}
		backend := &mockCore{
			respJSON: `
				{ "data": { "markets": [ 
					{ "market": "BTCETH", "last": "10", "changeLast": "20" },
					{ "market": "BTCDASH", "last": "15", "changeLast": "25" }
				] } }
			`,
		}
		client := &Client{core: backend}
		gotTickers, err := client.Tickers(
			[]string{"BTCETH", "BTCDASH"})
		if err != nil {
			t.Fatalf("want no error but got `%s", err.Error())
		}
		if !reflect.DeepEqual(wantTickers, gotTickers) {
			t.Errorf("want tickers `%v` but got `%v`", wantTickers,
				gotTickers)
			t.Log("want and got diff: ", pretty.Diff(wantTickers,
				gotTickers))
		}
		checkRequest(t, backend.request)
	})
}

func TestClient_Depth(t *testing.T) {
	wantMarket := "BTCETH"
	checkRequest := func(t *testing.T, got request) {
		// TODO (dimuls): validate request.Query
		wantVariables := depthRequestVariables{wantMarket}
		if !reflect.DeepEqual(wantVariables, got.Variables) {
			t.Errorf("want variables `%#v` but got `%#v`",
				wantVariables, got.Variables)
		}
	}
	t.Run("when core error", func(t *testing.T) {
		backend := &mockCore{
			error: errors.New("fail"),
		}
		client := &Client{core: backend}
		_, err := client.Depth(wantMarket)
		if err == nil {
			t.Fatal("want error but got no error")
		}
		if !strings.Contains(err.Error(), "failed to do request") {
			t.Fatalf("want json.Unmarshal error but got `%s`", err.Error())
		}
		checkRequest(t, backend.request)
	})
	t.Run("when invalid response json", func(t *testing.T) {
		backend := &mockCore{
			respJSON: `
				{ "errors": 123, "data": "qwerty" }
			`,
		}
		client := &Client{core: backend}
		_, err := client.Depth(wantMarket)
		if err == nil {
			t.Fatal("want error but got no error")
		}
		if !strings.Contains(err.Error(), "failed to json.Unmarshal") {
			t.Fatalf("want json.Unmarshal error but got `%s`", err.Error())
		}
		checkRequest(t, backend.request)
	})
	t.Run("when exchange error", func(t *testing.T) {
		backend := &mockCore{
			respJSON: `
				{ "errors": [{ "message": "some error" }] }
			`,
		}
		client := &Client{core: backend}
		_, err := client.Depth(wantMarket)
		if err == nil {
			t.Fatal("want error but got no error")
		}
		if !strings.Contains(err.Error(), "exchange error") {
			t.Fatalf("want exchange error but got `%s`", err.Error())
		}
		checkRequest(t, backend.request)
	})
	t.Run("when valid response without errors", func(t *testing.T) {
		wantDepth := Depth{
			Asks: []Ask{{
				Price:  dec(1),
				Volume: dec(2)}, {
				Price:  dec(0.5),
				Volume: dec(3.5)}},
			Bids: []Bid{{
				Price:  dec(1.5),
				Volume: dec(2.5)}},
		}
		backend := &mockCore{
			respJSON: `
				{ "data": { "depth": {
					"asks": [{ "price": "1", "volume": "2" },
						   { "price": "0.5", "volume": "3.5" }],
					"bids": [{ "price": "1.5", "volume": "2.5" }]
				} } }
			`,
		}
		client := &Client{core: backend}
		gotDepth, err := client.Depth(wantMarket)
		if err != nil {
			t.Fatalf("want no error but got `%s", err.Error())
		}
		if !reflect.DeepEqual(wantDepth, gotDepth) {
			t.Errorf("want depth `%#v` but got `%#v`", wantDepth,
				gotDepth)
			t.Log("want and got diff: ", pretty.Diff(wantDepth,
				gotDepth))
		}

		checkRequest(t, backend.request)
	})
}

func TestClient_Deposits(t *testing.T) {
	wantAsset := "ETH"
	wantOffset := int64(100)
	wantLimit := int64(50)
	checkRequest := func(t *testing.T, got request) {
		// TODO (dimuls): validate request.Query
		wantVariables := depositRequestVariables{
			Assets: []string{wantAsset},
			Offset: wantOffset,
			Limit:  wantLimit,
		}
		if !reflect.DeepEqual(wantVariables, got.Variables) {
			t.Errorf("want variables `%#v` but got `%#v`",
				wantVariables, got.Variables)
		}
	}
	t.Run("when core error", func(t *testing.T) {
		backend := &mockCore{
			error: errors.New("fail"),
		}
		client := &Client{core: backend}
		_, err := client.Deposits(wantAsset, wantOffset, wantLimit)
		if err == nil {
			t.Fatal("want error but got no error")
		}
		if !strings.Contains(err.Error(), "failed to do request") {
			t.Fatalf("want json.Unmarshal error but got `%s`", err.Error())
		}
		checkRequest(t, backend.request)
	})
	t.Run("when invalid response json", func(t *testing.T) {
		backend := &mockCore{
			respJSON: `
				{ "errors": 123, "data": "qwerty" }
			`,
		}
		client := &Client{core: backend}
		_, err := client.Deposits(wantAsset, wantOffset, wantLimit)
		if err == nil {
			t.Fatal("want error but got no error")
		}
		if !strings.Contains(err.Error(), "failed to json.Unmarshal") {
			t.Fatalf("want json.Unmarshal error but got `%s`", err.Error())
		}
		checkRequest(t, backend.request)
	})
	t.Run("when exchange error", func(t *testing.T) {
		backend := &mockCore{
			respJSON: `
				{ "errors": [{ "message": "some error" }] }
			`,
		}
		client := &Client{core: backend}
		_, err := client.Deposits(wantAsset, wantOffset, wantLimit)
		if err == nil {
			t.Fatal("want error but got no error")
		}
		if !strings.Contains(err.Error(), "exchange error") {
			t.Fatalf("want exchange error but got `%s`", err.Error())
		}
		checkRequest(t, backend.request)
	})
	t.Run("when unknown payment system", func(t *testing.T) {
		backend := &mockCore{
			respJSON: `
				{ "data": { "balanceUpdateRecords": [
					{ "change": "0.1", "time": 123, 
"paymentID": "some-id", "paymentType": "new-tech" }
				] } }
			`,
		}
		client := &Client{core: backend}
		_, err := client.Deposits(wantAsset, wantOffset, wantLimit)
		if err != nil {
			t.Fatalf("want no error but got error `%v`", err)
		}
		checkRequest(t, backend.request)
	})
	t.Run("when valid response without errors", func(t *testing.T) {
		wantDeposits := []Deposit{{
			PaymentID:   "some-id",
			PaymentType: "blockchain",
			Change:      dec(0.1),
			Time:        123,
		}, {
			PaymentID:   "some-id-2",
			PaymentType: "lightning",
			Change:      dec(-0.1),
			Time:        345,
		}}
		backend := &mockCore{
			respJSON: `
				{ "data": { "balanceUpdateRecords": [
					{ "change": "0.1", "time": 123, 
"paymentID": "some-id", "paymentType": "blockchain" },
					{ "change": "-0.1", "time": 345, 
"paymentID": "some-id-2", "paymentType": "lightning" }
				] } }
			`,
		}
		client := &Client{core: backend}
		gotDeposits, err := client.Deposits(wantAsset, wantOffset,
			wantLimit)
		if err != nil {
			t.Fatalf("want no error but got `%s", err.Error())
		}
		if !reflect.DeepEqual(wantDeposits, gotDeposits) {
			t.Errorf("want depth `%#v` but got `%#v`", wantDeposits,
				gotDeposits)
			t.Log("want and got diff: ", pretty.Diff(wantDeposits,
				gotDeposits))
		}
		checkRequest(t, backend.request)
	})
}

func TestClient_Order(t *testing.T) {
	wantID := int64(123)
	checkRequest := func(t *testing.T, got request) {
		// TODO (dimuls): validate request.Query
		wantVariables := orderRequestVariables{wantID}
		if !reflect.DeepEqual(wantVariables, got.Variables) {
			t.Errorf("want variables `%#v` but got `%#v`",
				wantVariables, got.Variables)
		}
	}
	t.Run("when core error", func(t *testing.T) {
		backend := &mockCore{
			error: errors.New("fail"),
		}
		client := &Client{core: backend}
		_, err := client.Order(wantID)
		if err == nil {
			t.Fatal("want error but got no error")
		}
		if !strings.Contains(err.Error(), "failed to do request") {
			t.Fatalf("want json.Unmarshal error but got `%s`", err.Error())
		}
		checkRequest(t, backend.request)
	})
	t.Run("when invalid response json", func(t *testing.T) {
		backend := &mockCore{
			respJSON: `
				{ "errors": 123, "data": "qwerty" }
			`,
		}
		client := &Client{core: backend}
		_, err := client.Order(wantID)
		if err == nil {
			t.Fatal("want error but got no error")
		}
		if !strings.Contains(err.Error(), "failed to json.Unmarshal") {
			t.Fatalf("want json.Unmarshal error but got `%s`", err.Error())
		}
		checkRequest(t, backend.request)
	})
	t.Run("when exchange error", func(t *testing.T) {
		backend := &mockCore{
			respJSON: `
				{ "errors": [{ "message": "some error" }] }
			`,
		}
		client := &Client{core: backend}
		_, err := client.Order(wantID)
		if err == nil {
			t.Fatal("want error but got no error")
		}
		if !strings.Contains(err.Error(), "exchange error") {
			t.Fatalf("want exchange error but got `%s`", err.Error())
		}
		checkRequest(t, backend.request)
	})
	t.Run("when unknown order status", func(t *testing.T) {
		backend := &mockCore{
			respJSON: `
				{ "data": { "order": {
					"id": 123,
					"status": "new-status",
					"amount": "0.1",
					"price": "-0.2",
					"dealMoney": "0.3",
					"dealStock": "-0.4",
					"left": "1"
				} } }
			`,
		}
		client := &Client{core: backend}
		_, err := client.Order(wantID)
		if err != nil {
			t.Fatalf("want no error but got no error `%v`", err)
		}
		checkRequest(t, backend.request)
	})
	t.Run("when valid response without errors", func(t *testing.T) {
		wantOrder := Order{
			ID:        123,
			Status:    "pending",
			Amount:    dec(0.1),
			Price:     dec(-0.2),
			DealMoney: dec(0.3),
			DealStock: dec(-0.4),
			Left:      dec(1),
		}
		backend := &mockCore{
			respJSON: `
				{ "data": { "order": {
					"id": 123,
					"status": "pending",
					"amount": "0.1",
					"price": "-0.2",
					"dealMoney": "0.3",
					"dealStock": "-0.4",
					"left": "1"
				} } }
			`,
		}
		client := &Client{core: backend}
		gotOrder, err := client.Order(wantID)
		if err != nil {
			t.Fatalf("want no error but got `%s", err.Error())
		}
		if !reflect.DeepEqual(wantOrder, gotOrder) {
			t.Errorf("want order `%#v` but got `%#v`", wantOrder,
				gotOrder)
			t.Log("want and got diff: ", pretty.Diff(wantOrder,
				gotOrder))
		}
		checkRequest(t, backend.request)
	})
}

func TestClient_CreateOrder(t *testing.T) {
	wantAmount := dec(0.5)
	wantMarket := "BTCETH"
	checkRequest := func(t *testing.T, got request) {
		// TODO (dimuls): validate request.Query
		wantVariables := createOrderRequestVariables{
			Market: wantMarket,
			Amount: wantAmount,
		}
		if !reflect.DeepEqual(wantVariables, got.Variables) {
			t.Errorf("want variables `%#v` but got `%#v`",
				wantVariables, got.Variables)
		}
	}
	t.Run("when core error", func(t *testing.T) {
		backend := &mockCore{
			error: errors.New("fail"),
		}
		client := &Client{core: backend}
		_, err := client.CreateOrder(wantMarket, wantAmount)
		if err == nil {
			t.Fatal("want error but got no error")
		}
		if !strings.Contains(err.Error(), "failed to do request") {
			t.Fatalf("want json.Unmarshal error but got `%s`", err.Error())
		}
		checkRequest(t, backend.request)
	})
	t.Run("when invalid response json", func(t *testing.T) {
		backend := &mockCore{
			respJSON: `
				{ "errors": 123, "data": "qwerty" }
			`,
		}
		client := &Client{core: backend}
		_, err := client.CreateOrder(wantMarket, wantAmount)
		if err == nil {
			t.Fatal("want error but got no error")
		}
		if !strings.Contains(err.Error(), "failed to json.Unmarshal") {
			t.Fatalf("want json.Unmarshal error but got `%s`", err.Error())
		}
		checkRequest(t, backend.request)
	})
	t.Run("when exchange error", func(t *testing.T) {
		backend := &mockCore{
			respJSON: `
				{ "errors": [{ "message": "some error" }] }
			`,
		}
		client := &Client{core: backend}
		_, err := client.CreateOrder(wantMarket, wantAmount)
		if err == nil {
			t.Fatal("want error but got no error")
		}
		if !strings.Contains(err.Error(), "exchange error") {
			t.Fatalf("want exchange error but got `%s`", err.Error())
		}
		checkRequest(t, backend.request)
	})
	t.Run("when unknown order status", func(t *testing.T) {
		backend := &mockCore{
			respJSON: `
				{ "data": { "createMarketOrder": {
					"id": 123,
					"status": "new-status",
					"amount": "0.1",
					"price": "-0.2",
					"dealMoney": "0.3",
					"dealStock": "-0.4",
					"left": "1"
				} } }
			`,
		}
		client := &Client{core: backend}
		_, err := client.CreateOrder(wantMarket, wantAmount)
		if err != nil {
			t.Fatalf("want no error but got error `%v`", err)
		}
		checkRequest(t, backend.request)
	})
	t.Run("when valid response without errors", func(t *testing.T) {
		wantOrder := Order{
			ID:        123,
			Status:    "finished",
			Amount:    dec(0.1),
			Price:     dec(-0.2),
			DealMoney: dec(0.3),
			DealStock: dec(-0.4),
			Left:      dec(1),
		}
		backend := &mockCore{
			respJSON: `
				{ "data": { "createMarketOrder": {
					"id": 123,
					"status": "finished",
					"amount": "0.1",
					"price": "-0.2",
					"dealMoney": "0.3",
					"dealStock": "-0.4",
					"left": "1"
				} } }
			`,
		}
		client := &Client{core: backend}
		gotOrder, err := client.CreateOrder(wantMarket, wantAmount)
		if err != nil {
			t.Fatalf("want no error but got `%s", err.Error())
		}
		if !reflect.DeepEqual(wantOrder, gotOrder) {
			t.Errorf("want order `%#v` but got `%#v`", wantOrder,
				gotOrder)
			t.Log("want and got diff: ", pretty.Diff(wantOrder,
				gotOrder))
		}
		checkRequest(t, backend.request)
	})
}

func TestClient_Withdraw(t *testing.T) {
	wantAsset := "ETH"
	wantAmount := dec(10)
	wantAddress := "some-address"
	checkRequest := func(t *testing.T, got request) {
		// TODO (dimuls): validate request.Query
		wantVariables := withdrawRequestVariables{
			Asset:   wantAsset,
			Amount:  wantAmount,
			Address: wantAddress,
		}
		if !reflect.DeepEqual(wantVariables, got.Variables) {
			t.Errorf("want variables `%#v` but got `%#v`",
				wantVariables, got.Variables)
		}
	}
	t.Run("when core error", func(t *testing.T) {
		backend := &mockCore{
			error: errors.New("fail"),
		}
		client := &Client{core: backend}
		_, err := client.Withdraw(wantAsset, wantAmount, wantAddress)
		if err == nil {
			t.Fatal("want error but got no error")
		}
		if !strings.Contains(err.Error(), "failed to do request") {
			t.Fatalf("want json.Unmarshal error but got `%s`", err.Error())
		}
		checkRequest(t, backend.request)
	})
	t.Run("when invalid response json", func(t *testing.T) {
		backend := &mockCore{
			respJSON: `
				{ "errors": 123, "data": "qwerty" }
			`,
		}
		client := &Client{core: backend}
		_, err := client.Withdraw(wantAsset, wantAmount, wantAddress)
		if err == nil {
			t.Fatal("want error but got no error")
		}
		if !strings.Contains(err.Error(), "failed to json.Unmarshal") {
			t.Fatalf("want json.Unmarshal error but got `%s`", err.Error())
		}
		checkRequest(t, backend.request)
	})
	t.Run("when exchange error", func(t *testing.T) {
		backend := &mockCore{
			respJSON: `
				{ "errors": [{ "message": "some error" }] }
			`,
		}
		client := &Client{core: backend}
		_, err := client.Withdraw(wantAsset, wantAmount, wantAddress)
		if err == nil {
			t.Fatal("want error but got no error")
		}
		if !strings.Contains(err.Error(), "exchange error") {
			t.Fatalf("want exchange error but got `%s`", err.Error())
		}
		checkRequest(t, backend.request)
	})
	t.Run("when valid response without errors", func(t *testing.T) {
		wantWithdrawal := Withdrawal{
			PaymentID:   "some-id",
			PaymentAddr: "some-address",
			Change:      dec(15.75),
		}
		backend := &mockCore{
			respJSON: `
				{ "data": { "withdrawWithBlockchain": {
					"paymentID": "some-id",
					"paymentAddr": "some-address",
					"change": "15.75"
				} } }
			`,
		}
		client := &Client{core: backend}
		gotWithdrawal, err := client.Withdraw(wantAsset, wantAmount,
			wantAddress)
		if err != nil {
			t.Fatalf("want no error but got `%s", err.Error())
		}
		if !reflect.DeepEqual(wantWithdrawal, gotWithdrawal) {
			t.Errorf("want withdrawal `%#v` but got `%#v`", wantWithdrawal,
				gotWithdrawal)
			t.Log("want and got diff: ", pretty.Diff(wantWithdrawal,
				gotWithdrawal))
		}
		checkRequest(t, backend.request)
	})
}

func TestClient_LightningNodeReachable(t *testing.T) {
	wantAsset := "ETH"
	wantIdentityPubKey := "some-pub-key"
	checkRequest := func(t *testing.T, got request) {
		// TODO (dimuls): validate request.Query
		wantVariables := reachableRequestVariables{
			Asset:          wantAsset,
			IdentityPubKey: wantIdentityPubKey,
		}
		if !reflect.DeepEqual(wantVariables, got.Variables) {
			t.Errorf("want variables `%#v` but got `%#v`",
				wantVariables, got.Variables)
		}
	}
	t.Run("when core error", func(t *testing.T) {
		backend := &mockCore{
			error: errors.New("fail"),
		}
		client := &Client{core: backend}
		_, err := client.LightningNodeReachable(wantAsset, wantIdentityPubKey)
		if err == nil {
			t.Fatal("want error but got no error")
		}
		if !strings.Contains(err.Error(), "failed to do request") {
			t.Fatalf("want json.Unmarshal error but got `%s`", err.Error())
		}
		checkRequest(t, backend.request)
	})
	t.Run("when invalid response json", func(t *testing.T) {
		backend := &mockCore{
			respJSON: `
				{ "errors": 123, "data": "qwerty" }
			`,
		}
		client := &Client{core: backend}
		_, err := client.LightningNodeReachable(wantAsset, wantIdentityPubKey)
		if err == nil {
			t.Fatal("want error but got no error")
		}
		if !strings.Contains(err.Error(), "failed to json.Unmarshal") {
			t.Fatalf("want json.Unmarshal error but got `%s`", err.Error())
		}
		checkRequest(t, backend.request)
	})
	t.Run("when exchange error", func(t *testing.T) {
		backend := &mockCore{
			respJSON: `
				{ "errors": [{ "message": "some error" }] }
			`,
		}
		client := &Client{core: backend}
		_, err := client.LightningNodeReachable(wantAsset, wantIdentityPubKey)
		if err == nil {
			t.Fatal("want error but got no error")
		}
		if !strings.Contains(err.Error(), "exchange error") {
			t.Fatalf("want exchange error but got `%s`", err.Error())
		}
		checkRequest(t, backend.request)
	})
	t.Run("when valid response without errors", func(t *testing.T) {
		wantReachable := false
		backend := &mockCore{
			respJSON: `
				{ "data": { "checkReachable": false } }
			`,
		}
		client := &Client{core: backend}
		gotReachable, err := client.LightningNodeReachable(wantAsset,
			wantIdentityPubKey)
		if err != nil {
			t.Fatalf("want no error but got `%s", err.Error())
		}
		if wantReachable != gotReachable {
			t.Errorf("want reachable `%t` but got `%t`",
				wantReachable, gotReachable)
		}
		checkRequest(t, backend.request)
	})
}

func TestClient_LightningNodeInfo(t *testing.T) {
	wantAsset := "ETH"
	checkRequest := func(t *testing.T, got request) {
		// TODO (dimuls): validate request.Query
		wantVariables := nodeInfoRequestVariables{
			Asset: wantAsset,
		}
		if !reflect.DeepEqual(wantVariables, got.Variables) {
			t.Errorf("want variables `%#v` but got `%#v`",
				wantVariables, got.Variables)
		}
	}
	t.Run("when core error", func(t *testing.T) {
		backend := &mockCore{
			error: errors.New("fail"),
		}
		client := &Client{core: backend}
		_, err := client.LightningNodeInfo(wantAsset)
		if err == nil {
			t.Fatal("want error but got no error")
		}
		if !strings.Contains(err.Error(), "failed to do request") {
			t.Fatalf("want json.Unmarshal error but got `%s`", err.Error())
		}
		checkRequest(t, backend.request)
	})
	t.Run("when invalid response json", func(t *testing.T) {
		backend := &mockCore{
			respJSON: `
				{ "errors": 123, "data": "qwerty" }
			`,
		}
		client := &Client{core: backend}
		_, err := client.LightningNodeInfo(wantAsset)
		if err == nil {
			t.Fatal("want error but got no error")
		}
		if !strings.Contains(err.Error(), "failed to json.Unmarshal") {
			t.Fatalf("want json.Unmarshal error but got `%s`", err.Error())
		}
		checkRequest(t, backend.request)
	})
	t.Run("when exchange error", func(t *testing.T) {
		backend := &mockCore{
			respJSON: `
				{ "errors": [{ "message": "some error" }] }
			`,
		}
		client := &Client{core: backend}
		_, err := client.LightningNodeInfo(wantAsset)
		if err == nil {
			t.Fatal("want error but got no error")
		}
		if !strings.Contains(err.Error(), "exchange error") {
			t.Fatalf("want exchange error but got `%s`", err.Error())
		}
		checkRequest(t, backend.request)
	})
	t.Run("when valid response without errors", func(t *testing.T) {
		wantInfo := LightningNodeInfo{
			Host:               "host",
			Port:               "port",
			MinAmount:          dec(0.1),
			MaxAmount:          dec(0.9),
			IdentityPubkey:     "pub-key",
			Alias:              "bob",
			NumPendingChannels: 12,
			NumActiveChannels:  6,
			NumPeers:           123,
			BlockHeight:        100,
			BlockHash:          "hash",
			SyncedToChain:      true,
			Testnet:            false,
			Chains:             []string{"chain-1", "chain-2"},
		}
		backend := &mockCore{
			respJSON: `
				{ "data": { "lightningInfo": {
					"host": "host",
					"port": "port",
					"minAmount": "0.1",
					"maxAmount": "0.9",
					"identityPubkey": "pub-key",
					"alias": "bob",
					"numPendingChannels": 12,
					"numActiveChannels": 6,
					"numPeers": 123,
					"blockHeight": 100,
					"blockHash": "hash",
					"syncedToChain": true,
					"testnet": false,
					"chains": ["chain-1", "chain-2"]
				} } }
			`,
		}
		client := &Client{core: backend}
		gotInfo, err := client.LightningNodeInfo(wantAsset)
		if err != nil {
			t.Fatalf("want no error but got `%s", err.Error())
		}
		if !reflect.DeepEqual(wantInfo, gotInfo) {
			t.Errorf("want order `%#v` but got `%#v`", wantInfo,
				gotInfo)
			t.Log("want and got diff: ", pretty.Diff(wantInfo,
				gotInfo))
		}
		checkRequest(t, backend.request)
	})
}

func TestClient_LightningCreateInvoice(t *testing.T) {
	wantAsset := "ETH"
	wantAmount := dec(0.123)
	checkRequest := func(t *testing.T, got request) {
		// TODO (dimuls): validate request.Query
		wantVariables := lightningCreateRequestVariables{
			Asset:  wantAsset,
			Amount: wantAmount,
		}
		if !reflect.DeepEqual(wantVariables, got.Variables) {
			t.Errorf("want variables `%#v` but got `%#v`",
				wantVariables, got.Variables)
		}
	}
	t.Run("when core error", func(t *testing.T) {
		backend := &mockCore{
			error: errors.New("fail"),
		}
		client := &Client{core: backend}
		_, err := client.LightningCreateInvoice(wantAsset, wantAmount)
		if err == nil {
			t.Fatal("want error but got no error")
		}
		if !strings.Contains(err.Error(), "failed to do request") {
			t.Fatalf("want json.Unmarshal error but got `%s`", err.Error())
		}
		checkRequest(t, backend.request)
	})
	t.Run("when invalid response json", func(t *testing.T) {
		backend := &mockCore{
			respJSON: `
				{ "errors": 123, "data": "qwerty" }
			`,
		}
		client := &Client{core: backend}
		_, err := client.LightningCreateInvoice(wantAsset, wantAmount)
		if err == nil {
			t.Fatal("want error but got no error")
		}
		if !strings.Contains(err.Error(), "failed to json.Unmarshal") {
			t.Fatalf("want json.Unmarshal error but got `%s`", err.Error())
		}
		checkRequest(t, backend.request)
	})
	t.Run("when exchange error", func(t *testing.T) {
		backend := &mockCore{
			respJSON: `
				{ "errors": [{ "message": "some error" }] }
			`,
		}
		client := &Client{core: backend}
		_, err := client.LightningCreateInvoice(wantAsset, wantAmount)
		if err == nil {
			t.Fatal("want error but got no error")
		}
		if !strings.Contains(err.Error(), "exchange error") {
			t.Fatalf("want exchange error but got `%s`", err.Error())
		}
		checkRequest(t, backend.request)
	})
	t.Run("when valid response without errors", func(t *testing.T) {
		wantInvoice := "some-invoice"
		backend := &mockCore{
			respJSON: `
				{ "data": { "generateLightningInvoice": "some-invoice" } }
			`,
		}
		client := &Client{core: backend}
		gotInvoice, err := client.LightningCreateInvoice(wantAsset,
			wantAmount)
		if err != nil {
			t.Fatalf("want no error but got `%s", err.Error())
		}
		if wantInvoice != gotInvoice {
			t.Errorf("want invoice `%#v` but got `%#v`",
				wantInvoice,
				gotInvoice)
		}
		checkRequest(t, backend.request)
	})
}

func TestClient_LightningWithdraw(t *testing.T) {
	wantAsset := "ETH"
	wantInvoice := "some-invoice"
	checkRequest := func(t *testing.T, got request) {
		// TODO (dimuls): validate request.Query
		wantVariables := lightningWithdrawRequestError{
			Asset:   wantAsset,
			Invoice: wantInvoice,
		}
		if !reflect.DeepEqual(wantVariables, got.Variables) {
			t.Errorf("want variables `%#v` but got `%#v`",
				wantVariables, got.Variables)
		}
	}
	t.Run("when core error", func(t *testing.T) {
		backend := &mockCore{
			error: errors.New("fail"),
		}
		client := &Client{core: backend}
		_, err := client.LightningWithdraw(wantAsset, wantInvoice)
		if err == nil {
			t.Fatal("want error but got no error")
		}
		if !strings.Contains(err.Error(), "failed to do request") {
			t.Fatalf("want json.Unmarshal error but got `%s`", err.Error())
		}
		checkRequest(t, backend.request)
	})
	t.Run("when invalid response json", func(t *testing.T) {
		backend := &mockCore{
			respJSON: `
				{ "errors": 123, "data": "qwerty" }
			`,
		}
		client := &Client{core: backend}
		_, err := client.LightningWithdraw(wantAsset, wantInvoice)
		if err == nil {
			t.Fatal("want error but got no error")
		}
		if !strings.Contains(err.Error(), "failed to json.Unmarshal") {
			t.Fatalf("want json.Unmarshal error but got `%s`", err.Error())
		}
		checkRequest(t, backend.request)
	})
	t.Run("when exchange error", func(t *testing.T) {
		backend := &mockCore{
			respJSON: `
				{ "errors": [{ "message": "some error" }] }
			`,
		}
		client := &Client{core: backend}
		_, err := client.LightningWithdraw(wantAsset, wantInvoice)
		if err == nil {
			t.Fatal("want error but got no error")
		}
		if !strings.Contains(err.Error(), "exchange error") {
			t.Fatalf("want exchange error but got `%s`", err.Error())
		}
		checkRequest(t, backend.request)
	})
	t.Run("when valid response without errors", func(t *testing.T) {
		wantWithdrawal := Withdrawal{
			PaymentID: "some-id",
		}
		backend := &mockCore{
			respJSON: `
				{ "data": { "withdrawWithLightning": {
					"paymentID": "some-id"
				} } }
			`,
		}
		client := &Client{core: backend}
		gotWithdrawal, err := client.LightningWithdraw(wantAsset, wantInvoice)
		if err != nil {
			t.Fatalf("want no error but got `%s", err.Error())
		}
		if !reflect.DeepEqual(wantWithdrawal, gotWithdrawal) {
			t.Errorf("want withdrawal `%#v` but got `%#v`",
				wantWithdrawal,
				gotWithdrawal)
			t.Log("want and got diff: ", pretty.Diff(wantWithdrawal,
				gotWithdrawal))
		}
		checkRequest(t, backend.request)
	})
}

// mockCore is client core client mock implementation for testing
// purpose
type mockCore struct {

	// request is last request passed to do() call
	request request

	// respJSON is a response JSON which will returned from next do() call
	respJSON string

	// error is an error which will returned from next do() call
	error error
}

// do implements core. Stores request and returns predefined respJSON
// and error.
func (c *mockCore) do(r request) ([]byte, error) {
	c.request = r
	return []byte(c.respJSON), c.error
}

func dec(f float64) decimal.Decimal {
	d, _ := decimal.NewFromString(fmt.Sprintf("%f", f))
	return d
}
