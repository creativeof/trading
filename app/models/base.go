package models

import (
	"database/sql"
	"fmt"
	"log"
	"time"
	"trading/config"

	_ "github.com/mattn/go-sqlite3"
)

const (
	tableNameSignalEvents = "signal_events"
)

var DBConnection *sql.DB

func GetCandleTableName(productCode string, duration time.Duration) string {
	return fmt.Sprintf("%s_%s", productCode, duration)
}

func init() {
	var err error
	DBConnection, err = sql.Open(config.Config.SQLDriver, config.Config.DBName)
	if err != nil {
		log.Fatalln(err)
	}
	cmd := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			time DATETIME PRIMARY KEY NOT NULL,
			product_code STRING,
			side STRING,
			price FLOAT,
			size FLOAT)`, tableNameSignalEvents)
	DBConnection.Exec(cmd)

	for _, duration := range config.Config.Durations {
		// BTC_JPY_1m
		tableName := GetCandleTableName(config.Config.ProductCode, duration)
		c := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			time DATETIME PRIMARY KEY NOT NULL,
			open FLOAT,
			close FLOAT,
			high FLOAT,
			low FLOAT,
			volume FLOAT)`, tableName)
		DBConnection.Exec(c)
	}
}
