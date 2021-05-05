package controllers

import (
	"log"
	"trading/app/models"
	"trading/bitflyer"
	"trading/config"
)

func StreamIngestionData() {
	var tickerChannl = make(chan bitflyer.Ticker)
	apiClient := bitflyer.New(config.Config.ApiKey, config.Config.ApiSecret)

	go apiClient.GetRealTimeTicker(config.Config.ProductCode, tickerChannl)

	go func() {
		for ticker := range tickerChannl {
			log.Printf("action=StreamIngestionData, %v", ticker)
			for _, duration := range config.Config.Durations {
				isCreated := models.CreateCandleWithDuration(ticker, ticker.ProductCode, duration)
				if isCreated == true && duration == config.Config.TradeDuration {
					// 分足のチャートが新しくできたときにトレードするかどうかの判断を実行
				}
			}
		}
	}()
}
