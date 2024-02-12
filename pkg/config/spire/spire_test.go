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

package spire

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/orcfax/oracle-suite/pkg/config"
	"github.com/orcfax/oracle-suite/pkg/log/null"
)

func TestConfig(t *testing.T) {
	tests := []struct {
		name string
		path string
		test func(*testing.T, *Config)
	}{
		{
			name: "agent",
			path: "config.hcl",
			test: func(t *testing.T, cfg *Config) {
				services, err := cfg.Services(null.New(), "", "")
				require.NoError(t, err)
				require.NotNil(t, services)
			},
		},
		{
			name: "client",
			path: "config.hcl",
			test: func(t *testing.T, cfg *Config) {
				services, err := cfg.ClientServices(null.New(), "", "")
				require.NoError(t, err)
				require.NotNil(t, services.SpireClient)
				require.NotNil(t, services.Logger)
			},
		},
		{
			name: "stream",
			path: "config.hcl",
			test: func(t *testing.T, cfg *Config) {
				services, err := cfg.StreamServices(null.New(), "", "")
				require.NoError(t, err)
				require.NotNil(t, services.Transport)
				require.NotNil(t, services.Logger)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var cfg Config
			err := config.LoadFiles(&cfg, []string{"./testdata/" + test.path})
			require.NoError(t, err)
			test.test(t, &cfg)
		})
	}
}
