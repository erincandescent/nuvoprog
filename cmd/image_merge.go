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
	"errors"
	"fmt"

	"github.com/erincandescent/nuvoprog/target"
	"github.com/spf13/cobra"
)

// imageMergeCmd represents the imageMerge command
var imageMergeCmd = &cobra.Command{
	Use:   "merge",
	Short: "Merge image files",
	Long:  `Merges configuration, APROM and optionally LDROM images into a composite image`,
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
		output, _ := cmd.Flags().GetString("output")

		d, err := ReadTargetData(config, image, aprom, ldrom, td, true)
		if err != nil {
			return err
		}

		w, err := openWrite(output)
		if err != nil {
			return err
		}
		d.Write(w)

		return nil
	},
}

func init() {
	imageCmd.AddCommand(imageMergeCmd)
	imageMergeCmd.Flags().StringP("output", "o", "", "Output file, e.g. image.ihx")
}
