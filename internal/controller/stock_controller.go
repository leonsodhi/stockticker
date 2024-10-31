package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"stockticker/internal/cache"
	"stockticker/internal/stockclient"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	// TODO: May want separate timeouts for reading and writing. Probably also want to make this a command line switch or env var
	CACHE_TIMEOUT = 15
)

type StockController struct {
	client  stockclient.Client
	numDays int
	symbol  string
	cache   cache.Client
}

func NewStockController(client stockclient.Client, cache cache.Client, symbol string, numDays int) (*StockController, error) {
	return &StockController{
		client:  client,
		numDays: numDays,
		symbol:  symbol,
		cache:   cache,
	}, nil
}

func (sc *StockController) Stock(ctx context.Context) (map[string]any, error) {
	var err error
	var stock *stockclient.Stock

	cacheCtx, cancel := context.WithTimeout(ctx, CACHE_TIMEOUT*time.Second)
	defer cancel()
	stock, cacheErr := sc.cachedStock(cacheCtx)
	if cacheErr != nil {
		log.Warnf("Failed to get stock from cache: %v", cacheErr)
	}

	if stock == nil {
		log.Debug("Response not cached")
		timer := prometheus.NewTimer(stockClientTimer.WithLabelValues("daily"))
		// TODO: Distributed rate limiting might be useful here depending on how the third-party implement rate limiting
		stock, err = sc.client.Stock(sc.symbol, stockclient.Ascending)
		timer.ObserveDuration()
		if err != nil {
			stockClientErrors.WithLabelValues("daily").Inc()
			return nil, err
		}

		// TODO: Is there a way to detect if the provider is lagged and cache for less time?
		if cacheErr == nil {
			ttl := cacheTTL()
			log.Debugf("Caching response with TTL: %v", ttl)
			cacheErr = sc.cacheStock(cacheCtx, stock, ttl)
			if cacheErr != nil {
				log.Warnf("Failed to cache stock: %v", cacheErr)
			}
		}
	} else {
		log.Debug("Response cached")
	}

	numDays := min(sc.numDays, len(stock.DailyData))
	log.Debugf("numDays: %d", numDays)
	nDaysOfDailyData := stock.DailyData[:numDays]
	viewData := map[string]any{
		"daysReq":   sc.numDays,
		"daysRet":   numDays,
		"dailyData": nDaysOfDailyData,
		"avgClose":  sc.avgClosePrice(nDaysOfDailyData),
	}
	return viewData, nil
}

// cachedStock attempts to get stock data from cache
func (sc *StockController) cachedStock(ctx context.Context) (*stockclient.Stock, error) {
	timer := prometheus.NewTimer(stockCacheTimer.WithLabelValues("read"))
	defer timer.ObserveDuration()

	stockStr, err := sc.cache.Get(ctx, fmt.Sprintf("symbol:%s", sc.symbol))
	if err != nil {
		stockCacheErrors.WithLabelValues("read").Inc()
		return nil, err
	}
	if stockStr == "" {
		return nil, nil
	}

	var stock stockclient.Stock
	err = json.Unmarshal([]byte(stockStr), &stock)
	if err != nil {
		return nil, err
	}
	return &stock, nil
}

// cacheStock caches the provided stock data
func (sc *StockController) cacheStock(ctx context.Context, stock *stockclient.Stock, ttl time.Duration) error {
	timer := prometheus.NewTimer(stockCacheTimer.WithLabelValues("write"))
	defer timer.ObserveDuration()

	data, err := json.Marshal(stock)
	if err != nil {
		return err
	}
	// TODO: Worth compressing before caching?
	err = sc.cache.Set(ctx, fmt.Sprintf("symbol:%s", sc.symbol), string(data), ttl)
	if err != nil {
		stockCacheErrors.WithLabelValues("write").Inc()
		return err
	}

	return nil
}

func cacheTTL() time.Duration {
	// TODO: This will almost certainly result in stale data being returned for multiple hours. Does that matter? Is there a better way?
	tomorrow := time.Now().AddDate(0, 0, 1)
	midnightTomorrow := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 5, 0, 0, tomorrow.Location())
	return time.Until(midnightTomorrow)
}

func (sc *StockController) avgClosePrice(dailyData []*stockclient.DayData) float64 {
	avgClose := float64(0)
	for _, d := range dailyData {
		avgClose += d.Close
	}

	avgClose /= float64(len(dailyData))
	return avgClose
}

func (sc *StockController) ViewTemplate() string {
	return "stock_controller_view.tmpl"
}
