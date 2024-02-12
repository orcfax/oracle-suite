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
	"fmt"
	"math/big"
	"sort"
	"time"

	"github.com/defiweb/go-eth/abi"
	"github.com/defiweb/go-eth/rpc"
	"github.com/defiweb/go-eth/types"
	"golang.org/x/exp/maps"

	"github.com/orcfax/oracle-suite/pkg/datapoint"
	"github.com/orcfax/oracle-suite/pkg/datapoint/value"
	"github.com/orcfax/oracle-suite/pkg/ethereum"
	"github.com/orcfax/oracle-suite/pkg/log"
	"github.com/orcfax/oracle-suite/pkg/log/null"
)

const CurveLoggerTag = "CURVE_ORIGIN"

type CurveConfig struct {
	Client                      rpc.RPC
	StableSwapContractAddresses ContractAddresses
	CryptoSwapContractAddresses ContractAddresses
	Logger                      log.Logger
	Blocks                      []int64
}

type Curve struct {
	client                       rpc.RPC
	stableSwapContractAddresses  ContractAddresses
	cryptoSwapContract2Addresses ContractAddresses
	erc20                        *ERC20
	blocks                       []int64
	logger                       log.Logger
}

func NewCurve(config CurveConfig) (*Curve, error) {
	if config.Client == nil {
		return nil, fmt.Errorf("ethereum client not set")
	}
	if config.Logger == nil {
		config.Logger = null.New()
	}

	erc20, err := NewERC20(config.Client)
	if err != nil {
		return nil, err
	}

	return &Curve{
		client:                       config.Client,
		stableSwapContractAddresses:  config.StableSwapContractAddresses,
		cryptoSwapContract2Addresses: config.CryptoSwapContractAddresses,
		erc20:                        erc20,
		blocks:                       config.Blocks,
		logger:                       config.Logger.WithField("curve", CurveLoggerTag),
	}, nil
}

//nolint:funlen,gocyclo
func (c *Curve) fetchDataPoints(
	ctx context.Context,
	contractAddresses ContractAddresses,
	pairs []value.Pair,
	secondary bool,
	block *big.Int,
) (
	map[value.Pair]datapoint.Point,
	error,
) {

	points := make(map[value.Pair]datapoint.Point)
	var getDy *abi.Method
	if !secondary {
		getDy = getDy1
	} else {
		getDy = getDy2
	}

	// Get all the token addresses based on their token indexes in the pool
	var callsToken []types.Call
	for _, pair := range pairs {
		contract, baseIndex, quoteIndex, err := contractAddresses.ByPair(pair)
		if err != nil {
			continue
		}
		callData, err := coins.EncodeArgs(baseIndex)
		if err != nil {
			continue
		}
		callsToken = append(callsToken, types.Call{
			To:    &contract,
			Input: callData,
		})
		callData, err = coins.EncodeArgs(quoteIndex)
		if err != nil {
			continue
		}
		callsToken = append(callsToken, types.Call{
			To:    &contract,
			Input: callData,
		})
	}

	var tokenDetails map[string]ERC20Details
	if len(callsToken) > 0 {
		resp, err := ethereum.MultiCall(ctx, c.client, callsToken, types.LatestBlockNumber)
		if err != nil {
			return nil, err
		}

		tokensMap := make(map[types.Address]struct{})
		for i := range resp {
			var address types.Address
			if err := coins.DecodeValues(resp[i], &address); err != nil {
				return nil, fmt.Errorf("failed decoding tokens in the pool: %w", err)
			}
			tokensMap[address] = struct{}{}
		}
		tokenDetails, err = c.erc20.GetSymbolAndDecimals(ctx, maps.Keys(tokensMap))
		if err != nil {
			return nil, fmt.Errorf("failed getting symbol & decimals for tokens of pool: %w", err)
		}
	}

	totals := make([]*big.Float, len(pairs))
	var calls []types.Call
	n := 0
	for _, pair := range pairs {
		if _, ok := tokenDetails[pair.Base]; !ok {
			points[pair] = datapoint.Point{Error: fmt.Errorf("not found base token: %s", pair.Base)}
			continue
		}
		if _, ok := tokenDetails[pair.Quote]; !ok {
			points[pair] = datapoint.Point{Error: fmt.Errorf("not found quote token: %s", pair.Quote)}
			continue
		}
		baseToken := tokenDetails[pair.Base]
		quoteToken := tokenDetails[pair.Quote]

		pool, baseIndex, quoteIndex, err := contractAddresses.ByPair(pair)
		if err != nil {
			points[pair] = datapoint.Point{Error: err}
			continue
		}

		// `get_dy` function requires to pass the token index in first two parameters in ascending order
		// and the third parameter is the token amount scaled up by first token's decimals
		// The return value is the token amount scaled up by second token's decimals
		var callData types.Bytes
		if baseIndex < quoteIndex {
			callData, err = getDy.EncodeArgs(
				baseIndex,
				quoteIndex,
				new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(baseToken.decimals)), nil),
			)
		} else { // inverted pair
			callData, err = getDy.EncodeArgs(
				quoteIndex,
				baseIndex,
				new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(quoteToken.decimals)), nil),
			)
		}

		if err != nil {
			points[pair] = datapoint.Point{Error: fmt.Errorf(
				"failed to get contract args for pair: %s: %w",
				pair.String(),
				err,
			)}
			continue
		}
		calls = append(calls, types.Call{
			To:    &pool,
			Input: callData,
		})
		totals[n] = new(big.Float).SetInt64(0)
		n++
	}

	if len(calls) > 0 {
		for _, blockDelta := range c.blocks {
			resp, err := ethereum.MultiCall(ctx, c.client, calls, types.BlockNumberFromUint64(uint64(block.Int64()-blockDelta)))
			if err != nil {
				return nil, err
			}

			n = 0
			for _, pair := range pairs {
				if points[pair].Error != nil {
					continue
				}
				_, baseIndex, quoteIndex, _ := contractAddresses.ByPair(pair)
				baseToken := tokenDetails[pair.Base]
				quoteToken := tokenDetails[pair.Quote]

				price := new(big.Float).SetInt(new(big.Int).SetBytes(resp[n][0:32]))
				// price = price / 10 ^ quoteDecimals
				if baseIndex < quoteIndex {
					// The return value of `get_dy` is the number scaled up by quote token
					price = new(big.Float).Quo(
						price,
						new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(quoteToken.decimals)), nil)),
					)
				} else { // inverted pair
					// The return value of `get_dy` is the number scaled up by base token
					price = new(big.Float).Quo(
						price,
						new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(baseToken.decimals)), nil)),
					)
				}
				totals[n] = totals[n].Add(totals[n], price)
				n++
			}
		}
	}

	n = 0
	for _, pair := range pairs {
		if points[pair].Error != nil {
			continue
		}
		avgPrice := new(big.Float).Quo(totals[n], new(big.Float).SetUint64(uint64(len(c.blocks))))
		n++

		// Invert the price if inverted price
		_, baseIndex, quoteIndex, _ := contractAddresses.ByPair(pair)
		if baseIndex > quoteIndex {
			avgPrice = new(big.Float).Quo(new(big.Float).SetUint64(1), avgPrice)
		}

		tick := value.NewTick(pair, avgPrice, nil)
		points[pair] = datapoint.Point{
			Value: tick,
			Time:  time.Now(),
		}
	}
	return points, nil
}

