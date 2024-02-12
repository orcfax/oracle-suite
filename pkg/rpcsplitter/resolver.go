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

package rpcsplitter

import (
	"errors"
	"math/big"
	"sort"

	"github.com/orcfax/oracle-suite/pkg/rpcsplitter/types"
)

var errNotEnoughResponses = errors.New("not enough responses from RPC servers")
var errDifferentResponses = errors.New("RPC servers returned different responses")

// resolver takes responses from different endpoints and returns a single
// response.
type resolver interface {
	resolve([]any) (any, error)
}

// defaultResolver compares responses with each other and returns the most
// common one. If there are multiple responses with the same number of
// occurrences but greater than minResponses, an error is returned.
type defaultResolver struct {
	minResponses int // specifies minimum number of occurrences of the most common response
}

// resolve implements resolver interface.
func (r *defaultResolver) resolve(resps []any) (any, error) {
	resps, errs := extractErrors(resps)
	if len(resps) < r.minResponses {
		return nil, addError(errNotEnoughResponses, errs...)
	}
	if len(resps) == 1 {
		return resps[0], nil
	}
	mostCommonResp := resps[0]
	mostCommonCounter := 0
	multiple := false
	for _, a := range resps {
		counter := 0
		for _, b := range resps {
			if compare(a, b) {
				counter++
			}
		}
		if counter > mostCommonCounter {
			multiple = false
			mostCommonResp = a
			mostCommonCounter = counter
		}
		if counter == mostCommonCounter && !compare(mostCommonResp, a) {
			multiple = true
		}
	}
	if multiple || mostCommonCounter < r.minResponses {
		return nil, addError(errDifferentResponses, errs...)
	}
	return mostCommonResp, nil
}

// gasValueResolver is designed to handle responses from methods returning a
// gas value. The way how the response is calculated depends on the number of
// responses:
// * one response: returns value as is
// * two responses: returns the lowest one
// * three or more responses: returns the median value
type gasValueResolver struct {
	minResponses int // specifies minimum number of valid responses
}

// resolve implements resolver interface.
func (r *gasValueResolver) resolve(resps []any) (any, error) {
	resps, errs := extractErrors(resps)
	ns := filterByNumberType(resps)
	if len(ns) < r.minResponses {
		return nil, addError(errNotEnoughResponses, errs...)
	}
	if len(ns) == 1 {
		return resps[0], nil
	}
	if len(ns) == 2 {
		// With two correct answers, it is safer to return the lower value.
		// Otherwise, the compromised endpoint may return a very high gas
		// price. If this price is used to determine transaction fees, it
		// could cause clients to lose money on transaction fees.
		a := ns[0].Big()
		b := ns[1].Big()
		if a.Cmp(b) > 0 {
			return bigToNumberPtr(b), nil
		}
		return bigToNumberPtr(a), nil
	}
	// Calculate the median.
	sort.Slice(ns, func(i, j int) bool {
		return ns[i].Big().Cmp(ns[j].Big()) < 0
	})
	if len(ns)%2 == 0 {
		m := len(ns) / 2
		bx := ns[m-1].Big()
		by := ns[m].Big()
		return bigToNumberPtr(new(big.Int).Div(new(big.Int).Add(bx, by), big.NewInt(2))), nil
	}
	return ns[len(ns)/2], nil
}

// blockNumberResolver is designed to handle responses from eth_blockNumber method.
//
// Because some RPC endpoints may be behind others, the blockNumberResolver
// uses the lowest block number of all responses, but the difference from the
// last known cannot be less than specified in the maxBlocksBehind parameter.
type blockNumberResolver struct {
	minResponses    int // specifies minimum number of valid responses
	maxBlocksBehind int // specifies how far behind the last known block the returned block can be
}

// resolve implements resolver interface.
func (r *blockNumberResolver) resolve(resps []any) (any, error) {
	resps, errs := extractErrors(resps)
	ns := filterByNumberType(resps)
	if len(ns) < r.minResponses {
		return nil, addError(errNotEnoughResponses, errs...)
	}
	if len(ns) == 1 {
		return ns[0], nil
	}
	// Find the highest block number in the given responses:
	high := ns[0].Big()
	for _, n := range ns {
		nb := n.Big()
		if high.Cmp(nb) < 0 {
			high = nb
		}
	}
	// Find the lowest block number that is higher or equal to high-maxBlocksBehind:
	block := high
	for _, n := range ns {
		nb := n.Big()
		if new(big.Int).Sub(high, nb).Cmp(big.NewInt(int64(r.maxBlocksBehind))) <= 0 && nb.Cmp(block) < 0 {
			block = nb
		}
	}
	return bigToNumberPtr(block), nil
}

func extractErrors(resps []any) (filtered []any, errs []error) {
	for _, r := range resps {
		if e, ok := r.(error); ok {
			errs = append(errs, e)
		} else {
			filtered = append(filtered, r)
		}
	}
	return
}

func filterByNumberType(resps []any) (s []*types.Number) {
	for _, r := range resps {
		if t, ok := r.(*types.Number); ok {
			s = append(s, t)
		}
	}
	return
}

func bigToNumberPtr(x *big.Int) *types.Number {
	n := types.BigToNumber(x)
	return &n
}
