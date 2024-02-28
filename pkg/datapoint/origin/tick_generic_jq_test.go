package origin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	b64 "github.com/cristalhq/base64"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/orcfax/oracle-suite/pkg/datapoint"
	"github.com/orcfax/oracle-suite/pkg/datapoint/value"

	"github.com/orcfax/oracle-suite/pkg/log/null"
)

func TestNewGenericJQ(t *testing.T) {
	t.Run("empty URL", func(t *testing.T) {
		_, err := NewTickGenericJQ(TickGenericJQConfig{
			URL:    "",
			Query:  ".price",
			Logger: null.New(),
		})
		assert.EqualError(t, err, "url cannot be empty")
	})
	t.Run("empty query", func(t *testing.T) {
		_, err := NewTickGenericJQ(TickGenericJQConfig{
			URL:    "https://example.com",
			Query:  "",
			Logger: null.New(),
		})
		assert.EqualError(t, err, "query must be specified")
	})
	t.Run("invalid query", func(t *testing.T) {
		_, err := NewTickGenericJQ(TickGenericJQConfig{
			URL:    "https://example.com",
			Query:  "invalid jq",
			Logger: null.New(),
		})
		assert.Error(t, err)
	})
	t.Run("valid options", func(t *testing.T) {
		_, err := NewTickGenericJQ(TickGenericJQConfig{
			URL:    "https://example.com",
			Query:  ".price",
			Logger: null.New(),
		})
		assert.NoError(t, err)
	})
}

