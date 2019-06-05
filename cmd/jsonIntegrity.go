// Copyright © 2019 NAME HERE <EMAIL ADDRESS>
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
	"archive/zip"
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	log "github.com/sirupsen/logrus"
)

// jsonIntegrityCmd represents the jsonIntegrity command
var jsonIntegrityCmd = &cobra.Command{
	Use:   "jsonIntegrity",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {

		// Add command to run project - DONE
		// Add code to read files from zip file
		// Add code to iterate over each file
		// Add code to validate each file

		r, err := zip.OpenReader("/Users/joseordaz/Desktop/pipeline.zip")
		if err != nil {
			panic(err)
		}
		defer r.Close()

		var obj map[string]interface{}

		for _, f := range r.File {
			if f.Name != "export.json" {

				log.Infof("Validating file => %s", f.Name)

				rc, err := f.Open()
				if err != nil {
					panic(err)
				}
				defer rc.Close()

				gz, err := gzip.NewReader(rc)
				if err != nil {
					panic(err)
				}
				defer gz.Close()

				scanner := bufio.NewScanner(gz)

				line := 0
				for scanner.Scan() {
					line++
					str := scanner.Text()

					err = json.Unmarshal([]byte(str), &obj)
					if err != nil {
						log.Warnf("WRONG JSON FORMAT IN LINE(%d) CONTENT => %s \n", line, str)
						log.Error(err)
					}
				}

			}
		}

		fmt.Println("jsonIntegrity called")
	},
}

func init() {
	rootCmd.AddCommand(jsonIntegrityCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// jsonIntegrityCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// jsonIntegrityCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
