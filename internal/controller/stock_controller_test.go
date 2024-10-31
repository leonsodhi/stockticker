package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"stockticker/internal/cache"
	"stockticker/internal/stockclient"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/stretchr/testify/require"
)

var (
	dailyData = []*stockclient.DayData{
		{
			Date:  time.Date(2019, 9, 20, 0, 0, 0, 0, time.UTC),
			Close: 90.35,
		},

		{
			Date:  time.Date(2019, 9, 19, 0, 0, 0, 0, time.UTC),
			Close: 94.4,
		},
	}

	cachedDailyData = []*stockclient.DayData{
		{
			Date:  time.Date(2020, 10, 3, 0, 0, 0, 0, time.UTC),
			Close: 200.65,
		},
		{
			Date:  time.Date(2020, 10, 2, 0, 0, 0, 0, time.UTC),
			Close: 10.35,
		},
		{
			Date:  time.Date(2020, 10, 1, 0, 0, 0, 0, time.UTC),
			Close: 360.4,
		},
	}
)

// Mock stock client
type mockStockClient struct {
	Symbol string
}

func (sc *mockStockClient) Stock(symbol string, sortOrder stockclient.Order) (*stockclient.Stock, error) {
	sc.Symbol = symbol
	stock := &stockclient.Stock{DailyData: dailyData}
	return stock, nil
}

// Mock cache
type mockCacheClient struct {
	GetKey string
	Cache  map[string]string
}

func NewMockCacheClient() *mockCacheClient {
	return &mockCacheClient{
		Cache: make(map[string]string),
	}
}

func (c *mockCacheClient) Get(ctx context.Context, key string) (string, error) {
	c.GetKey = key
	return c.Cache[key], nil
}

func (c *mockCacheClient) Set(ctx context.Context, key, val string, ttl time.Duration) error {
	c.Cache[key] = val
	return nil
}

func (c *mockCacheClient) Close() {}

// Mock failing cache
type mockFailingCacheClient struct {
	Key string
}

func (c *mockFailingCacheClient) Get(ctx context.Context, key string) (string, error) {
	c.Key = key
	return "", fmt.Errorf("Get failure")
}

func (c *mockFailingCacheClient) Set(ctx context.Context, key, val string, ttl time.Duration) error {
	return fmt.Errorf("Set failure")
}

func (c *mockFailingCacheClient) Close() {}

func assertViewData(t *testing.T, viewData map[string]any, numDaysReq int, expAvgClose float64) {
	require.Contains(t, viewData, "daysReq")
	require.Contains(t, viewData, "daysRet")
	require.Contains(t, viewData, "dailyData")
	require.Contains(t, viewData, "avgClose")

	numDays := numDaysReq
	if numDaysReq > len(dailyData) {
		numDays = len(dailyData)
	}
	require.Equal(t, numDaysReq, viewData["daysReq"])
	require.Equal(t, numDays, viewData["daysRet"])
	require.ElementsMatch(t, dailyData[:numDays], viewData["dailyData"])
	require.Equal(t, expAvgClose, viewData["avgClose"])
}

func assertCachedViewData(t *testing.T, viewData map[string]any) {
	require.Contains(t, viewData, "daysReq")
	require.Contains(t, viewData, "daysRet")
	require.Contains(t, viewData, "dailyData")
	require.Contains(t, viewData, "avgClose")

	require.Equal(t, 3, viewData["daysReq"])
	require.Equal(t, 3, viewData["daysRet"])
	require.ElementsMatch(t, cachedDailyData, viewData["dailyData"])
	require.Equal(t, 190.46666666666666666666666666667, viewData["avgClose"])
}

func TestMain(m *testing.M) {
	log.SetOutput(io.Discard)
	code := m.Run()
	os.Exit(code)
}

func TestStock(t *testing.T) {
	t.Run("Stock with different symbols and days without caching", func(t *testing.T) {
		var tests = []struct {
			symbol      string
			numDays     int
			expAvgClose float64
		}{
			{"TSLA", len(dailyData), 92.375},
			{"TSLA", len(dailyData) - 1, 90.35},
			{"TSLA", len(dailyData) + 1, 92.375},
			{"TSLA", len(dailyData) + 100, 92.375},
			{"MSFT", len(dailyData), 92.375},
			{"MSFT", len(dailyData) - 1, 90.35},
			{"MSFT", len(dailyData) + 1, 92.375},
			{"MSFT", len(dailyData) + 100, 92.375},
		}
		for _, test := range tests {
			stockClient := &mockStockClient{}
			cacheClient, _ := cache.NewNullClient("", 0)
			stockCtrler, err := NewStockController(stockClient, cacheClient, test.symbol, test.numDays)
			require.NoError(t, err)

			ctx := context.Background()
			viewData, err := stockCtrler.Stock(ctx)
			require.Equal(t, test.symbol, stockClient.Symbol)
			require.NoError(t, err)
			assertViewData(t, viewData, test.numDays, test.expAvgClose)
		}
	})

	t.Run("Stock where result for MSFT and 2 days will be cached", func(t *testing.T) {
		ctx := context.Background()
		stockClient := &mockStockClient{}
		cacheClient := NewMockCacheClient()

		stockCtrler, err := NewStockController(stockClient, cacheClient, "NVDA", 2)
		require.NoError(t, err)

		stockStr, _ := cacheClient.Get(ctx, "symbol:NVDA")
		require.Empty(t, stockStr)

		viewData, err := stockCtrler.Stock(ctx)
		require.NoError(t, err)

		stockStr, _ = cacheClient.Get(ctx, "symbol:NVDA")
		var stock stockclient.Stock
		err = json.Unmarshal([]byte(stockStr), &stock)
		require.NoError(t, err)
		require.ElementsMatch(t, stock.DailyData, dailyData)

		assertViewData(t, viewData, 2, 92.375)
	})

	t.Run("Stock with cached result for NVDA and 3 days", func(t *testing.T) {
		ctx := context.Background()
		stockClient := &mockStockClient{}
		cacheClient := NewMockCacheClient()

		data, err := json.Marshal(&stockclient.Stock{
			DailyData: cachedDailyData,
		})
		require.NoError(t, err)
		cacheClient.Set(ctx, "symbol:NVDA", string(data), 100*time.Hour)

		stockCtrler, err := NewStockController(stockClient, cacheClient, "NVDA", 3)
		require.NoError(t, err)

		viewData, err := stockCtrler.Stock(ctx)
		require.Equal(t, "symbol:NVDA", cacheClient.GetKey)
		require.NoError(t, err)
		assertCachedViewData(t, viewData)
	})

	t.Run("Stock with failing caching for AAPL and 2 days", func(t *testing.T) {
		stockClient := &mockStockClient{}
		cacheClient := &mockFailingCacheClient{}
		stockCtrler, err := NewStockController(stockClient, cacheClient, "AAPL", 2)
		require.NoError(t, err)

		ctx := context.Background()
		viewData, err := stockCtrler.Stock(ctx)
		require.Equal(t, "AAPL", stockClient.Symbol)
		require.NoError(t, err)
		assertViewData(t, viewData, 2, 92.375)
	})
}
