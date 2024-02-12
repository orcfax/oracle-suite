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

package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/orcfax/oracle-suite/pkg/datapoint"
)

const (
	formatPlain  = "plain"
	formatTrace  = "trace"
	formatJSON   = "json"
	formatOrcfax = "orcfax"
)

type formatTypeValue struct {
	format string
}

func (v *formatTypeValue) String() string {
	if v.format == "" {
		return formatPlain
	}
	return v.format
}

func (v *formatTypeValue) Set(s string) error {
	switch strings.ToLower(s) {
	case formatPlain:
		v.format = formatPlain
	case formatTrace:
		v.format = formatTrace
	case formatJSON:
		v.format = formatJSON
	case formatOrcfax:
		v.format = formatOrcfax
	default:
		return fmt.Errorf("unsupported format: %s", s)
	}
	return nil
}

func (v *formatTypeValue) Type() string {
	return "plain|trace|json|orcfax"
}

func getModelsNames(ctx context.Context, provider datapoint.Provider, args []string) []string {
	if len(args) == 0 {
		return provider.ModelNames(ctx)
	}
	return args
}
