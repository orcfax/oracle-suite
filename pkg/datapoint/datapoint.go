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

package datapoint

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	lg "log"

	"github.com/defiweb/go-eth/types"

	"github.com/chronicleprotocol/oracle-suite/internal/identity"
	"github.com/chronicleprotocol/oracle-suite/pkg/datapoint/value"
	"github.com/chronicleprotocol/oracle-suite/pkg/log"
	"github.com/chronicleprotocol/oracle-suite/pkg/util/treerender"
)

// Provider provides data points.
//
// A data point is a value obtained from a source. For example, a data
// point can be a price of an asset at a specific time obtained from
// an exchange.
//
// A model describes how a data point is calculated and obtained. For example,
// a model can describe from which sources data points are obtained and how
// they are combined to calculate a final value. Details of how models work
// depend on a specific implementation.
type Provider interface {
	// ModelNames returns a list of supported data models.
	ModelNames(ctx context.Context) []string

	// DataPoint returns a data point for the given model.
	DataPoint(ctx context.Context, model string) (Point, error)

	// DataPoints returns a map of data points for the given models.
	DataPoints(ctx context.Context, models ...string) (map[string]Point, error)

	// Model returns a price model for the given asset pair.
	Model(ctx context.Context, model string) (Model, error)

	// Models describes price models which are used to calculate prices.
	// If no pairs are specified, models for all pairs are returned.
	Models(ctx context.Context, models ...string) (map[string]Model, error)
}

// Signer is responsible for signing data points.
type Signer interface {
	// Supports returns true if the signer supports the given data point.
	Supports(ctx context.Context, data Point) bool

	// Sign signs a data point using the given key.
	Sign(ctx context.Context, model string, data Point) (*types.Signature, error)
}

// Recoverer is responsible for recovering addresses from signatures.
type Recoverer interface {
	// Supports returns true if the recoverer supports the given data point.
	Supports(ctx context.Context, data Point) bool

	// Recover recovers the address from the given signature.
	Recover(ctx context.Context, model string, data Point, signature types.Signature) (*types.Address, error)
}

// Model is a simplified representation of a model which is used to obtain
// a data point. The main purpose of this structure is to help the end
// user to understand how data points values are calculated and obtained.
//
// This structure is purely informational. The way it is used depends on
// a specific implementation.
type Model struct {
	// Meta contains metadata for the model. It should contain information
	// about the model and its parameters.
	//
	// The "type" metadata field is used to determine the type of the model.
	//
	// Meta values must be marshalable to JSON.
	Meta map[string]any

	// Models is a list of sub models used to calculate price.
	Models []Model
}

// MarshalJSON implements the json.Marshaler interface.
func (m Model) MarshalJSON() ([]byte, error) {
	meta := m.Meta
	meta["models"] = m.Models
	return json.Marshal(meta)
}

// MarshalTrace returns a human-readable representation of the model.
func (m Model) MarshalTrace() ([]byte, error) {
	return treerender.RenderTree(func(node any) treerender.NodeData {
		meta := map[string]any{}
		model := node.(Model)
		typ := "node"
		for k, v := range model.Meta {
			if k == "type" {
				typ, _ = v.(string)
				continue
			}
			meta[k] = v
		}
		var models []any
		for _, m := range model.Models {
			models = append(models, m)
		}
		return treerender.NodeData{
			Name:      typ,
			Params:    meta,
			Ancestors: models,
			Error:     nil,
		}
	}, []any{m}, 0), nil
}

// Point represents a data point. It can represent any value obtained from
// an origin. It can be a price, a volume, a market cap, etc. The value
// itself is represented by the Value interface and can be anything, a number,
// a string, or even a complex structure.
//
// Before using this data, you should check if it is valid by calling
// Point.Validate() method.
type Point struct {
	// Value is the value of the data point.
	Value value.Value

	// Time is the time when the data point was obtained.
	Time time.Time

	// SubPoints is a list of sub data points that are used to obtain this
	// data point.
	SubPoints []Point

	// Meta contains metadata for the data point. It may contain additional
	// information about the data point, such as the origin it was obtained
	// from, etc.
	//
	// Meta values must be marshalable to JSON.
	Meta map[string]any

	// Error is an optional error which occurred during obtaining the price.
	// If error is not nil, then the price is invalid and should not be used.
	//
	// Point may be invalid for other reasons, hence you should always check
	// the data point for validity by calling Point.Validate() method.
	Error error
}

