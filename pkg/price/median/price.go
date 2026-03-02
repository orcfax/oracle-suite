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

package median

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"
)

const PriceMultiplier = 1e18

var ErrPriceNotSet = errors.New("unable to sign a price because the price is not set")
var ErrUnmarshallingFailure = errors.New("unable to unmarshal given JSON")

func errUnmarshalling(s string, err error) error {
	return fmt.Errorf("%w: %s: %s", ErrUnmarshallingFailure, s, err)
}

type Price struct {
	Wat string    // Wat is the asset name.
	Val *big.Int  // Val is the asset price multiplied by PriceMultiplier.
	Age time.Time // Age is the time when the price was obtained.
}

// jsonPrice is the JSON representation of the Price structure.
type jsonPrice struct {
	Wat string `json:"wat"`
	Val string `json:"val"`
	Age int64  `json:"age"`
	V   string `json:"v"`
	R   string `json:"r"`
	S   string `json:"s"`
}

func (p *Price) SetFloat64Price(price float64) {
	pf := new(big.Float).SetFloat64(price)
	pf = new(big.Float).Mul(pf, new(big.Float).SetFloat64(PriceMultiplier))
	pi, _ := pf.Int(nil)
	p.Val = pi
}

func (p *Price) Float64Price() float64 {
	x := new(big.Float).SetInt(p.Val)
	x = new(big.Float).Quo(x, new(big.Float).SetFloat64(PriceMultiplier))
	f, _ := x.Float64()
	return f
}

func (p *Price) MarshalJSON() ([]byte, error) {
	bts := p.Sig.Bytes()
	v := bts[64]
	r := bts[:32]
	s := bts[32:64]
	return json.Marshal(jsonPrice{
		Wat: p.Wat,
		Val: p.Val.String(),
		Age: p.Age.Unix(),
		V:   hex.EncodeToString([]byte{v}),
		R:   hex.EncodeToString(r),
		S:   hex.EncodeToString(s),
	})
}

func (p *Price) h() []byte {
	// Median:
	val := make([]byte, 32)
	p.Val.FillBytes(val)

	// Time:
	age := make([]byte, 32)
	binary.BigEndian.PutUint64(age[24:], uint64(p.Age.Unix()))

	// Asset name:
	wat := make([]byte, 32)
	copy(wat, p.Wat)

	h := make([]byte, 96)
	copy(h[0:32], val)
	copy(h[32:64], age)
	copy(h[64:96], wat)
	return h
}
