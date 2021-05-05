package bitflyer

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

const baseURL = "https://api.bitflyer.com/v1/"

type APIClient struct {
	key        string
	secret     string
	httpClient *http.Client
}

func New(key, secret string) *APIClient {
	apiClient := &APIClient{key, secret, &http.Client{}}
	return apiClient
}

func (api APIClient) header(method, endpoint string, body []byte) map[string]string {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	// log.Println(timestamp)
	message := timestamp + method + endpoint + string(body)

	mac := hmac.New(sha256.New, []byte(api.secret))
	mac.Write([]byte(message))
	sign := hex.EncodeToString(mac.Sum(nil))

	return map[string]string{
		"ACCESS-KEY":       api.key,
		"ACCESS-TIMESTAMP": timestamp,
		"ACCESS-SIGN":      sign,
		"Content-Type":     "application/json",
	}
}

func (api *APIClient) doRequest(method, urlPath string, query map[string]string, data []byte) (body []byte, err error) {
	baseURL, err := url.Parse(baseURL)
	if err != nil {
		return
	}
	apiURL, err := url.Parse(urlPath)
	if err != nil {
		return
	}
	endpoint := baseURL.ResolveReference(apiURL).String()
	log.Printf("action=doRequest endpoint=%s", endpoint)

	req, err := http.NewRequest(method, endpoint, bytes.NewBuffer(data))
	if err != nil {
		return
	}
	q := req.URL.Query()
	for key, value := range query {
		q.Add(key, value)
	}
	req.URL.RawQuery = q.Encode()

	for key, value := range api.header(method, req.URL.RequestURI(), data) {
		req.Header.Add(key, value)
	}

	resp, err := api.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// 残高取得
type Balance struct {
	CurrentCode string  `json:"currency_code"`
	Amount      float64 `json:"amount"`
	Available   float64 `json:"available"`
}

func (api APIClient) GetBalance() ([]Balance, error) {
	url := "me/getbalance"
	resp, err := api.doRequest("GET", url, map[string]string{}, nil)
	log.Printf("url=%s resp=%s", url, string(resp))
	if err != nil {
		log.Printf("action=GetBalance err=%s", err.Error())
		return nil, err
	}

	var balance []Balance
	err = json.Unmarshal(resp, &balance)
	if err != nil {
		log.Printf("action=GetBalance err=%s", err.Error())
		return nil, err
	}

	return balance, nil
}

type Ticker struct {
	ProductCode     string  `json:"product_code"`
	State           string  `json:"state"`
	Timestamp       string  `json:"timestamp"`
	TickID          int     `json:"tick_id"`
	BestBid         float64 `json:"best_bid"`
	BestAsk         float64 `json:"best_ask"`
	BestBidSize     float64 `json:"best_bid_size"`
	BestAskSize     float64 `json:"best_ask_size"`
	TotalBidDepth   float64 `json:"total_bid_depth"`
	TotalAskDepth   float64 `json:"total_ask_depth"`
	MarketBidSize   float64 `json:"market_bid_size"`
	MarketAskSize   float64 `json:"market_ask_size"`
	Ltp             float64 `json:"ltp"`
	Volume          float64 `json:"volume"`
	VolumeByProduct float64 `json:"volume_by_product"`
}

func (t *Ticker) GetMidPrice() float64 {
	return (t.BestBid + t.BestAsk) / 2
}

func (t *Ticker) DateTime() time.Time {
	// リアルタイム(JSON-RPC 2.0 over WebSocket)で取ってくるTickerはゾーン情報が含まれる
	// 2021-01-11T04:53:42.4858142Z

	dateTime, err := time.Parse(time.RFC3339, t.Timestamp)
	if err != nil {
		log.Printf("action=DateTime, err=%s", err.Error())
		// GET /v1/ticker でbiflyerの返してくるタイムスタンプ
		// 2021-01-11T04:47:33.087
		/*
			ゾーン情報がないので、RFC3339に変換できずにエラーになる
			action=DateTime, err=parsing time "2021-01-11T04:47:33.087" as "2006-01-02T15:04:05Z07:00": cannot parse "" as "Z07:00"
			0001-01-01 00:00:00 +0000 UTC
		*/
	}
	return dateTime
}

func (t *Ticker) TruncateDateTime(duration time.Duration) time.Time {
	return t.DateTime().Truncate(duration)
}

// Ticker取得
func (api APIClient) GetTicker(productCode string) (*Ticker, error) {
	url := "ticker"
	resp, err := api.doRequest("GET", url, map[string]string{"product_code": productCode}, nil)
	if err != nil {
		return nil, err
	}

	var ticker Ticker
	err = json.Unmarshal(resp, &ticker)
	if err != nil {
		return nil, err
	}

	return &ticker, nil
}

type JsonRPC2 struct {
	Version string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	Result  interface{} `json:"result,omitempty"`
	Id      *int        `json:"id,omitempty"`
}

type SubscribeParams struct {
	Channel string `json:"channel"`
}

// Tickerリアルタイム取得
// JSON-RPC 2.0 over WebSocket (https://bf-lightning-api.readme.io/docs/endpoint-json-rpc)
func (api *APIClient) GetRealTimeTicker(symbol string, ch chan<- Ticker) {
	// エンドポイント wss://ws.lightstream.bitflyer.com/json-rpc
	u := url.URL{Scheme: "wss", Host: "ws.lightstream.bitflyer.com", Path: "/json-rpc"}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	channel := fmt.Sprintf("lightning_ticker_%s", symbol)

	// cにJsonRPC2を書き込む
	if err := c.WriteJSON(&JsonRPC2{Version: "2.0", Method: "subscribe", Params: &SubscribeParams{channel}}); err != nil {
		log.Fatal("subscribe:", err)
		return
	}

OUTER:
	for {
		message := new(JsonRPC2)
		// &{  <nil> <nil> <nil>}

		if err := c.ReadJSON(message); err != nil {
			log.Println("read:", err)
			return
		}
		// message:
		// &{2.0
		//   channelMessage
		//   map[channel:lightning_ticker_BTC_JPY
		// 	     message:map[best_ask:3.608338e+06 best_ask_size:0.95 best_bid:3.605808e+06 best_bid_size:0.05 ltp:3.6078e+06 market_ask_size:0 market_bid_size:0 product_code:BTC_JPY state:RUNNING tick_id:1.609472e+06 timestamp:2021-01-06T17:36:49.6636152Z total_ask_depth:611.6120311 total_bid_depth:1270.51372065 volume:14052.36146998 volume_by_product:14052.36146998]
		//   ]
		//  <nil>
		//  <nil>
		// }

		if message.Method == "channelMessage" {
			switch v := message.Params.(type) {
			// message.Params:
			// map[channel:lightning_ticker_BTC_JPY
			//     message:map[best_ask:3.608338e+06 best_ask_size:0.95 best_bid:3.605808e+06 best_bid_size:0.05 ltp:3.6078e+06 market_ask_size:0 market_bid_size:0 product_code:BTC_JPY state:RUNNING tick_id:1.609472e+06 timestamp:2021-01-06T17:36:49.6636152Z total_ask_depth:611.6120311 total_bid_depth:1270.51372065 volume:14052.36146998 volume_by_product:14052.36146998]
			//    ]
			case map[string]interface{}:
				for key, binary := range v {
					if key == "message" {
						// binary:
						// map[best_ask:3.604469e+06 best_ask_size:0.2164 best_bid:3.602e+06 best_bid_size:0.08 ltp:3.604628e+06 market_ask_size:0 market_bid_size:0 product_code:BTC_JPY state:RUNNING tick_id:1.672529e+06 timestamp:2021-01-06T18:22:38.8030207Z total_ask_depth:595.00775963 total_bid_depth:1259.28180002 volume:13994.73045442 volume_by_product:13994.73045442]

						// map → []byte
						marshaTic, err := json.Marshal(binary)
						if err != nil {
							continue OUTER
						}

						var ticker Ticker
						// []byte → struct
						if err := json.Unmarshal(marshaTic, &ticker); err != nil {
							continue OUTER
						}
						// ticker:
						// &{BTC_JPY RUNNING 2021-01-06T18:22:38.8030207Z 1672529 3.602e+06 3.604469e+06 0.08 0.2164 1259.28180002 595.00775963 0 0 3.604628e+06 13994.73045442 13994.73045442}
						ch <- ticker
					}
				}
			}
		}
	}
}

type Order struct {
	ID                     int     `json:"id"`
	ChildOrderAcceptanceID string  `json:"child_order_acceptance_id"`
	ProductCode            string  `json:"product_code"`
	ChildOrderType         string  `json:"child_order_type"`
	Side                   string  `json:"side"`
	Price                  float64 `json:"price"`
	Size                   float64 `json:"size"`
	MinuteToExpires        int     `json:"minute_to_expire"`
	TimeInForce            string  `json:"time_in_force"`
	Status                 string  `json:"status"`
	ErrorMessage           string  `json:"error_message"`
	AveragePrice           float64 `json:"average_price"`
	ChildOrderState        string  `json:"child_order_state"`
	ExpireDate             string  `json:"expire_date"`
	ChildOrderDate         string  `json:"child_order_date"`
	OutstandingSize        float64 `json:"outstanding_size"`
	CancelSize             float64 `json:"cancel_size"`
	ExecutedSize           float64 `json:"executed_size"`
	TotalCommission        float64 `json:"total_commission"`
	Count                  int     `json:"count"`
	Before                 int     `json:"before"`
	After                  int     `json:"after"`
}

type ResponseSendChildOrder struct {
	ChildOrderAcceptanceID string `json:"child_order_acceptance_id"`
}

// 注文
func (api *APIClient) SendOrder(order *Order) (*ResponseSendChildOrder, error) {
	// struct(*ResponseSendChildOrder)に関してはポインタで返してやった方がオーバーヘッドがない
	// ポインタ型を返すと「コピーを作って渡す」というものではないので一般的にはポインタ型を渡して返すことが多い
	// （実際にこの値を書き換えるかというわけではないがAPIの返り値でもこのような表現がある）
	data, err := json.Marshal(order)
	if err != nil {
		return nil, err
	}
	url := "me/sendchildorder"
	resp, err := api.doRequest("POST", url, map[string]string{}, data)
	if err != nil {
		return nil, err
	}
	var response ResponseSendChildOrder
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

// 注文一覧取得
func (api *APIClient) ListOrder(query map[string]string) ([]Order, error) {
	// スライス([]Order)は既定配列に対するエイリアスを作成するといこうことなのでポインタとして返さなくて良い
	// （ポインタとして使ってるのと一緒）
	resp, err := api.doRequest("GET", "me/getchildorders", query, nil)
	if err != nil {
		return nil, err
	}
	var responseListOrder []Order
	err = json.Unmarshal(resp, &responseListOrder)
	if err != nil {
		return nil, err
	}
	return responseListOrder, nil
}