// Validate returns an error if the data point is invalid.
func (p Point) Validate() error {
	if p.Error != nil {
		return p.Error
	}
	if p.Value == nil {
		return fmt.Errorf("value is not set")
	}
	if v, ok := p.Value.(value.ValidatableValue); ok {
		if err := v.Validate(); err != nil {
			return err
		}
	}
	if p.Time.IsZero() {
		return fmt.Errorf("time is not set")
	}
	return nil
}

// MarshalJSON implements the json.Marshaler interface.
func (p Point) MarshalJSON() ([]byte, error) {
	data := make(map[string]any)
	data["value"] = p.Value
	data["time"] = p.Time.In(time.UTC).Format(time.RFC3339Nano)
	var points []any
	for _, t := range p.SubPoints {
		points = append(points, t)
	}
	if len(points) > 0 {
		data["sub_points"] = points
	}
	if len(p.Meta) > 0 {
		data["meta"] = p.Meta
	}
	if err := p.Validate(); err != nil {
		data["error"] = err.Error()
	}
	return json.Marshal(data)
}

// MarshalTrace returns a human-readable representation of the tick.
func (p Point) MarshalTrace() ([]byte, error) {
	return treerender.RenderTree(func(node any) treerender.NodeData {
		meta := make(map[string]any)
		point := node.(Point)
		typ := "data_point"
		if point.Value != nil {
			meta["value"] = point.Value.Print()
		}
		meta["time"] = point.Time.In(time.UTC).Format(time.RFC3339Nano)
		var points []any
		for _, t := range point.SubPoints {
			points = append(points, t)
		}
		for k, v := range point.Meta {
			// The trace report is more sensitive to what is output and
			// so custom fields are ignored.
			if k == "headers" || k == "collector" || k == "request_url" {
				continue
			}
			if k == "type" {
				typ, _ = v.(string)
				continue
			}
			meta["meta."+k] = v
		}
		return treerender.NodeData{
			Name:      typ,
			Params:    meta,
			Ancestors: points,
			Error:     point.Validate(),
		}
	}, []any{p}, 0), nil
}

// createContentSignature provides a rudimentary way to create a
// checksum to validate a collector's message against. The message
// helps to prove the origin of a message and helps to protect against
// rounding errors using floating point numbers.
func createContentSignature(timestamp string, values []string, nodeID string) string {
	/* Reference impl:

	```golang
		hash := sha256.New()
		hash.Write([]byte("2023-09-12T14:08:15Z"))
		hash.Write([]byte("0.248848"))
		hash.Write([]byte("0.2489"))
		hash.Write([]byte("0.2488563207"))
		hash.Write([]byte("9165f28e-012e-4790-bf38-cce43184bc7d"))
		hexHash := fmt.Sprintf("%x", bs)
		hexHash == "6dd329aaba26cf4d1175eafef13e8f49b41d2c36be6832987cb559bd715dcfd2"
	```
	*/

	hash := sha256.New()
	hash.Write([]byte(timestamp))
	for _, value := range values {
		hash.Write([]byte(value))
	}
	hash.Write([]byte(nodeID))
	hashBytes := hash.Sum(nil)
	hexHash := fmt.Sprintf("%x", hashBytes)
	return string(hexHash)
}

// priceToString returns a string representation of a price value.
func priceToString(price value.Tick) (string, error) {
	if price.Price != nil {
		return price.Price.String(), nil
	}
	return "", fmt.Errorf("price is nil and cannot be converted")
}

