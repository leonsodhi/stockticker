package stockclient

import (
	"time"
)

type Order int

const (
	Ascending Order = iota
	Descending
)

type DayData struct {
	Date  time.Time
	Close float64
}

type Stock struct {
	DailyData []*DayData
}

type Client interface {
	Stock(symbol string, sortOrder Order) (*Stock, error)
}
