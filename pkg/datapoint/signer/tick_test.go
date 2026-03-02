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

package signer

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/orcfax/oracle-suite/pkg/datapoint"
	"github.com/orcfax/oracle-suite/pkg/datapoint/value"
)

func TestTick_Supports(t *testing.T) {
	t.Run("supported data point", func(t *testing.T) {
		s := NewTickSigner(privKey)
		assert.True(t, s.Supports(context.Background(), datapoint.Point{Value: value.Tick{}}))
	})
	t.Run("unsupported data point", func(t *testing.T) {
		s := NewTickSigner(privKey)
		assert.False(t, s.Supports(context.Background(), datapoint.Point{Value: value.StaticValue{}}))
	})
}

func TestTick_Sign(t *testing.T) {
	signer := NewTickSigner(privKey)
	signature, err := signer.Sign(context.Background(), "AAABBB", datapoint.Point{
		Value:     value.NewTick(value.Pair{Base: "AAA", Quote: "BBB"}, 42, 0),
		Time:      time.Unix(1605371361, 0),
		SubPoints: nil,
		Meta:      nil,
		Error:     nil,
	})
	require.NoError(t, err)
	assert.Equal(t, expSignature, *signature)
}