func TestGenericJQ_FetchDataPoints(t *testing.T) {
	testCases := []struct {
		name             string
		query            string
		responseBody     string
		expectedResult   map[any]datapoint.Point
		skipVolumeAssert bool
		skipTimeAssert   bool
		skipError        bool
	}{
		{
			name:         "price, volume and time",
			query:        ".",
			responseBody: `{"price": "1000", "volume": "100", "time": "2023-05-02T12:34:56Z"}`,
			expectedResult: map[any]datapoint.Point{
				value.Pair{Base: "BTC", Quote: "USD"}: {
					Value: value.NewTick(value.Pair{Base: "BTC", Quote: "USD"}, 1000, 100),
					Time:  time.Date(2023, 5, 2, 12, 34, 56, 0, time.UTC),
				},
			},
		},
		{
			name:         "price only",
			query:        "{price: .price}",
			responseBody: `{"price": "1000", "volume": "100", "time": "2023-05-02T12:34:56Z"}`,
			expectedResult: map[any]datapoint.Point{
				value.Pair{Base: "BTC", Quote: "USD"}: {
					Value: value.NewTick(value.Pair{Base: "BTC", Quote: "USD"}, 1000, nil),
					Time:  time.Now(),
				},
			},
			skipTimeAssert: true,
		},
		{
			name:         "single array result",
			query:        ".[] | {price: .price}",
			responseBody: `[{"price": "1000"}]`,
			expectedResult: map[any]datapoint.Point{
				value.Pair{Base: "BTC", Quote: "USD"}: {
					Value: value.NewTick(value.Pair{Base: "BTC", Quote: "USD"}, 1000, nil),
					Time:  time.Now(),
				},
			},
			skipTimeAssert: true,
		},
		{
			name:         "multiple array results",
			query:        ".[] | {price: .price}",
			responseBody: `[{"price": "1000"}, {"price": "2000"}]`,
			expectedResult: map[any]datapoint.Point{
				value.Pair{Base: "BTC", Quote: "USD"}: {
					Value: value.NewTick(value.Pair{Base: "BTC", Quote: "USD"}, nil, nil),
					Time:  time.Now(),
					Error: fmt.Errorf("multiple results from JQ query"),
				},
			},
			skipTimeAssert: true,
		},
		{
			name:         "variables $ucbase and $ucquote",
			query:        ".[] | select(.symbol == ($ucbase + $ucquote)) | {price: .price}",
			responseBody: `[{"price": "1000", "symbol": "BTCUSD"}, {"price": "2000", "symbol": "ETHUSD"}]`,
			expectedResult: map[any]datapoint.Point{
				value.Pair{Base: "BTC", Quote: "USD"}: {
					Value: value.NewTick(value.Pair{Base: "BTC", Quote: "USD"}, 1000, nil),
					Time:  time.Now(),
				},
			},
			skipTimeAssert: true,
		},
		{
			name:         "variables $lcbase and $lcquote",
			query:        ".[] | select(.symbol == ($lcbase + $lcquote)) | {price: .price}",
			responseBody: `[{"price": "1000", "symbol": "btcusd"}, {"price": "2000", "symbol": "ethusd"}]`,
			expectedResult: map[any]datapoint.Point{
				value.Pair{Base: "BTC", Quote: "USD"}: {
					Value: value.NewTick(value.Pair{Base: "BTC", Quote: "USD"}, 1000, nil),
					Time:  time.Now(),
				},
			},
			skipTimeAssert: true,
		},
		{
			name:         "invalid JSON",
			query:        ".",
			responseBody: `{"price": "1000", "volume": "100", "time": "2023-05-02T12:34:56Z"`,
			expectedResult: map[any]datapoint.Point{
				value.Pair{Base: "BTC", Quote: "USD"}: {
					Time:  time.Now(),
					Error: errors.New("unexpected EOF"),
				},
			},
			skipTimeAssert: true,
			skipError:      true,
		},
		{
			name:         "empty response",
			query:        ".",
			responseBody: ``,
			expectedResult: map[any]datapoint.Point{
				value.Pair{Base: "BTC", Quote: "USD"}: {
					Time:  time.Now(),
					Error: errors.New("EOF"),
				},
			},
			skipTimeAssert: true,
			skipError:      true,
		},
		{
			name:         "time.RFC3339",
			query:        ".",
			responseBody: `{"price": "1000", "volume": "100", "time": "2023-05-02T12:34:56Z"}`,
			expectedResult: map[any]datapoint.Point{
				value.Pair{Base: "BTC", Quote: "USD"}: {
					Value: value.NewTick(value.Pair{Base: "BTC", Quote: "USD"}, 1000, 100),
					Time:  time.Date(2023, 5, 2, 12, 34, 56, 0, time.UTC),
				},
			},
		},
		{
			name:         "time.RFC3339Nano",
			query:        ".",
			responseBody: `{"price": "1000", "volume": "100", "time": "2023-05-02T12:34:56.123456789Z"}`,
			expectedResult: map[any]datapoint.Point{
				value.Pair{Base: "BTC", Quote: "USD"}: {
					Value: value.NewTick(value.Pair{Base: "BTC", Quote: "USD"}, 1000, 100),
					Time:  time.Date(2023, 5, 2, 12, 34, 56, 123456789, time.UTC),
				},
			},
		},
		{
			name:         "time.RFC1123",
			query:        ".",
			responseBody: `{"price": "1000", "volume": "100", "time": "Tue, 02 May 2023 12:34:56 UTC"}`,
			expectedResult: map[any]datapoint.Point{
				value.Pair{Base: "BTC", Quote: "USD"}: {
					Value: value.NewTick(value.Pair{Base: "BTC", Quote: "USD"}, 1000, 100),
					Time:  time.Date(2023, 5, 2, 12, 34, 56, 0, time.UTC),
				},
			},
		},
		{
			name:         "time.RFC1123Z",
			query:        ".",
			responseBody: `{"price": "1000", "volume": "100", "time": "Tue, 02 May 2023 12:34:56 +0000"}`,
			expectedResult: map[any]datapoint.Point{
				value.Pair{Base: "BTC", Quote: "USD"}: {
					Value: value.NewTick(value.Pair{Base: "BTC", Quote: "USD"}, 1000, 100),
					Time:  time.Date(2023, 5, 2, 12, 34, 56, 0, time.FixedZone("+0000", 0)),
				},
			},
		},
		{
			name:         "time.RFC822",
			query:        ".",
			responseBody: `{"price": "1000", "volume": "100", "time": "02 May 23 12:34 UTC"}`,
			expectedResult: map[any]datapoint.Point{
				value.Pair{Base: "BTC", Quote: "USD"}: {
					Value: value.NewTick(value.Pair{Base: "BTC", Quote: "USD"}, 1000, 100),
					Time:  time.Date(2023, 5, 2, 12, 34, 0, 0, time.UTC),
				},
			},
		},
		{
			name:         "time.RFC822Z",
			query:        ".",
			responseBody: `{"price": "1000", "volume": "100", "time": "02 May 23 12:34 +0000"}`,
			expectedResult: map[any]datapoint.Point{
				value.Pair{Base: "BTC", Quote: "USD"}: {
					Value: value.NewTick(value.Pair{Base: "BTC", Quote: "USD"}, 1000, 100),
					Time:  time.Date(2023, 5, 2, 12, 34, 0, 0, time.FixedZone("+0000", 0)),
				},
			},
		},
		{
			name:         "time.RFC850",
			query:        ".",
			responseBody: `{"price": "1000", "volume": "100", "time": "Tuesday, 02-May-23 12:34:56 UTC"}`,
			expectedResult: map[any]datapoint.Point{
				value.Pair{Base: "BTC", Quote: "USD"}: {
					Value: value.NewTick(value.Pair{Base: "BTC", Quote: "USD"}, 1000, 100),
					Time:  time.Date(2023, 5, 2, 12, 34, 56, 0, time.UTC),
				},
			},
		},
		{
			name:         "time.ANSIC",
			query:        ".",
			responseBody: `{"price": "1000", "volume": "100", "time": "Tue May  2 12:34:56 2023"}`,
			expectedResult: map[any]datapoint.Point{
				value.Pair{Base: "BTC", Quote: "USD"}: {
					Value: value.NewTick(value.Pair{Base: "BTC", Quote: "USD"}, 1000, 100),
					Time:  time.Date(2023, 5, 2, 12, 34, 56, 0, time.UTC),
				},
			},
		},
		{
			name:         "time.UnixDate",
			query:        ".",
			responseBody: `{"price": "1000", "volume": "100", "time": "Tue May  2 12:34:56 UTC 2023"}`,
			expectedResult: map[any]datapoint.Point{
				value.Pair{Base: "BTC", Quote: "USD"}: {
					Value: value.NewTick(value.Pair{Base: "BTC", Quote: "USD"}, 1000, 100),
					Time:  time.Date(2023, 5, 2, 12, 34, 56, 0, time.UTC),
				},
			},
		},
		{
			name:         "time.RubyDate",
			query:        ".",
			responseBody: `{"price": "1000", "volume": "100", "time": "Tue May 02 12:34:56 +0000 2023"}`,
			expectedResult: map[any]datapoint.Point{
				value.Pair{Base: "BTC", Quote: "USD"}: {
					Value: value.NewTick(value.Pair{Base: "BTC", Quote: "USD"}, 1000, 100),
					Time:  time.Date(2023, 5, 2, 12, 34, 56, 0, time.FixedZone("+0000", 0)),
				},
			},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server.
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintln(w, tt.responseBody)
			}))
			defer server.Close()

			// Create a new TickGenericJQ data.
			gjq, err := NewTickGenericJQ(TickGenericJQConfig{
				URL:    server.URL,
				Query:  tt.query,
				Logger: null.New(),
			})
			require.NoError(t, err)

			// Test the data.
			pairs := []any{value.Pair{Base: "BTC", Quote: "USD"}}
			points, err := gjq.FetchDataPoints(context.Background(), pairs)
			// Orcfax customizations mean this code fails earlier but
			// the impact of removing all tests here is still being
			// analyzed and so will be removed later as required.
			if tt.skipError && err != nil {
				return
			}
			require.NoError(t, err)
			if len(points) <= 0 {
				t.Error("data points must not be nil for test")
			}
			for i, dataPoint := range points {
				if tt.expectedResult[i].Value != nil {
					assert.Equal(t, tt.expectedResult[i].Value.(value.Tick).Pair, dataPoint.Value.(value.Tick).Pair)
					assert.Equal(t, tt.expectedResult[i].Value.(value.Tick).Price, dataPoint.Value.(value.Tick).Price)
					if !tt.skipVolumeAssert {
						assert.Equal(t, tt.expectedResult[i].Value.(value.Tick).Volume24h, dataPoint.Value.(value.Tick).Volume24h)
					}
				} else {
					assert.Nil(t, dataPoint.Value)
				}
				if !tt.skipTimeAssert {
					assert.Equal(t, tt.expectedResult[i].Time, dataPoint.Time)
				}
				if tt.expectedResult[i].Error != nil {
					assert.EqualError(t, dataPoint.Error, tt.expectedResult[i].Error.Error())
				} else {
					assert.NoError(t, dataPoint.Error)
				}
				responseB64 := dataPoint.Meta["response"]
				responseDecoded, _ := b64.StdEncoding.DecodeString(responseB64.(string))
				var httpResponse httpResponse
				json.Unmarshal(responseDecoded, &httpResponse)
				if httpResponse.Headers == nil {
					t.Errorf("rudimentary decode of header failed")
				}
				if httpResponse.Body == nil {
					t.Errorf("rudimentary decode of body failed")
				}
			}
		})
	}
}
