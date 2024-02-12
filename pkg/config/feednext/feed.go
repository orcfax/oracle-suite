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

package feed

import (
	"fmt"
	"time"

	"github.com/hashicorp/hcl/v2"

	"github.com/orcfax/oracle-suite/pkg/datapoint"
	"github.com/orcfax/oracle-suite/pkg/datapoint/signer"

	ethereumConfig "github.com/orcfax/oracle-suite/pkg/config/ethereum"
	"github.com/orcfax/oracle-suite/pkg/feed"

	"github.com/orcfax/oracle-suite/pkg/log"
	"github.com/orcfax/oracle-suite/pkg/transport"
	"github.com/orcfax/oracle-suite/pkg/util/timeutil"
)

const (
	tickPriceBroadcastMaxPrecision  = 18
	tickVolumeBroadcastMaxPrecision = 18
)

type Config struct {
	// EthereumKey is the name of the Ethereum key to use for signing prices.
	EthereumKey string `hcl:"ethereum_key"`

	// Interval is the interval at which to publish prices in seconds.
	Interval uint32 `hcl:"interval"`

	DataModels []string `hcl:"data_models"`

	// HCL fields:
	Range   hcl.Range       `hcl:",range"`
	Content hcl.BodyContent `hcl:",content"`

	// Configured service:
	feed *feed.Feed
}

type Dependencies struct {
	KeysRegistry ethereumConfig.KeyRegistry
	DataProvider datapoint.Provider
	Transport    transport.Service
	Logger       log.Logger
}

func (c *Config) ConfigureFeed(d Dependencies) (*feed.Feed, error) {
	if c.feed != nil {
		return c.feed, nil
	}
	if c.Interval == 0 {
		return nil, hcl.Diagnostics{&hcl.Diagnostic{
			Summary:  "Validation error",
			Detail:   "Interval cannot be zero",
			Severity: hcl.DiagError,
			Subject:  c.Content.Attributes["interval"].Range.Ptr(),
		}}
	}
	ethereumKey, ok := d.KeysRegistry[c.EthereumKey]
	if !ok {
		return nil, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Validation error",
			Detail:   fmt.Sprintf("Ethereum key %q is not configured", c.EthereumKey),
			Subject:  c.Content.Attributes["ethereum_key"].Range.Ptr(),
		}
	}
	hooks := []feed.Hook{
		feed.NewTickPrecisionHook(tickPriceBroadcastMaxPrecision, tickVolumeBroadcastMaxPrecision),
		feed.NewTickTraceHook(),
	}
	cfg := feed.Config{
		DataModels:   c.DataModels,
		DataProvider: d.DataProvider,
		Signers:      []datapoint.Signer{signer.NewTickSigner(ethereumKey)},
		Hooks:        hooks,
		Transport:    d.Transport,
		Interval:     timeutil.NewTicker(time.Second * time.Duration(c.Interval)),
		Logger:       d.Logger,
	}
	feedService, err := feed.New(cfg)
	if err != nil {
		return nil, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Runtime error",
			Detail:   fmt.Sprintf("Failed to create the ConfigureFeed service: %v", err),
			Subject:  c.Range.Ptr(),
		}
	}
	c.feed = feedService
	return feedService, nil
}
