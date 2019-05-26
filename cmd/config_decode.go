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

// configDecodeCmd represents the configDecode command
var configDecodeCmd = &cobra.Command{
	Use:   "decode",
	Short: "Decodes configuration bytes",
	Long:  `Takes either a config string or an image and decodes configuration bytes`,
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
		data, err := ReadTargetData(config, image, "", "", td, false)
		if err != nil {
			return err
		}

		cfgo := td.Config.NewConfig()
		if err := cfgo.UnmarshalBinary(data.Config); err != nil {
			return err
		}

		buf, err := json.MarshalIndent(cfgo, "", "    ")
		if err != nil {
			return err
		}

		fmt.Println(string(buf))

		return nil
	},
}

func init() {
	configCmd.AddCommand(configDecodeCmd)

	configDecodeCmd.Flags().StringP("image", "i", "", "Image file, e.g. image.ihx")
	configDecodeCmd.Flags().StringP("config", "c", "", "Configuration, e.g. 6FFBFFFF or @config.json")
}
