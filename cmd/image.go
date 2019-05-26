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
	"github.com/spf13/cobra"
)

// imageCmd represents the image command
var imageCmd = &cobra.Command{
	Use:   "image",
	Short: "Image manipulation commands",
	Long:  `Commands for manipulating images`,
}

func init() {
	rootCmd.AddCommand(imageCmd)

	imageCmd.PersistentFlags().StringP("image", "i", "", "Image file, e.g. image.ihx")
	imageCmd.PersistentFlags().StringP("config", "c", "", "Configuration, e.g. 6FFBFFFF or @config.json")
	imageCmd.PersistentFlags().StringP("aprom", "a", "", "APROM file e.g. aprom.ihx")
	imageCmd.PersistentFlags().StringP("ldrom", "l", "", "LDROM file e.g. ldrom.ihx")
}
