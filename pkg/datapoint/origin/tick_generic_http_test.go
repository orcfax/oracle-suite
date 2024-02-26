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

func TestGenericHTTP_FetchDataPoints(t *testing.T) {
	const testPayload string = "{\"test_data\": true}"
	testCases := []struct {
		name           string
		pairs          []any
		options        TickGenericHTTPConfig
		expectedResult map[any]datapoint.Point
		expectedURLs   []string
	}{
		{
			name:  "simple test",
			pairs: []any{value.Pair{Base: "BTC", Quote: "USD"}},
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
			expectedResult: map[any]datapoint.Point{
				value.Pair{Base: "BTC", Quote: "USD"}: {
					Value: value.NewTick(value.Pair{Base: "BTC", Quote: "USD"}, 1000, 100),
					Time:  time.Date(2023, 5, 2, 12, 34, 56, 0, time.UTC),
				},
			},
			expectedURLs: []string{"/?base=BTC&quote=USD"},
		},
		{
			name:  "one url for all pairs",
			pairs: []any{value.Pair{Base: "BTC", Quote: "USD"}, value.Pair{Base: "ETH", Quote: "USD"}},
			options: TickGenericHTTPConfig{
				URL: "/dataPoints",
				Callback: func(ctx context.Context, pairs []value.Pair, body io.Reader) (map[any]datapoint.Point, error) {
					return map[any]datapoint.Point{
						value.Pair{Base: "BTC", Quote: "USD"}: {
							Value: value.NewTick(value.Pair{Base: "BTC", Quote: "USD"}, 1000, 100),
							Time:  time.Date(2023, 5, 2, 12, 34, 56, 0, time.UTC),
						},
						value.Pair{Base: "ETH", Quote: "USD"}: {
							Value: value.NewTick(value.Pair{Base: "ETH", Quote: "USD"}, 2000, 200),
							Time:  time.Date(2023, 5, 2, 12, 34, 56, 0, time.UTC),
						},
					}, nil
				},
			},
			expectedResult: map[any]datapoint.Point{
				value.Pair{Base: "BTC", Quote: "USD"}: {
					Value: value.NewTick(value.Pair{Base: "BTC", Quote: "USD"}, 1000, 100),
					Time:  time.Date(2023, 5, 2, 12, 34, 56, 0, time.UTC),
				},
				value.Pair{Base: "ETH", Quote: "USD"}: {
					Value: value.NewTick(value.Pair{Base: "ETH", Quote: "USD"}, 2000, 200),
					Time:  time.Date(2023, 5, 2, 12, 34, 56, 0, time.UTC),
				},
			},
			expectedURLs: []string{"/dataPoints"},
		},
		{
			name:  "one url per pair",
			pairs: []any{value.Pair{Base: "BTC", Quote: "USD"}, value.Pair{Base: "ETH", Quote: "USD"}},
			options: TickGenericHTTPConfig{
				URL: "/?base=${ucbase}&quote=${ucquote}",
				Callback: func(ctx context.Context, pairs []value.Pair, body io.Reader) (map[any]datapoint.Point, error) {
					if len(pairs) != 1 {
						t.Fatal("expected one pair")
					}
					switch pairs[0].String() {
					case "BTC/USD":
						return map[any]datapoint.Point{
							value.Pair{Base: "BTC", Quote: "USD"}: {
								Value: value.NewTick(value.Pair{Base: "BTC", Quote: "USD"}, 1000, 100),
								Time:  time.Date(2023, 5, 2, 12, 34, 56, 0, time.UTC),
							},
						}, nil
					case "ETH/USD":
						return map[any]datapoint.Point{
							value.Pair{Base: "ETH", Quote: "USD"}: {
								Value: value.NewTick(value.Pair{Base: "ETH", Quote: "USD"}, 2000, 200),
								Time:  time.Date(2023, 5, 2, 12, 34, 56, 0, time.UTC),
							},
						}, nil
					}
					return nil, nil
				},
			},
			expectedResult: map[any]datapoint.Point{
				value.Pair{Base: "BTC", Quote: "USD"}: {
					Value: value.NewTick(value.Pair{Base: "BTC", Quote: "USD"}, 1000, 100),
					Time:  time.Date(2023, 5, 2, 12, 34, 56, 0, time.UTC),
				},
				value.Pair{Base: "ETH", Quote: "USD"}: {
					Value: value.NewTick(value.Pair{Base: "ETH", Quote: "USD"}, 2000, 200),
					Time:  time.Date(2023, 5, 2, 12, 34, 56, 0, time.UTC),
				},
			},
			expectedURLs: []string{"/?base=BTC&quote=USD", "/?base=ETH&quote=USD"},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server.
			var requests []*http.Request
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests = append(requests, r)
				fmt.Fprint(w, testPayload)
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
			// Because of changing order in `requests` sometimes, the assertion of comparing urls may have a break.
			sort.Slice(requests, func(i, j int) bool {
				return requests[i].URL.String() < requests[j].URL.String()
			})
			for i, url := range tt.expectedURLs {
				assert.Equal(t, url, requests[i].URL.String())
			}
			if len(points) <= 0 {
				t.Error("data points must not be nil for test")
			}
			for i, dataPoint := range points {
				assert.Equal(t, tt.expectedResult[i].Value.Print(), dataPoint.Value.Print())
				assert.Equal(t, tt.expectedResult[i].Time, dataPoint.Time)
			}
			// Provide some basic tests for the existence of the response
			// headers and body that are being set for Orcfax.
			for _, dataPoint := range points {
				var httpResponse httpResponse
				json.Unmarshal(dataPoint.Meta["response"].([]byte), &httpResponse)
				if httpResponse.Headers == nil {
					t.Errorf("rudimentary decode of header failed")
				}
				if httpResponse.Body != testPayload {
					t.Errorf("response body doesn't match payload")
				}
			}
		})
	}
}