// MarshalOrcfax returns an Orcfax validator collector profile,
func (p Point) MarshalOrcfax() (value.OrcfaxMessage, error) {

	const utcTimeFormat = "2006-01-02T15:04:05Z"

	if p.Error != nil {
		lg.Printf("to handle, errors if not enough data, e.g. TUSD/USD %s", p.Error)
		collectorData := value.OrcfaxCollectorData{}
		msg := value.OrcfaxMessage{}
		collectorData.Errors = append(collectorData.Errors, fmt.Sprintf("%s", p.Error))
		msg.Message = collectorData
		return msg, nil
	}

	medianPrice := p.Value.(value.Tick)
	feedPair := medianPrice.Pair
	collectorData := value.OrcfaxCollectorData{}
	calculated, _ := priceToString(medianPrice)
	collectorData.CalculatedValue = calculated
	collectorData.Timestamp = time.Now().UTC().Format(utcTimeFormat)
	var dataPoints []string
	var rawData []value.OrcfaxRaw

	lg.Println("nb. ADA/BTC requires further testing, e.g. for expired data points")

	for _, t := range p.SubPoints {
		for _, tt := range t.SubPoints {
			origin := tt.Meta["origin"]
			collector := tt.Meta["collector"]
			subPointTick, ok := tt.Value.(value.Tick)
			if !ok {
				collectorData.Errors = append(
					collectorData.Errors,
					fmt.Sprintf(
						"%s: (%s) error with type casting tick",
						feedPair,
						origin,
					),
				)
				// continue onto the next collector.
				continue
			}
			priceConverted, err := priceToString(subPointTick)
			if err != nil {
				collectorData.Errors = append(
					collectorData.Errors,
					fmt.Sprintf("%s: (%s) %s",
						feedPair,
						origin,
						err,
					),
				)
				// continue onto the next collector.
				continue
			}
			val, ok := tt.Meta["headers"].(string)
			if !ok {
				collectorData.Errors = append(
					collectorData.Errors,
					fmt.Sprintf(
						"%s: ('%s') error with type casting header",
						feedPair,
						origin,
					),
				)
				// continue onto the next collector.
				continue
			}
			// continue to add to the raw data output.
			raw := value.OrcfaxRaw{}
			raw.Collector = fmt.Sprintf("%s.%s", origin, collector)
			raw.Response = val
			raw.RequestURL = tt.Meta["request_url"].(string)
			raw.RequestTimestamp = tt.Time.UTC().Format(utcTimeFormat)
			rawData = append(rawData, raw)
			dataPoints = append(dataPoints, priceConverted)
		}
	}

	collectorData.DataPoints = dataPoints
	collectorData.Raw = rawData
	collectorData.ContentSignature = createContentSignature(
		collectorData.Timestamp,
		collectorData.DataPoints,
		"todo-provide-a-node-identifier-here",
	)

	// NB. This is the Chronicle Labs concept of validation. We need to
	// verify this and also augment it with ours.
	if err := p.Validate(); err != nil {
		collectorData.Errors = append(collectorData.Errors, err.Error())
	}

	msg := value.OrcfaxMessage{}
	msg.Message = collectorData

	const idLoc = "/tmp/.node-identity.json"

	// Load the identity to enable it to be updated.
	ident, err := identity.LoadCache(idLoc)
	if err != nil {
		// In future implementations we need the handshake to
		// determine whether or not a new identity can just be
		// created.
		lg.Printf("error loading existing identity: '%s' cannot retrieve previous data", err)
	}

	if (ident == identity.Identity{}) {
		lg.Printf("creating a new id")
	} else {
		lg.Printf("retrieved id: '%s'", ident.NodeID)
	}

	// get default loc for id
	// do something with the websocket...
	websocket := "ws://"

	loc := identity.GetIdentity(ident.NodeID, websocket)
	msg.Message.Identity = loc

	return msg, nil
}

func PointLogFields(p Point) log.Fields {
	fields := log.Fields{}
	if p.Value != nil {
		fields["point.value"] = p.Value.Print()
	}
	if !p.Time.IsZero() {
		fields["point.time"] = p.Time
	}
	if err := p.Validate(); err != nil {
		fields["point.error"] = err.Error()
	}
	for k, v := range p.Meta {
		fields["point.meta."+k] = v
	}
	return fields
}
