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
	"encoding/json"
	"errors"
	"fmt"

	"github.com/erincandescent/nuvoprog/target"
	"github.com/spf13/cobra"
)

// imageSplit represents the imageMerge command
var imageSplit = &cobra.Command{
	Use:   "split",
	Short: "Split image files",
	Long:  `Splits an image file into APROM, LDROM and Config components`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if targetName == "" {
			return errors.New("Target device not specified")
		}

		td := target.ByName(targetName)
		if td == nil {
			return fmt.Errorf("Target device '%s' not found", targetName)
		}

		config, _ := cmd.Flags().GetString("config")
		image, _ := cmd.Flags().GetString("image")
		aprom, _ := cmd.Flags().GetString("aprom")
		ldrom, _ := cmd.Flags().GetString("ldrom")

		d, err := ReadTargetData("", image, "", "", td, true)
		if err != nil {
			return err
		}

		if config != "" {
			if len(d.Config) == 0 {
				return errors.New("Asked to write config which is not present")
			}

			cfg, err := td.Config.Decode(d.Config)
			if err != nil {
				return err
			}

			buf, err := json.MarshalIndent(cfg, "", "    ")
			if err != nil {
				return err
			}

			f, err := openWrite(config)
			if err != nil {
				return err
			}

			if _, err := f.Write(buf); err != nil {
				return err
			}

			if err := f.Close(); err != nil {
				return err
			}
		}

		if aprom != "" {
			f, err := openWrite(aprom)
			if err != nil {
				return err
			}

			if err := d.WriteAPROM(f); err != nil {
				return err
			}
		}

		if ldrom != "" {
			f, err := openWrite(ldrom)
			if err != nil {
				return err
			}

			if err := d.WriteLDROM(f); err != nil {
				return err
			}
		}

		return nil
	},
}

func init() {
	imageCmd.AddCommand(imageSplit)
}
