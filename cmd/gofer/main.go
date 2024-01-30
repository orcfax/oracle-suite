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
	"fmt"
	"os"

	suite "github.com/chronicleprotocol/oracle-suite"
	"github.com/chronicleprotocol/oracle-suite/cmd"
	"github.com/chronicleprotocol/oracle-suite/pkg/config/dataprovider"
	gofer "github.com/chronicleprotocol/oracle-suite/pkg/config/gofernext"
	"github.com/spf13/cobra"
)

var (
	version = "dev-0.0.0"
	commit  = "000000000000000000000000000000000baddeed"
	date    = "1970-01-01T00:00:01Z"
)

const appname = "gofer"
const logTimeFormat = "2006-01-02 15:04:05"

var agent string = dataprovider.UserAgent

func versionFunc() *cobra.Command {
	var versionCmd = &cobra.Command{
		Use:   "version",
		Short: "print the version details",
		Long:  `print the version details`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(os.Stderr, "%s: %s (%s) commit: %s date: %s\n", appname, agent, version, commit, date)
		},
	}
	return versionCmd
}

func main() {
	var config gofer.Config
	cf := cmd.ConfigFlagsForConfig(config)

	var lf cmd.LoggerFlags
	c := cmd.NewRootCommand("gofer", suite.Version, &cf, &lf)

	c.AddCommand(
		cmd.NewRunCmd(&config, &cf, &lf),
		cmd.NewRenderConfigCmd(&config, &cf),
		NewModelsCmd(&config, &cf, &lf),
		NewDataCmd(&config, &cf, &lf),
		versionFunc(),
	)

	if err := c.Execute(); err != nil {
		os.Exit(1)
	}
}
