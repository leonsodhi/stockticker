package stockclient

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStock(t *testing.T) {
	successResp := `{
		"Meta Data": {
			"1. Information": "Weekly Prices (open, high, low, close) and Volumes",
			"2. Symbol": "SIX2.DEX",
			"3. Last Refreshed": "2019-09-20",
			"4. Output Size": "Compact",
			"5. Time Zone": "US/Eastern"
		},
		"Time Series (Daily)": {
			"2019-09-20": {
				"1. open": "93.2500",
				"2. high": "94.2000",
				"3. low": "89.5500",
				"4. close": "90.3500",
				"5. volume": "199054"
			},
			"2019-09-13": {
				"1. open": "92.3000",
				"2. high": "95.4000",
				"3. low": "91.5000",
				"4. close": "94.4000",
				"5. volume": "254033"
			}
		}
	}`

	jsonErrResp := `{
		"Error Message": "Invalid API call. Please retry or visit the documentation (https://www.alphavantage.co/documentation/) for TIME_SERIES_DAILY."
	}`

	invalidJson := "INVALID_JSON"

	resp := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, r.URL.Path, "/query")

		params := r.URL.Query()
		require.Contains(t, params, "function")
		require.Contains(t, params, "symbol")
		require.Contains(t, params, "apikey")
		require.Contains(t, params, "outputsize")
		require.Equal(t, "TIME_SERIES_DAILY", params["function"][0])
		require.Equal(t, "DUMMY_SYMBOL", params["symbol"][0])
		require.Equal(t, "DUMMY_API_KEY", params["apikey"][0])
		require.Equal(t, "full", params["outputsize"][0])

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(resp))
	}))
	defer server.Close()

	t.Run("Client request with a success response", func(t *testing.T) {
		resp = successResp
		BaseURL = server.URL
		client, err := NewAlphaVantageClient("DUMMY_API_KEY")
		require.NoError(t, err)

		stock, err := client.Stock("DUMMY_SYMBOL", Ascending)
		require.NoError(t, err)
		require.Len(t, stock.DailyData, 2)

		t1, _ := time.Parse(time.DateOnly, "2019-09-20")
		require.Equal(t, t1, stock.DailyData[0].Date)
		require.Equal(t, 90.35, stock.DailyData[0].Close)

		t2, _ := time.Parse(time.DateOnly, "2019-09-13")
		require.Equal(t, t2, stock.DailyData[1].Date)
		require.Equal(t, 94.4, stock.DailyData[1].Close)
	})

	t.Run("Client request with a JSON error response", func(t *testing.T) {
		resp = jsonErrResp
		BaseURL = server.URL
		client, err := NewAlphaVantageClient("DUMMY_API_KEY")
		require.NoError(t, err)

		_, err = client.Stock("DUMMY_SYMBOL", Ascending)
		require.Error(t, err)
	})

	t.Run("Client request with an invalid JSON response", func(t *testing.T) {
		resp = invalidJson
		BaseURL = server.URL
		client, err := NewAlphaVantageClient("DUMMY_API_KEY")
		require.NoError(t, err)

		_, err = client.Stock("DUMMY_SYMBOL", Ascending)
		require.Error(t, err)
	})
}
