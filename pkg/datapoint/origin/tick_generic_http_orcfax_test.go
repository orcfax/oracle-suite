package origin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/orcfax/oracle-suite/pkg/datapoint"
	"github.com/orcfax/oracle-suite/pkg/datapoint/value"
)

type httpResponse struct {
	Headers map[string][]string
	Body    string
}

// Orcfax specific tests mirror those in the generic http test suite
// but we ware focused on ensuring that we have predictable outcomes
// from the headers/response body that we're record upstream in
// archival metadata.
func TestGenericHTTP_FetchDataPointsOrcfax(t *testing.T) {
	testCases := []struct {
		name            string
		pairs           []any
		options         TickGenericHTTPConfig
		payload         string
		expectedURLs    []string
		expectError     bool
		expectedStrings []string
	}{
		{
			name:  "basic key value dict",
			pairs: []any{value.Pair{Base: "ADA", Quote: "USD"}},
			options: TickGenericHTTPConfig{
				URL: "/?base=${ucbase}&quote=${ucquote}",
				Callback: func(ctx context.Context, pairs []value.Pair, body io.Reader) (map[any]datapoint.Point, error) {
					return map[any]datapoint.Point{
						value.Pair{Base: "BTC", Quote: "USD"}: {
							Value: value.NewTick(value.Pair{Base: "BTC", Quote: "USD"}, 1000, 100),
							Time:  time.Date(2023, 5, 2, 12, 34, 56, 0, time.UTC),
						},
					}, nil
				},
			},
			payload:         "{\"x\": \"y\"}",
			expectedURLs:    []string{"/?base=ADA&quote=USD"},
			expectedStrings: []string{"x", "y"},
		},
		{
			name:  "invalid json - incomplete array",
			pairs: []any{value.Pair{Base: "ADA", Quote: "USD"}},
			options: TickGenericHTTPConfig{
				URL: "/?base=${ucbase}&quote=${ucquote}",
				Callback: func(ctx context.Context, pairs []value.Pair, body io.Reader) (map[any]datapoint.Point, error) {
					return map[any]datapoint.Point{
						value.Pair{Base: "BTC", Quote: "USD"}: {
							Value: value.NewTick(value.Pair{Base: "BTC", Quote: "USD"}, 1000, 100),
							Time:  time.Date(2023, 5, 2, 12, 34, 56, 0, time.UTC),
						},
					}, nil
				},
			},
			payload:      "[[]",
			expectedURLs: []string{"/?base=ADA&quote=USD"},
			expectError:  true,
		},
		// ADA/USD expected results.
		{
			name:  "invalid json - incomplete array",
			pairs: []any{value.Pair{Base: "ADA", Quote: "USD"}},
			options: TickGenericHTTPConfig{
				URL: "/?base=${ucbase}&quote=${ucquote}",
				Callback: func(ctx context.Context, pairs []value.Pair, body io.Reader) (map[any]datapoint.Point, error) {
					return map[any]datapoint.Point{
						value.Pair{Base: "BTC", Quote: "USD"}: {
							Value: value.NewTick(value.Pair{Base: "BTC", Quote: "USD"}, 1000, 100),
							Time:  time.Date(2023, 5, 2, 12, 34, 56, 0, time.UTC),
						},
					}, nil
				},
			},
			payload:         "{\"timestamp\":\"1709109657\",\"open\":\"0.62539\",\"high\":\"0.63674\",\"low\":\"0.60778\",\"last\":\"0.63674\",\"volume\":\"1464531.68593555\",\"vwap\":\"0.62291\",\"bid\":\"0.63635\",\"ask\":\"0.63662\",\"side\":\"0\",\"open_24\":\"0.61966\",\"percent_change_24\":\"2.76\"}",
			expectedURLs:    []string{"/?base=ADA&quote=USD"},
			expectedStrings: []string{"open", "low", "timestamp", "last"},
		},
		{
			name:  "invalid json - incomplete array",
			pairs: []any{value.Pair{Base: "ADA", Quote: "USD"}},
			options: TickGenericHTTPConfig{
				URL: "/?base=${ucbase}&quote=${ucquote}",
				Callback: func(ctx context.Context, pairs []value.Pair, body io.Reader) (map[any]datapoint.Point, error) {
					return map[any]datapoint.Point{
						value.Pair{Base: "BTC", Quote: "USD"}: {
							Value: value.NewTick(value.Pair{Base: "BTC", Quote: "USD"}, 1000, 100),
							Time:  time.Date(2023, 5, 2, 12, 34, 56, 0, time.UTC),
						},
					}, nil
				},
			},
			payload:         "[{\"symbol\":\"ADAUSD\",\"ask\":\"0.6361319\",\"bid\":\"0.6360904\",\"last\":\"0.6358728\",\"low\":\"0.6077000\",\"high\":\"0.6363823\",\"open\":\"0.6195687\",\"volume\":\"31965148\",\"volumeQuote\":\"19901988.0035152\",\"timestamp\":\"2024-02-28T08:41:00.046Z\"}]",
			expectedURLs:    []string{"/?base=ADA&quote=USD"},
			expectedStrings: []string{"symbol", "bid", "timestamp", "ADAUSD"},
		},
		{
			name:  "invalid json - incomplete array",
			pairs: []any{value.Pair{Base: "ADA", Quote: "USD"}},
			options: TickGenericHTTPConfig{
				URL: "/?base=${ucbase}&quote=${ucquote}",
				Callback: func(ctx context.Context, pairs []value.Pair, body io.Reader) (map[any]datapoint.Point, error) {
					return map[any]datapoint.Point{
						value.Pair{Base: "BTC", Quote: "USD"}: {
							Value: value.NewTick(value.Pair{Base: "BTC", Quote: "USD"}, 1000, 100),
							Time:  time.Date(2023, 5, 2, 12, 34, 56, 0, time.UTC),
						},
					}, nil
				},
			},
			payload:         "[[\"tADAUSD\",0.63574,221903.80674973,0.63622,93629.40217521,0.01766,0.02859086,0.63534,1662963.79301175,0.63633,0.60648]]",
			expectedURLs:    []string{"/?base=ADA&quote=USD"},
			expectedStrings: []string{"0.63633", "0.63622", "tADAUSD"},
		},
		{
			name:  "invalid json - incomplete array",
			pairs: []any{value.Pair{Base: "ADA", Quote: "USD"}},
			options: TickGenericHTTPConfig{
				URL: "/?base=${ucbase}&quote=${ucquote}",
				Callback: func(ctx context.Context, pairs []value.Pair, body io.Reader) (map[any]datapoint.Point, error) {
					return map[any]datapoint.Point{
						value.Pair{Base: "BTC", Quote: "USD"}: {
							Value: value.NewTick(value.Pair{Base: "BTC", Quote: "USD"}, 1000, 100),
							Time:  time.Date(2023, 5, 2, 12, 34, 56, 0, time.UTC),
						},
					}, nil
				},
			},
			payload:         "{\"error\":[],\"result\":{\"ADA/USD\":{\"a\":[\"0.636508\",\"7\",\"7.000\"],\"b\":[\"0.636462\",\"34\",\"34.000\"],\"c\":[\"0.636498\",\"52.90149400\"],\"v\":[\"2062267.92645757\",\"7606915.49312130\"],\"p\":[\"0.627181\",\"0.622697\"],\"t\":[1712,5800],\"l\":[\"0.615894\",\"0.608149\"],\"h\":[\"0.636508\",\"0.636508\"],\"o\":\"0.624298\"}}}",
			expectedURLs:    []string{"/?base=ADA&quote=USD"},
			expectedStrings: []string{"result", "ADA/USD", "0.636508"},
		},
		{
			name:  "invalid json - incomplete array",
			pairs: []any{value.Pair{Base: "ADA", Quote: "USD"}},
			options: TickGenericHTTPConfig{
				URL: "/?base=${ucbase}&quote=${ucquote}",
				Callback: func(ctx context.Context, pairs []value.Pair, body io.Reader) (map[any]datapoint.Point, error) {
					return map[any]datapoint.Point{
						value.Pair{Base: "BTC", Quote: "USD"}: {
							Value: value.NewTick(value.Pair{Base: "BTC", Quote: "USD"}, 1000, 100),
							Time:  time.Date(2023, 5, 2, 12, 34, 56, 0, time.UTC),
						},
					}, nil
				},
			},
			payload:         "{\"ask\":\"0.6367\",\"bid\":\"0.6366\",\"volume\":\"37424319.11\",\"trade_id\":103975867,\"price\":\"0.6365\",\"size\":\"16.28\",\"time\":\"2024-02-28T08:40:44.036112Z\",\"rfq_volume\":\"2120277.505950\",\"conversions_volume\":\"0\"}",
			expectedURLs:    []string{"/?base=ADA&quote=USD"},
			expectedStrings: []string{"ask", "trade_id", "bid"},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server.
			var requests []*http.Request
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests = append(requests, r)
				fmt.Fprintf(w, tt.payload)
			}))
			defer server.Close()

			// Create the data.
			tt.options.URL = server.URL + tt.options.URL
			gh, err := NewTickGenericHTTP(tt.options)
			require.NoError(t, err)

			// Test the data.
			points, err := gh.FetchDataPoints(context.Background(), tt.pairs)
			require.NoError(t, err)
			require.Len(t, requests, len(tt.expectedURLs))
			// Because of changing order in `requests` sometimes, the
			// assertion of comparing urls may have a break.
			sort.Slice(requests, func(i, j int) bool {
				return requests[i].URL.String() < requests[j].URL.String()
			})
			for i, url := range tt.expectedURLs {
				assert.Equal(t, url, requests[i].URL.String())
			}
			if len(points) <= 0 {
				t.Error("data points must not be nil for test")
			}
			// Provide some more complex tests around the data structure
			// parsed into the response from the collector.
			for _, dataPoint := range points {
				var httpResponse httpResponse
				json.Unmarshal(dataPoint.Meta["response"].([]byte), &httpResponse)
				if httpResponse.Headers["Content-Length"] == nil || httpResponse.Headers["Content-Type"] == nil {
					t.Errorf(
						"expected header fields are not set: content-length: '%s', content-type: '%s'",
						httpResponse.Headers["Content-Length"],
						httpResponse.Headers["Content-Type"],
					)
				}
				if tt.expectedStrings == nil && !tt.expectError {
					t.Errorf("expected strings for test should not be nil")
				}
				if httpResponse.Body != tt.payload {
					t.Error("response body doesn't match payload")
				}
			}
		})
	}
}
