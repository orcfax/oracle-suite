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

package relay

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/defiweb/go-eth/rpc"
	"github.com/defiweb/go-eth/types"
	"github.com/stretchr/testify/assert"

	"github.com/orcfax/oracle-suite/pkg/contract"
	"github.com/orcfax/oracle-suite/pkg/contract/chronicle"
	"github.com/orcfax/oracle-suite/pkg/contract/mock"
	"github.com/orcfax/oracle-suite/pkg/transport/messages"
	"github.com/orcfax/oracle-suite/pkg/util/bn"
)

func TestScribe(t *testing.T) {
	testFeed := types.MustAddressFromHex("0x1111111111111111111111111111111111111111")
	testFeed2 := types.MustAddressFromHex("0x2222222222222222222222222222222222222222")
	mockLogger := newMockLogger(t)
	mockContract := newMockScribeContract(t)
	mockMuSigStore := newMockSignatureProvider(t)

	scribe := &scribe{
		log:        mockLogger,
		muSigStore: mockMuSigStore,
		contract:   mockContract,
		dataModel:  "ETH/USD",
		spread:     0.05,
		expiration: 10 * time.Minute,
	}

	t.Run("above spread", func(t *testing.T) {
		scribe.cachedState = scribeState{}
		mockLogger.reset(t)
		mockContract.reset(t)
		mockMuSigStore.reset(t)

		ctx := context.Background()
		musigTime := time.Now()
		musigCommitment := types.MustAddressFromHex("0x1234567890123456789012345678901234567890")
		musigSignature := big.NewInt(1234567890)
		mockLogger.InfoFn = func(args ...any) {}
		mockLogger.DebugFn = func(args ...any) {}
		mockContract.ClientFn = func() rpc.RPC { return nil }
		mockContract.AddressFn = func() types.Address { return types.Address{} }
		mockContract.WatFn = func() contract.TypedSelfCaller[string] {
			return mock.NewTypedCaller[string](t).MockResult("ETH/USD", nil)
		}
		mockContract.BarFn = func() contract.TypedSelfCaller[int] {
			return mock.NewTypedCaller[int](t).MockResult(1, nil)
		}
		mockContract.FeedsFn = func() contract.TypedSelfCaller[[]types.Address] {
			return mock.NewTypedCaller[[]types.Address](t).MockResult(
				[]types.Address{testFeed},
				nil,
			)
		}
		mockContract.ReadFn = func(ctx context.Context) (chronicle.PokeData, error) {
			return chronicle.PokeData{
				Val: bn.DecFixedPoint(100, chronicle.ScribePricePrecision),
				Age: time.Now().Add(-1 * time.Minute),
			}, nil
		}
		mockMuSigStore.SignaturesByDataModelFn = func(model string) []*messages.MuSigSignature {
			assert.Equal(t, "ETH/USD", model)
			return []*messages.MuSigSignature{
				{
					MuSigMessage: &messages.MuSigMessage{
						Signers: []types.Address{testFeed},
						MsgMeta: messages.MuSigMeta{Meta: messages.MuSigMetaTickV1{
							Wat: "ETH/USD",
							Val: bn.DecFixedPoint(110, chronicle.ScribePricePrecision),
							Age: musigTime,
						}},
					},
					Commitment:       musigCommitment,
					SchnorrSignature: musigSignature,
				},
			}
		}

		pokeCalled := false
		mockContract.PokeFn = func(pokeData chronicle.PokeData, schnorrData chronicle.SchnorrData) contract.SelfTransactableCaller {
			pokeCalled = true
			assert.Equal(t, bn.DecFixedPoint(110, chronicle.ScribePricePrecision), pokeData.Val)
			assert.Equal(t, musigTime, pokeData.Age)
			assert.Equal(t, musigCommitment, schnorrData.Commitment)
			assert.Equal(t, musigSignature, schnorrData.Signature)
			return mock.NewCaller(t).MockAllowAllCalls()
		}

		scribe.createRelayCall(ctx)
		assert.True(t, pokeCalled)
	})

	t.Run("within spread", func(t *testing.T) {
		scribe.cachedState = scribeState{}
		mockLogger.reset(t)
		mockContract.reset(t)
		mockMuSigStore.reset(t)

		ctx := context.Background()
		musigTime := time.Now()
		musigCommitment := types.MustAddressFromHex("0x1234567890123456789012345678901234567890")
		musigSignature := big.NewInt(1234567890)
		mockLogger.InfoFn = func(args ...any) {}
		mockLogger.DebugFn = func(args ...any) {}
		mockContract.ClientFn = func() rpc.RPC { return nil }
		mockContract.AddressFn = func() types.Address { return types.Address{} }
		mockContract.WatFn = func() contract.TypedSelfCaller[string] {
			return mock.NewTypedCaller[string](t).MockResult("ETH/USD", nil)
		}
		mockContract.BarFn = func() contract.TypedSelfCaller[int] {
			return mock.NewTypedCaller[int](t).MockResult(1, nil)
		}
		mockContract.FeedsFn = func() contract.TypedSelfCaller[[]types.Address] {
			return mock.NewTypedCaller[[]types.Address](t).MockResult(
				[]types.Address{testFeed},
				nil,
			)
		}
		mockContract.ReadFn = func(ctx context.Context) (chronicle.PokeData, error) {
			return chronicle.PokeData{
				Val: bn.DecFixedPoint(100, chronicle.ScribePricePrecision),
				Age: time.Now().Add(-1 * time.Minute),
			}, nil
		}
		mockMuSigStore.SignaturesByDataModelFn = func(model string) []*messages.MuSigSignature {
			assert.Equal(t, "ETH/USD", model)
			return []*messages.MuSigSignature{
				{
					MuSigMessage: &messages.MuSigMessage{
						Signers: []types.Address{testFeed},
						MsgMeta: messages.MuSigMeta{Meta: messages.MuSigMetaTickV1{
							Wat: "ETH/USD",
							Val: bn.DecFixedPoint(100, chronicle.ScribePricePrecision),
							Age: musigTime,
						}},
					},
					Commitment:       musigCommitment,
					SchnorrSignature: musigSignature,
				},
			}
		}

		scribe.createRelayCall(ctx)
	})

	t.Run("expired", func(t *testing.T) {
		scribe.cachedState = scribeState{}
		mockLogger.reset(t)
		mockContract.reset(t)
		mockMuSigStore.reset(t)

		ctx := context.Background()
		musigTime := time.Now()
		musigCommitment := types.MustAddressFromHex("0x1234567890123456789012345678901234567890")
		musigSignature := big.NewInt(1234567890)
		mockLogger.InfoFn = func(args ...any) {}
		mockLogger.DebugFn = func(args ...any) {}
		mockContract.ClientFn = func() rpc.RPC { return nil }
		mockContract.AddressFn = func() types.Address { return types.Address{} }
		mockContract.WatFn = func() contract.TypedSelfCaller[string] {
			return mock.NewTypedCaller[string](t).MockResult("ETH/USD", nil)
		}
		mockContract.BarFn = func() contract.TypedSelfCaller[int] {
			return mock.NewTypedCaller[int](t).MockResult(1, nil)
		}
		mockContract.FeedsFn = func() contract.TypedSelfCaller[[]types.Address] {
			return mock.NewTypedCaller[[]types.Address](t).MockResult(
				[]types.Address{testFeed},
				nil,
			)
		}
		mockContract.ReadFn = func(ctx context.Context) (chronicle.PokeData, error) {
			return chronicle.PokeData{
				Val: bn.DecFixedPoint(100, chronicle.ScribePricePrecision),
				Age: time.Now().Add(-15 * time.Minute),
			}, nil
		}
		mockMuSigStore.SignaturesByDataModelFn = func(model string) []*messages.MuSigSignature {
			assert.Equal(t, "ETH/USD", model)
			return []*messages.MuSigSignature{
				{
					MuSigMessage: &messages.MuSigMessage{
						Signers: []types.Address{testFeed},
						MsgMeta: messages.MuSigMeta{Meta: messages.MuSigMetaTickV1{
							Wat: "ETH/USD",
							Val: bn.DecFixedPoint(100, chronicle.ScribePricePrecision),
							Age: musigTime,
						}},
					},
					Commitment:       musigCommitment,
					SchnorrSignature: musigSignature,
				},
			}
		}

		pokeCalled := false
		mockContract.PokeFn = func(pokeData chronicle.PokeData, schnorrData chronicle.SchnorrData) contract.SelfTransactableCaller {
			pokeCalled = true
			assert.Equal(t, bn.DecFixedPoint(100, chronicle.ScribePricePrecision), pokeData.Val)
			assert.Equal(t, musigTime, pokeData.Age)
			assert.Equal(t, musigCommitment, schnorrData.Commitment)
			assert.Equal(t, musigSignature, schnorrData.Signature)
			return mock.NewCaller(t).MockAllowAllCalls()
		}

		scribe.createRelayCall(ctx)
		assert.True(t, pokeCalled)
	})

	t.Run("old signature", func(t *testing.T) {
		scribe.cachedState = scribeState{}
		mockLogger.reset(t)
		mockContract.reset(t)
		mockMuSigStore.reset(t)

		ctx := context.Background()
		musigTime := time.Now().Add(-15 * time.Minute)
		musigCommitment := types.MustAddressFromHex("0x1234567890123456789012345678901234567890")
		musigSignature := big.NewInt(1234567890)
		mockLogger.InfoFn = func(args ...any) {}
		mockLogger.DebugFn = func(args ...any) {}
		mockContract.ClientFn = func() rpc.RPC { return nil }
		mockContract.AddressFn = func() types.Address { return types.Address{} }
		mockContract.WatFn = func() contract.TypedSelfCaller[string] {
			return mock.NewTypedCaller[string](t).MockResult("ETH/USD", nil)
		}
		mockContract.BarFn = func() contract.TypedSelfCaller[int] {
			return mock.NewTypedCaller[int](t).MockResult(1, nil)
		}
		mockContract.FeedsFn = func() contract.TypedSelfCaller[[]types.Address] {
			return mock.NewTypedCaller[[]types.Address](t).MockResult(
				[]types.Address{testFeed},
				nil,
			)
		}
		mockContract.ReadFn = func(ctx context.Context) (chronicle.PokeData, error) {
			return chronicle.PokeData{
				Val: bn.DecFixedPoint(100, chronicle.ScribePricePrecision),
				Age: time.Now().Add(-1 * time.Minute),
			}, nil
		}
		mockMuSigStore.SignaturesByDataModelFn = func(model string) []*messages.MuSigSignature {
			assert.Equal(t, "ETH/USD", model)
			return []*messages.MuSigSignature{
				{
					MuSigMessage: &messages.MuSigMessage{
						Signers: []types.Address{testFeed},
						MsgMeta: messages.MuSigMeta{Meta: messages.MuSigMetaTickV1{
							Wat: "ETH/USD",
							Val: bn.DecFixedPoint(110, chronicle.ScribePricePrecision),
							Age: musigTime,
						}},
					},
					Commitment:       musigCommitment,
					SchnorrSignature: musigSignature,
				},
			}
		}

		scribe.createRelayCall(ctx)
	})

	t.Run("broken message", func(t *testing.T) {
		invalidMessages := []*messages.MuSigSignature{
			{
				MuSigMessage:     nil,
				Commitment:       types.ZeroAddress,
				SchnorrSignature: nil,
			},
			{
				MuSigMessage: &messages.MuSigMessage{
					MsgMeta: messages.MuSigMeta{Meta: nil},
				},
				Commitment:       types.ZeroAddress,
				SchnorrSignature: nil,
			},
			{
				MuSigMessage: &messages.MuSigMessage{
					MsgMeta: messages.MuSigMeta{Meta: nil},
				},
				Commitment:       types.ZeroAddress,
				SchnorrSignature: big.NewInt(1234567890),
			},
			{
				MuSigMessage: &messages.MuSigMessage{
					MsgMeta: messages.MuSigMeta{Meta: messages.MuSigMetaTickV1{
						Wat: "ETH/USD",
						Val: nil,
						Age: time.Now(),
					}},
				},
				Commitment:       types.ZeroAddress,
				SchnorrSignature: nil,
			},
			{
				MuSigMessage: &messages.MuSigMessage{
					MsgMeta: messages.MuSigMeta{Meta: messages.MuSigMetaTickV1{
						Wat: "ETH/USD",
						Val: bn.DecFixedPoint(110, chronicle.ScribePricePrecision),
						Age: time.Now(),
					}},
				},
				Commitment:       types.ZeroAddress,
				SchnorrSignature: nil,
			},
			{
				MuSigMessage: &messages.MuSigMessage{
					MsgMeta: messages.MuSigMeta{Meta: messages.MuSigMetaTickV1{
						Wat: "ETH/USD",
						Val: nil,
						Age: time.Now(),
					}},
				},
				Commitment:       types.ZeroAddress,
				SchnorrSignature: big.NewInt(1234567890),
			},
		}

		for i, m := range invalidMessages {
			t.Run(fmt.Sprintf("msg-%d", i+1), func(t *testing.T) {
				scribe.cachedState = scribeState{}
				mockLogger.reset(t)
				mockContract.reset(t)
				mockMuSigStore.reset(t)

				ctx := context.Background()
				mockLogger.InfoFn = func(args ...any) {}
				mockLogger.WarnFn = func(args ...any) {}
				mockLogger.DebugFn = func(args ...any) {}
				mockContract.ClientFn = func() rpc.RPC { return nil }
				mockContract.AddressFn = func() types.Address { return types.Address{} }
				mockContract.WatFn = func() contract.TypedSelfCaller[string] {
					return mock.NewTypedCaller[string](t).MockResult("ETH/USD", nil)
				}
				mockContract.BarFn = func() contract.TypedSelfCaller[int] {
					return mock.NewTypedCaller[int](t).MockResult(1, nil)
				}
				mockContract.FeedsFn = func() contract.TypedSelfCaller[[]types.Address] {
					return mock.NewTypedCaller[[]types.Address](t).MockResult(
						[]types.Address{testFeed},
						nil,
					)
				}
				mockContract.ReadFn = func(ctx context.Context) (chronicle.PokeData, error) {
					return chronicle.PokeData{
						Val: bn.DecFixedPoint(100, chronicle.ScribePricePrecision),
						Age: time.Now().Add(-15 * time.Minute),
					}, nil
				}
				mockMuSigStore.SignaturesByDataModelFn = func(model string) []*messages.MuSigSignature {
					assert.Equal(t, "ETH/USD", model)
					return []*messages.MuSigSignature{m}
				}

				scribe.createRelayCall(ctx)
			})
		}
	})

	t.Run("wrong singers count", func(t *testing.T) {
		scribe.cachedState = scribeState{}
		mockLogger.reset(t)
		mockContract.reset(t)
		mockMuSigStore.reset(t)

		ctx := context.Background()
		musigTime := time.Now()
		musigCommitment := types.MustAddressFromHex("0x1234567890123456789012345678901234567890")
		musigSignature := big.NewInt(1234567890)
		mockLogger.InfoFn = func(args ...any) {}
		mockLogger.DebugFn = func(args ...any) {}
		mockLogger.WarnFn = func(args ...any) {}
		mockContract.ClientFn = func() rpc.RPC { return nil }
		mockContract.AddressFn = func() types.Address { return types.Address{} }
		mockContract.WatFn = func() contract.TypedSelfCaller[string] {
			return mock.NewTypedCaller[string](t).MockResult("ETH/USD", nil)
		}
		mockContract.BarFn = func() contract.TypedSelfCaller[int] {
			return mock.NewTypedCaller[int](t).MockResult(2, nil)
		}
		mockContract.FeedsFn = func() contract.TypedSelfCaller[[]types.Address] {
			return mock.NewTypedCaller[[]types.Address](t).MockResult(
				[]types.Address{testFeed, testFeed2},
				nil,
			)
		}
		mockContract.ReadFn = func(ctx context.Context) (chronicle.PokeData, error) {
			return chronicle.PokeData{
				Val: bn.DecFixedPoint(100, chronicle.ScribePricePrecision),
				Age: time.Now().Add(-15 * time.Minute),
			}, nil
		}
		mockMuSigStore.SignaturesByDataModelFn = func(model string) []*messages.MuSigSignature {
			assert.Equal(t, "ETH/USD", model)
			return []*messages.MuSigSignature{
				{
					MuSigMessage: &messages.MuSigMessage{
						Signers: []types.Address{testFeed},
						MsgMeta: messages.MuSigMeta{Meta: messages.MuSigMetaTickV1{
							Wat: "ETH/USD",
							Val: bn.DecFixedPoint(110, chronicle.ScribePricePrecision),
							Age: musigTime,
						}},
					},
					Commitment:       musigCommitment,
					SchnorrSignature: musigSignature,
				},
			}
		}

		scribe.createRelayCall(ctx)
	})

	t.Run("call error", func(t *testing.T) {
		scribe.cachedState = scribeState{}
		mockLogger.reset(t)
		mockContract.reset(t)
		mockMuSigStore.reset(t)

		ctx := context.Background()
		mockContract.ClientFn = func() rpc.RPC { return nil }
		mockContract.AddressFn = func() types.Address { return types.Address{} }
		mockContract.WatFn = func() contract.TypedSelfCaller[string] {
			return mock.NewTypedCaller[string](t).MockResult("ETH/USD", nil)
		}
		mockContract.BarFn = func() contract.TypedSelfCaller[int] {
			return mock.NewTypedCaller[int](t).MockResult(1, nil)
		}
		mockContract.FeedsFn = func() contract.TypedSelfCaller[[]types.Address] {
			return mock.NewTypedCaller[[]types.Address](t).MockResult([]types.Address{}, errors.New("foo"))
		}
		mockContract.ReadFn = func(ctx context.Context) (chronicle.PokeData, error) {
			return chronicle.PokeData{
				Val: bn.DecFixedPoint(100, chronicle.ScribePricePrecision),
				Age: time.Now().Add(-15 * time.Minute),
			}, nil
		}
		errLogCalled := false
		mockLogger.ErrorFn = func(args ...any) {
			errLogCalled = true
		}

		scribe.createRelayCall(ctx)
		assert.True(t, errLogCalled)
	})
}