//nolint:gocyclo
func (c *Curve) FetchDataPoints(ctx context.Context, query []any) (map[any]datapoint.Point, error) {
	pairs, ok := queryToPairs(query)
	if !ok {
		return nil, fmt.Errorf("invalid query type: %T, expected []Pair", query)
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].String() < pairs[j].String()
	})

	block, err := c.client.BlockNumber(ctx)

	if err != nil {
		return nil, fmt.Errorf("cannot get block number, %w", err)
	}

	points := make(map[any]datapoint.Point)

	// Filter pairs into two group according to their data types in `get_dy`; `int128`, `uint256`
	pairs1 := make(map[value.Pair]struct{})
	pairs2 := make(map[value.Pair]struct{})
	for _, pair := range pairs {
		_, baseIndex, quoteIndex, err := c.stableSwapContractAddresses.ByPair(pair)
		if err == nil && baseIndex >= 0 && quoteIndex >= 0 {
			pairs1[pair] = struct{}{}
			continue
		}
		_, baseIndex, quoteIndex, err = c.cryptoSwapContract2Addresses.ByPair(pair)
		if err == nil && baseIndex >= 0 && quoteIndex >= 0 {
			pairs2[pair] = struct{}{}
			continue
		}
		if err != nil {
			points[pair] = datapoint.Point{Error: err}
			continue
		}
	}

	points1, err1 := c.fetchDataPoints(ctx, c.stableSwapContractAddresses, maps.Keys(pairs1), false, block)
	points2, err2 := c.fetchDataPoints(ctx, c.cryptoSwapContract2Addresses, maps.Keys(pairs2), true, block)
	if err1 != nil {
		return points, err1
	}
	if err2 != nil {
		return points, err2
	}
	if points1 == nil && points2 == nil {
		return points, fmt.Errorf("failed to fetch data points")
	}

	for pair, point := range points1 {
		points[pair] = point
	}
	for pair, point := range points2 {
		points[pair] = point
	}
	return points, nil
}
