//  Copyright (C) 2021-2023 Chronicle Labs, Inc. 2023 Orcfax Ltd.
//
//  This program is free software: you can redistribute it and/or modify
//  it under the terms of the GNU Affero General Public License as
//  published by the Free Software Foundation, either version 3 of the
//  License, or (at your option) any later version.
//
//  This program is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of
//  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//  GNU Affero General Public License for more details.
//
//  You should have received a copy of the GNU Affero General Public License
//  along with this program.  If not, see <http://www.gnu.org/licenses/>.

package origin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	lg "log"
	"net/http"
	"strings"
	"time"

	"github.com/orcfax/oracle-suite/pkg/datapoint"
	"github.com/orcfax/oracle-suite/pkg/datapoint/value"
	"github.com/orcfax/oracle-suite/pkg/log"
	"github.com/orcfax/oracle-suite/pkg/log/null"
	"github.com/orcfax/oracle-suite/pkg/util/interpolate"
)

const TickGenericHTTPLoggerTag = "TICK_GENERIC_HTTP_ORIGIN"

type HTTPCallback func(ctx context.Context, pairs []value.Pair, data io.Reader) (map[any]datapoint.Point, error)

type TickGenericHTTPConfig struct {
	// URL is an TickGenericHTTP endpoint that returns JSON data. It may contain
	// the following variables:
	//   - ${lcbase} - lower case base asset
	//   - ${ucbase} - upper case base asset
	//   - ${lcquote} - lower case quote asset
	//   - ${ucquote} - upper case quote asset
	//   - ${lcbases} - lower case base assets joined by commas
	//   - ${ucbases} - upper case base assets joined by commas
	//   - ${lcquotes} - lower case quote assets joined by commas
	//   - ${ucquotes} - upper case quote assets joined by commas
	URL string

	// Headers is a set of TickGenericHTTP headers that are sent with each request.
	Headers http.Header

	// Callback is a function that is used to parse the response body.
	Callback HTTPCallback

	// Client is an TickGenericHTTP client that is used to fetch data from the
	// TickGenericHTTP endpoint. If nil, http.DefaultClient is used.
	Client *http.Client

	// Logger is an TickGenericHTTP logger that is used to log errors. If nil,
	// null logger is used.
	Logger log.Logger
}

// TickGenericHTTP is a generic http price provider that can fetch prices from
// an HTTP endpoint. The callback function is used to parse the response body.
type TickGenericHTTP struct {
	url      string
	client   *http.Client
	headers  http.Header
	callback HTTPCallback
	logger   log.Logger
}

// NewTickGenericHTTP creates a new TickGenericHTTP instance.
func NewTickGenericHTTP(config TickGenericHTTPConfig) (*TickGenericHTTP, error) {
	if config.URL == "" {
		return nil, fmt.Errorf("url cannot be empty")
	}
	if config.Callback == nil {
		return nil, fmt.Errorf("callback cannot be nil")
	}
	if config.Client == nil {
		config.Client = &http.Client{}
	}
	if config.Logger == nil {
		config.Logger = null.New()
	}
	return &TickGenericHTTP{
		url:      config.URL,
		client:   config.Client,
		headers:  config.Headers,
		callback: config.Callback,
		logger:   config.Logger.WithField("tag", TickGenericHTTPLoggerTag),
	}, nil
}

