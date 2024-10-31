package stockclient

// Code adapted from https://github.com/sklinkert/alphavantage

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"time"
)

var (
	BaseURL = "https://www.alphavantage.co"
)

type TimeSeriesData struct {
	Close float64 `json:"4. close,string"`
}

// TimeSeries represents the overall struct for time series
type TimeSeries struct {
	TimeSeriesDaily map[string]TimeSeriesData `json:"Time Series (Daily)"`
}

type ErrorResponse struct {
	ErrorMessage *string `json:"Error Message"`
}

type StockClient struct {
	apiKey     string
	httpClient *http.Client
}

func NewAlphaVantageClient(apiKey string) (*StockClient, error) {
	httpClient := &http.Client{
		Timeout: time.Second * 30,
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 5,
		},
	}

	return &StockClient{
		apiKey:     apiKey,
		httpClient: httpClient,
	}, nil
}

func toStruct(buf []byte) ([]*DayData, error) {
	errResp := &ErrorResponse{}
	err := json.Unmarshal(buf, errResp)
	if err == nil {
		if errResp.ErrorMessage != nil {
			return nil, fmt.Errorf("failed to get stock data: %s", *errResp.ErrorMessage)
		}
	}

	// TODO: Is returning no data always an error?
	timeSeries := &TimeSeries{}
	if err := json.Unmarshal(buf, timeSeries); err != nil {
		return nil, err
	}

	dailyData := []*DayData{}
	for dateStr, data := range timeSeries.TimeSeriesDaily {
		date, err := time.Parse(time.DateOnly, dateStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse date '%s' from response JSON: %w", dateStr, err)
		}

		dayData := &DayData{
			Date:  date,
			Close: data.Close,
		}
		dailyData = append(dailyData, dayData)
	}

	return dailyData, nil
}

func (c *StockClient) Stock(symbol string, sortOrder Order) (*Stock, error) {
	url := fmt.Sprintf("%s/query?function=TIME_SERIES_DAILY&symbol=%s&apikey=%s&outputsize=%s", BaseURL, symbol, c.apiKey, "full")
	body, _, err := c.makeHTTPRequest(url)
	if err != nil {
		return nil, err
	}

	// Useful for manual testing
	// body := []byte(`{
	// 	"Meta Data": {
	// 		"1. Information": "Weekly Prices (open, high, low, close) and Volumes",
	// 		"2. Symbol": "SIX2.DEX",
	// 		"3. Last Refreshed": "2019-09-20",
	// 		"4. Output Size": "Compact",
	// 		"5. Time Zone": "US/Eastern"
	// 	},
	// 	"Time Series (Daily)": {
	// 		"2019-09-20": {
	// 			"1. open": "93.2500",
	// 			"2. high": "94.2000",
	// 			"3. low": "89.5500",
	// 			"4. close": "90.3500",
	// 			"5. volume": "199054"
	// 		},
	// 		"2019-09-13": {
	// 			"1. open": "92.3000",
	// 			"2. high": "95.4000",
	// 			"3. low": "91.5000",
	// 			"4. close": "94.4000",
	// 			"5. volume": "254033"
	// 		}
	// 	}
	// }
	// `)

	dailyData, err := toStruct(body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	sort(dailyData, sortOrder)

	return &Stock{
		DailyData: dailyData,
	}, nil
}

func sort(dailyData []*DayData, sortOrder Order) {
	if sortOrder == Ascending {
		slices.SortFunc(dailyData, func(a, b *DayData) int {
			if a.Date.Before(b.Date) {
				return 1
			}
			if a.Date.After(b.Date) {
				return -1
			}
			return 0
		})
	} else {
		slices.SortFunc(dailyData, func(a, b *DayData) int {
			if a.Date.Before(b.Date) {
				return -1
			}
			if a.Date.After(b.Date) {
				return 1
			}
			return 0
		})
	}
}

func (c *StockClient) makeHTTPRequest(url string) ([]byte, int, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("building http request failed: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("reading response failed: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, fmt.Errorf("unexpected status code: expected %d, got %d",
			http.StatusOK, resp.StatusCode)
	}

	return body, resp.StatusCode, nil
}
