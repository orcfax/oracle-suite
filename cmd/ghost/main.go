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

package main

import (
	"os"

	suite "github.com/orcfax/oracle-suite"
	"github.com/orcfax/oracle-suite/cmd"
	ghost "github.com/orcfax/oracle-suite/pkg/config/ghostnext"
)

func main() {
	var config ghost.Config
	cf := cmd.ConfigFlagsForConfig(config)

	var lf cmd.LoggerFlags
	c := cmd.NewRootCommand("ghost", suite.Version, &cf, &lf)

	c.AddCommand(
		cmd.NewRunCmd(&config, &cf, &lf),
		cmd.NewRenderConfigCmd(&config, &cf),
	)

	if err := c.Execute(); err != nil {
		os.Exit(1)
	}
}
