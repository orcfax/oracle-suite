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

package dataprovider

import (
	"fmt"
	"net/http"

	"github.com/orcfax/oracle-suite/pkg/datapoint/origin"
	utilHCL "github.com/orcfax/oracle-suite/pkg/util/hcl"

	"github.com/hashicorp/hcl/v2"
)

// Configure UserAgent dynamically through build parameters and
// initialize below. Export for verification in gofer version info.
var UserAgent string

var userAgentApp = "orcfax-chronicle-collector"

// version is added dynamically through -ldflags options providing
// access to release tags.
var version = "0.0.0"

func init() {
	UserAgent = fmt.Sprintf("%s/%s", userAgentApp, version)
}

type configOrigin struct {
	// Name of the origin.
	Name string `hcl:"name,label"`

	// Type is the type of the origin.
	Type string `hcl:"type"`

	// OriginConfig is the configuration of the origin.
	// Handled by PostDecodeBlock method.
	OriginConfig any

	// HCL fields:
	Content hcl.BodyContent `hcl:",content"`
	Remain  hcl.Body        `hcl:",remain"`
	Range   hcl.Range       `hcl:",range"`
}

// configOriginStatic is a configuration for the static origin.
type configOriginStatic struct{}

// configOriginTickGenericJQ is a configuration for the TickGenericJQ origin.
type configOriginTickGenericJQ struct {
	URL string `hcl:"url"` // Do not use config.URL because it encodes $ sign
	JQ  string `hcl:"jq"`
}

func (c *configOrigin) PostDecodeBlock(
	ctx *hcl.EvalContext,
	_ *hcl.BodySchema,
	_ *hcl.Block,
	_ *hcl.BodyContent) hcl.Diagnostics {

	var config any
	switch c.Type {
	case "static":
		config = &configOriginStatic{}
	case "tick_generic_jq":
		config = &configOriginTickGenericJQ{}
	default:
		return hcl.Diagnostics{{
			Severity: hcl.DiagError,
			Summary:  "Validation error",
			Detail:   fmt.Sprintf("Unknown origin: %s", c.Type),
			Subject:  c.Range.Ptr(),
		}}
	}
	if diags := utilHCL.Decode(ctx, c.Remain, config); diags.HasErrors() {
		return diags
	}
	c.OriginConfig = config
	return nil
}

func (c configOrigin) OnEncodeBlock(body *utilHCL.Block) hcl.Diagnostics {
	return utilHCL.Encode(c.OriginConfig, body)
}

func (c *configOrigin) configureOrigin(d Dependencies) (origin.Origin, error) {
	switch o := c.OriginConfig.(type) {
	case *configOriginStatic:
		return origin.NewStatic(), nil
	case *configOriginTickGenericJQ:
		// Add an Orcfax user-agent.
		headers := http.Header{}
		headers.Add("user-agent", UserAgent)
		origin, err := origin.NewTickGenericJQ(origin.TickGenericJQConfig{
			URL:   o.URL,
			Query: o.JQ,
			// Headers are nil in the default configuration. We set
			// these for Orcfax here.
			Headers: headers,
			Client:  d.HTTPClient,
			Logger:  d.Logger,
		})
		if err != nil {
			return nil, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Runtime error",
				Detail:   fmt.Sprintf("Failed to create jq origin: %s", err),
				Subject:  c.Range.Ptr(),
			}
		}
		return origin, nil
	}
	return nil, fmt.Errorf("unknown origin %s", c.Type)
}