// FetchDataPoints implements the Origin interface.
func (g *TickGenericHTTP) FetchDataPoints(ctx context.Context, query []any) (map[any]datapoint.Point, error) {
	pairs, ok := queryToPairs(query)
	if !ok {
		return nil, fmt.Errorf("invalid query type: %T, expected []Pair", query)
	}
	points := make(map[any]datapoint.Point)
	for url, pairs := range g.group(pairs) {
		g.logger.
			WithFields(log.Fields{
				"url":   url,
				"pairs": pairs,
			}).
			Debug("HTTP request")
		// Perform TickGenericHTTP request.
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			fillDataPointsWithError(points, pairs, err)
			continue
		}
		req.Header = g.headers
		req = req.WithContext(ctx)
		// Set a request timeout for the http client.
		g.client.Timeout = time.Duration(10 * time.Second)
		// Execute TickGenericHTTP request.
		res, err := g.client.Do(req)
		if err != nil {
			fillDataPointsWithError(points, pairs, err)
			continue
		}
		defer res.Body.Close()
		// Create a secondary buffer to allow it to be read multiple
		// times across this function.
		responseBuffer, _ := io.ReadAll(res.Body)
		readerOne := io.NopCloser(bytes.NewBuffer(responseBuffer))
		readerTwo := io.NopCloser(bytes.NewBuffer(responseBuffer))
		// Assign one copy back to the original CloseReader object.
		res.Body = readerOne
		rawBuffer := new(strings.Builder)
		_, _ = io.Copy(rawBuffer, readerTwo)
		var responseBody map[string]interface{}
		// The shape of the response may be a dictionary. While we have
		// access t a string via the rawBuffer variable we want this
		// data to remain easily machine readable as with the headers.
		if err := json.Unmarshal([]byte(rawBuffer.String()), &responseBody); err != nil {
			var bodyArr []interface{}
			// The shape of the response may be an array.
			if err := json.Unmarshal([]byte(rawBuffer.String()), &bodyArr); err != nil {
				// The shape of the response is not understood or cannot
				// be parsed, e.g. it is invalid JSON in some way.
				return points, fmt.Errorf("error processing the response body")
			}
			// We create a data key to associate the array with and
			// attach it here.
			responseBody = make(map[string]interface{})
			responseBody["data"] = bodyArr
		}
		// Create a collector message object consisting of para- and
		// meta-data about the collection activity.
		collectorMessage := make(map[string]interface{})
		collectorMessage["headers"] = res.Header
		collectorMessage["body"] = responseBody
		collectorJSON, err := json.MarshalIndent(collectorMessage, "", "  ")
		if err != nil {
			return points, fmt.Errorf("error processing response json")
		}
		resPoints, err := g.callback(ctx, pairs, res.Body)
		if err != nil {
			fillDataPointsWithError(points, pairs, err)
			continue
		}
		// Run callback function.
		for pair, point := range resPoints {
			// NB: Meta may be initialized later with the following
			// data:
			//
			// map[
			//   expiry_threshold:5m0s
			//   freshness_threshold:1m0s
			//   origin:bitstamp
			//   query:ADA/EUR
			//   type:origin
			// ]
			//
			point.Meta = make(map[string]any)
			// Add Orcfax metadata.
			point.Meta["response"] = []byte(collectorJSON)
			point.Meta["request_url"] = url
			point.Meta["collector"] = "tick_generic_jq"
			points[pair] = point
		}
	}
	return points, nil
}

// group interpolates the URL by substituting the base and quote, and then
// groups the resulting pairs by the interpolated URL.
func (g *TickGenericHTTP) group(pairs []value.Pair) map[string][]value.Pair {
	pairMap := make(map[string][]value.Pair)
	parsedURL := interpolate.Parse(g.url)
	bases := make([]string, 0, len(pairs))
	quotes := make([]string, 0, len(pairs))
	for _, pair := range pairs {
		bases = append(bases, pair.Base)
		quotes = append(quotes, pair.Quote)
	}
	for _, pair := range pairs {
		url := parsedURL.Interpolate(func(variable interpolate.Variable) string {
			switch variable.Name {
			case "lcbase":
				return strings.ToLower(pair.Base)
			case "ucbase":
				return strings.ToUpper(pair.Base)
			case "lcquote":
				return strings.ToLower(pair.Quote)
			case "ucquote":
				return strings.ToUpper(pair.Quote)
			case "lcbases":
				return strings.ToLower(strings.Join(bases, ","))
			case "ucbases":
				return strings.ToUpper(strings.Join(bases, ","))
			case "lcquotes":
				return strings.ToLower(strings.Join(quotes, ","))
			case "ucquotes":
				return strings.ToUpper(strings.Join(quotes, ","))
			default:
				return variable.Default
			}
		})
		lg.Printf("compiled URL for %s: %s", pair, url)
		pairMap[url] = append(pairMap[url], pair)
	}
	return pairMap
}
