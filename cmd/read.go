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
	"github.com/erincandescent/nuvoprog/protocol"
	"github.com/spf13/cobra"
)

// readCmd represents the read command
var readCmd = &cobra.Command{
	Use:   "read [outfile.ihx]",
	Short: "Read device flash contents",
	Long:  `Read out the contents of the device's flash`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		dev, td, err := connectToTarget()
		if err != nil {
			return err
		}
		defer resetAndCloseDevice(dev)

		d := NewTargetData(td)

		if td.Config.ReadSize != 0 {
			bytes, err := dev.ReadMemory(protocol.ConfigSpace, 0, td.Config.ReadSize)
			if err != nil {
				return err
			}

			d.Config = bytes
		}

		aprom, err := d.APROM()
		if err != nil {
			return nil
		}

		ldrom, err := d.LDROM()
		if err != nil {
			return nil
		}

		for i := 0; i < len(aprom); i += 32 {
			data, err := dev.ReadMemory(protocol.ProgramSpace, uint16(i), 32)
			if err != nil {
				return err
			}

			copy(aprom[i:], data)
		}

		for i := 0; i < len(ldrom); i += 32 {
			data, err := dev.ReadMemory(protocol.ProgramSpace, uint16(i+int(td.LDROMOffset)), 32)
			if err != nil {
				return err
			}

			copy(ldrom[i:], data)
		}

		w, err := openWrite(args[0])
		if err != nil {
			return err
		}
		d.Write(w)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(readCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// readCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// readCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
