package controllers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"trading/app/models"
	"trading/config"
)

var templates = template.Must(template.ParseFiles("app/views/google.html", "app/views/chart.html"))

/*
  チャート表示(サンプル)
  http://localhost:8080/example/
*/
func viewExampleHandler(w http.ResponseWriter, r *http.Request) {
	limit := 100
	duration := "1m"
	durationTime := config.Config.Durations[duration]
	df, _ := models.GetAllCandle(config.Config.ProductCode, durationTime, limit)

	err := templates.ExecuteTemplate(w, "google.html", df.Candles)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

/*
  チャート表示(Ajax非同期)
  http://localhost:8080/chart/
*/
func viewChartHandler(w http.ResponseWriter, r *http.Request) {
	err := templates.ExecuteTemplate(w, "chart.html", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type JSONError struct {
	Error string `json:"error"`
	Code  int    `json:"code"`
}

// JSONで何かエラーがあった時にはJSON型でエラーを返す
func APIError(w http.ResponseWriter, errMessage string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	jsonError, err := json.Marshal(JSONError{Error: errMessage, Code: code})
	if err != nil {
		log.Fatal(err)
	}
	w.Write(jsonError)
}

// /api/candle/を呼ぶアクセス
var apiValidPath = regexp.MustCompile("^/api/candle/$")

func apiMakeHandler(fn func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := apiValidPath.FindStringSubmatch(r.URL.Path)
		if len(m) == 0 {
			APIError(w, "Not found", http.StatusNotFound)
		}
		fn(w, r)
	}
}

/*
  DBの値を取得してJSONにして値を返すAPI
  http://localhost:8080/candle/?product_code=BTC_JPY&duration=1h&limit=1

  レスポンス
	{
		"product_code": "BTC_JPY",
		"duration": 3600000000000,
		"candles": [
			{
				"product_code": "BTC_JPY",
				"duration": 3600000000000,
				"time": "2021-01-11T06:00:00Z",
				"open": 3527323.5,
				"close": 3567013,
				"high": 3567013,
				"low": 3520259.5,
				"volume": 3060504.8185722404
			}
		]
	}
*/
func apiCandleHandler(w http.ResponseWriter, r *http.Request) {
	productCode := r.URL.Query().Get("product_code")
	if productCode == "" {
		APIError(w, "No product_code param", http.StatusBadRequest)
		return
	}

	strLimit := r.URL.Query().Get("limit")
	limit, err := strconv.Atoi(strLimit)
	if strLimit == "" || err != nil || limit < 0 || limit > 1000 {
		limit = 1000
	}

	duration := r.URL.Query().Get("duration")
	if duration == "" {
		duration = "1m"
	}
	durationTime := config.Config.Durations[duration]

	df, _ := models.GetAllCandle(productCode, durationTime, limit)

	js, err := json.Marshal(df)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func StartWebServer() error {
	http.HandleFunc("/api/candle/", apiMakeHandler(apiCandleHandler))
	http.HandleFunc("/chart/", viewChartHandler)
	http.HandleFunc("/example/", viewExampleHandler)
	return http.ListenAndServe(fmt.Sprintf(":%d", config.Config.Port), nil)
}
