// Copyright © 2019 Erin Shepherd
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
	"fmt"

	"github.com/erincandescent/nuvoprog/protocol"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// devicesCmd represents the devices command
var devicesCmd = &cobra.Command{
	Use:   "devices",
	Short: "List connected programmers",
	Long:  `Lisy connected programmers and their firmware versions`,
	RunE: func(cmd *cobra.Command, args []string) error {
		devs, err := protocol.Connect()
		if err != nil {
			return err
		}

		for _, dev := range devs {
			fmt.Printf("[%s] ", dev.Path())
			ver, err := dev.GetVersion()
			if err != nil {
				color.Red(err.Error())
				fmt.Println()
				continue
			}

			fmt.Println(ver)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(devicesCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// devicesCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// devicesCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
