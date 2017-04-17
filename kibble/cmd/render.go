// Copyright © 2017 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"github.com/indiereign/shift72-kibble/kibble/render"
	"github.com/spf13/cobra"
)

var renderRunAsAdmin bool
var verbose bool

// renderCmd represents the render command
var renderCmd = &cobra.Command{
	Use:   "render",
	Short: "Render the entire site",
	Long: `Render templates using the available data.

Kibble is used to build and develop custom sites to run on the SHIFT72 platform.`,
	Run: func(cmd *cobra.Command, args []string) {
		render.Render(renderRunAsAdmin, verbose)
	},
}

func init() {
	RootCmd.AddCommand(renderCmd)
	renderCmd.Flags().BoolVar(&renderRunAsAdmin, "admin", false, "Render using admin credentials")
	renderCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "verbose logging")
}
