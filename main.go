package main

import (
	"trading/app/controllers"
	"trading/config"
	"trading/utils"
)

func main() {
	utils.LoggingSettings(config.Config.LogFile)

	// apiClient := bitflyer.New(config.Config.ApiKey, config.Config.ApiSecret)

	/*
		資産残高取得　GET /v1/me/getbalance
	*/
	// fmt.Println(apiClient.GetBalance())
	// fmt.Println("---------------------------------------------------------")

	/*
		Ticker取得　GET /v1/ticker
	*/
	// ticker, _ := apiClient.GetTicker("BTC_USD")
	// fmt.Println(ticker)
	// fmt.Println(ticker.GetMidPrice())
	// fmt.Println(ticker.DateTime())
	// fmt.Println(ticker.TruncateDateTime(time.Hour))
	// fmt.Println("---------------------------------------------------------")

	/*
		Tickerリアルタイム取得&DB登録
	*/
	controllers.StreamIngestionData()

	/*
		Webサーバー起動
	*/
	controllers.StartWebServer()

	/*
		Tickerリアルタイム取得
		JSON-RPC 2.0 over WebSocket
	*/
	// tickerChannel := make(chan bitflyer.Ticker)
	// go apiClient.GetRealTimeTicker(config.Config.ProductCode, tickerChannel)
	// for ticker := range tickerChannel {
	// 	fmt.Println(ticker)
	// 	fmt.Println(ticker.GetMidPrice())
	// 	fmt.Println(ticker.DateTime())
	// 	fmt.Println(ticker.TruncateDateTime(time.Second))
	// 	fmt.Println(ticker.TruncateDateTime(time.Minute))
	// 	fmt.Println(ticker.TruncateDateTime(time.Hour))
	// }

	/*
		新規注文を出す　POST /v1/me/sendchildorder
	*/
	// order := &bitflyer.Order{
	// 	ProductCode:     config.Config.ProductCode,
	// 	ChildOrderType:  "LIMIT",
	// 	Side:            "BUY",
	// 	Price:           7000,
	// 	Size:            0.01,
	// 	MinuteToExpires: 1,
	// 	TimeInForce:     "GTC", // キャンセルするまで有効
	// }
	// res, _ := apiClient.SendOrder(order)
	// fmt.Println(res.ChildOrderAcceptanceID)

	/*
		注文の一覧を取得　GET /v1/me/getchildorders
	*/
	// i := "JRF20181012-144016-140584"
	// params := map[string]string{
	// 	"product_code":              config.Config.ProductCode,
	// 	"child_order_acceptance_id": i,
	// }
	// r, _ := apiClient.ListOrder(params)
	// fmt.Println(r)

	/*
		DBテーブル作成 base.goのinitを実行
	*/
	// fmt.Println(models.DBConnection)
}
