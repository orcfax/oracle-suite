//  Copyright (C) 2021-2023 Chronicle Labs, Inc.
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
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/orcfax/oracle-suite/pkg/datapoint"
	"github.com/orcfax/oracle-suite/pkg/datapoint/value"
)

// Origin provides dataPoint prices for a given set of pairs from an external
// source.
type Origin interface {
	// FetchDataPoints fetches data points for the given list of queries.
	//
	// A query is an any type that can be used to query the origin for a data
	// point. For example, a query could be a pair of assets.
	//
	// Note that this method does not guarantee that data points will be
	// returned for all pairs nor in the same order as the pairs. The caller
	// must verify returned data.
	FetchDataPoints(ctx context.Context, query []any) (map[any]datapoint.Point, error)
}

const maxTokenCount = 3 // Maximum token count being used as key of the contract

type AssetPair [maxTokenCount]string

func (a AssetPair) String() string {
	var s string
	for i := 0; i < len(a); i++ {
		if i > 0 && len(a[i]) > 0 {
			s += "/"
		}
		s += a[i]
	}
	return s
}

func (a AssetPair) MarshalJSON() ([]byte, error) {
	var s string
	for i := 0; i < len(a); i++ {
		if i > 0 && len(a[i]) > 0 {
			s += "/" // separator
		}
		s += a[i]
	}
	return json.Marshal(s)
}

func (a *AssetPair) UnmarshalText(text []byte) error {
	ss := strings.Split(string(text), "/")
	if len(ss) < 2 {
		return fmt.Errorf("asset pair must have at least two tokens, got %q", string(text))
	}
	pairs := AssetPair{"", "", ""}
	for i := 0; i < len(ss) && i < len(pairs); i++ {
		pairs[i] = strings.ToUpper(ss[i])
	}
	*a = pairs
	return nil
}

func (a AssetPair) IndexOf(token string) int {
	for i, val := range a {
		if val == token {
			return i
		}
	}
	return -1
}

func fillDataPointsWithError(points map[any]datapoint.Point, pairs []value.Pair, err error) map[any]datapoint.Point {
	var target = points
	if target == nil {
		target = make(map[any]datapoint.Point)
	}
	for _, pair := range pairs {
		target[pair] = datapoint.Point{Error: err}
	}
	return target
}

func queryToPairs(query []any) ([]value.Pair, bool) {
	pairs := make([]value.Pair, len(query))
	for i, q := range query {
		switch q := q.(type) {
		case value.Pair:
			pairs[i] = q
		default:
			return nil, false
		}
	}
	return pairs, true
}
