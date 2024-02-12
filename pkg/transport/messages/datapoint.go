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

package messages

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/defiweb/go-eth/types"

	"google.golang.org/protobuf/proto"

	"github.com/orcfax/oracle-suite/pkg/datapoint"
	"github.com/orcfax/oracle-suite/pkg/datapoint/value"
	"github.com/orcfax/oracle-suite/pkg/log"
	"github.com/orcfax/oracle-suite/pkg/transport"
	"github.com/orcfax/oracle-suite/pkg/transport/messages/pb"
)

const DataPointV1MessageName = "data_point/v1"

const maxSubPointReferenceDepth = 2

type DataPoint struct {
	transport.AppInfo

	// Model is the name of the data model.
	Model string

	// Value is a binary representation of the data point.
	Point datapoint.Point

	// Signature is the feed signature of the data point.
	ECDSASignature types.Signature
}

func (d DataPoint) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"model":           d.Model,
		"point":           d.Point,
		"ecdsa_signature": d.ECDSASignature.String(),
	})
}

func (d *DataPoint) Marshall() ([]byte, error) {
	return json.Marshal(d)
}

func (d *DataPoint) Unmarshall(b []byte) error {
	err := json.Unmarshal(b, d)
	if err != nil {
		return err
	}
	return nil
}

// MarshallBinary implements the transport.Message interface.
func (d DataPoint) MarshallBinary() ([]byte, error) {
	var err error
	msg := &pb.DataPointMessage{}
	msg.Model = d.Model
	msg.DataPoint, err = dataPointToProtobuf(d.Point, maxSubPointReferenceDepth)
	if err != nil {
		return nil, err
	}
	msg.EcdsaSignature = d.ECDSASignature.Bytes()
	msg.AppInfo = appInfoToProtobuf(d.AppInfo)
	return proto.Marshal(msg)
}

// UnmarshallBinary implements the transport.Message interface.
func (d *DataPoint) UnmarshallBinary(data []byte) error {
	var err error
	msg := &pb.DataPointMessage{}
	if err := proto.Unmarshal(data, msg); err != nil {
		return err
	}
	d.Model = msg.Model
	if d.Point, err = dataPointFromProtobuf(msg.DataPoint); err != nil {
		return err
	}
	if d.ECDSASignature, err = types.SignatureFromBytes(msg.EcdsaSignature); err != nil {
		return err
	}
	d.AppInfo = appInfoFromProtobuf(msg.AppInfo)
	return nil
}

func dataPointToProtobuf(dp datapoint.Point, referenceDepth int) (*pb.DataPoint, error) {
	var err error
	msg := &pb.DataPoint{}
	if msg.Value, err = dataPointValueToProtobuf(dp.Value); err != nil {
		return nil, err
	}
	isReference := dp.Meta["type"] == "reference"
	if isReference {
		referenceDepth--
	}
	msg.Timestamp = dp.Time.Unix()
	if !(isReference && referenceDepth <= 0) {
		msg.SubPoints = make([]*pb.DataPoint, len(dp.SubPoints))
		for i, subPoint := range dp.SubPoints {
			msg.SubPoints[i], err = dataPointToProtobuf(subPoint, referenceDepth)
			if err != nil {
				return nil, err
			}
		}
	}
	msg.Meta = make(map[string][]byte, len(dp.Meta))
	for k, v := range dp.Meta {
		val, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		msg.Meta[k] = val
	}
	return msg, nil
}

func dataPointFromProtobuf(msg *pb.DataPoint) (datapoint.Point, error) {
	if msg == nil {
		return datapoint.Point{}, errors.New("data point is nil")
	}
	var err error
	dp := datapoint.Point{}
	dp.Value, err = dataPointValueFromProtobuf(msg.Value)
	if err != nil {
		return datapoint.Point{}, err
	}
	dp.Time = time.Unix(msg.Timestamp, 0)
	dp.SubPoints = make([]datapoint.Point, len(msg.SubPoints))
	for i, subPoint := range msg.SubPoints {
		dp.SubPoints[i], err = dataPointFromProtobuf(subPoint)
		if err != nil {
			return datapoint.Point{}, err
		}
	}
	dp.Meta = make(map[string]any, len(msg.Meta))
	for k, v := range msg.Meta {
		var val any
		if err := json.Unmarshal(v, &val); err != nil {
			return datapoint.Point{}, err
		}
		dp.Meta[k] = val
	}
	return dp, nil
}

func dataPointValueToProtobuf(val value.Value) (*pb.DataPointValue, error) {
	var err error
	msg := &pb.DataPointValue{}
	switch typ := val.(type) {
	case value.StaticValue:
		var static []byte
		if typ.Value != nil {
			static, err = decFloatPointToBytes(typ.Value)
			if err != nil {
				return nil, err
			}
		}
		msg.Value = &pb.DataPointValue_Static{
			Static: static,
		}
	case value.Tick:
		var (
			price     []byte
			volume24H []byte
		)
		if typ.Price != nil {
			price, err = decFloatPointToBytes(typ.Price)
			if err != nil {
				return nil, err
			}
		}
		if typ.Volume24h != nil {
			volume24H, err = decFloatPointToBytes(typ.Volume24h)
			if err != nil {
				return nil, err
			}
		}
		msg.Value = &pb.DataPointValue_Tick{
			Tick: &pb.DataPointTickValue{
				Pair:      typ.Pair.String(),
				Price:     price,
				Volume24H: volume24H,
			},
		}
	}
	return msg, nil
}

func dataPointValueFromProtobuf(msg *pb.DataPointValue) (value.Value, error) {
	if msg == nil {
		return nil, errors.New("data point value is nil")
	}
	switch typ := msg.Value.(type) {
	case *pb.DataPointValue_Static:
		val := value.StaticValue{}
		if typ.Static != nil {
			static, err := bytesToDecFloatPoint(typ.Static)
			if err != nil {
				return nil, err
			}
			val.Value = static
		}
		return val, nil
	case *pb.DataPointValue_Tick:
		val := value.Tick{}
		pair, err := value.PairFromString(typ.Tick.Pair)
		if err != nil {
			return nil, err
		}
		val.Pair = pair
		if typ.Tick.Price != nil {
			price, err := bytesToDecFloatPoint(typ.Tick.Price)
			if err != nil {
				return nil, err
			}
			val.Price = price
		}
		if typ.Tick.Volume24H != nil {
			volume24H, err := bytesToDecFloatPoint(typ.Tick.Volume24H)
			if err != nil {
				return nil, err
			}
			val.Volume24h = volume24H
		}
		return val, nil
	}
	return nil, nil
}

func (d *DataPoint) GobEncode() ([]byte, error) {
	return d.MarshallBinary()
}

func (d *DataPoint) GobDecode(b []byte) error {
	return d.UnmarshallBinary(b)
}

func DataPointMessageLogFields(d DataPoint) log.Fields {
	f := log.Fields{
		"point.model":     d.Model,
		"point.signature": d.ECDSASignature.String(),
	}
	for k, v := range datapoint.PointLogFields(d.Point) {
		f[k] = v
	}
	return f
}
