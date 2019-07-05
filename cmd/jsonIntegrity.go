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
	"regexp"
	"strings"
	"sync"

	"github.com/spf13/cobra"

	log "github.com/sirupsen/logrus"
)

// jsonIntegrityCmd represents the jsonIntegrity command
var jsonIntegrityCmd = &cobra.Command{
	Use:   "jsonIntegrity",
	Short: "Validate json structure",
	Long:  `This takes a zip folder with json.gz files and validate every line in those files has a correct json structure`,
	Run: func(cmd *cobra.Command, args []string) {

		r, err := zip.OpenReader(args[0])
		if err != nil {
			panic(err)
		}
		defer r.Close()

		re := regexp.MustCompile(`0001-01-01T00:00:00\.000000Z\+00:00`)

		reGPG := regexp.MustCompile(`"gpg_signed":true`)

		var wg sync.WaitGroup

		wg.Add(len(r.File))

		for _, f := range r.File {
			go func(f *zip.File) {
				defer wg.Done()
				var obj map[string]interface{}
				var fullContentFile string
				if f.Name != "export.json" && strings.Contains(f.Name, ".json") {

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
					buf := make([]byte, 0, 64*1024)
					scanner.Buffer(buf, 1024*1024)
					line := 0
					for scanner.Scan() {
						line++
						str := scanner.Text()

						fullContentFile += str

						if re.Match([]byte(str)) {
							log.Warnf("IT SEEM DATE IS WRONG, LINE(%d), CONTENT => %s, FILE => %s \n", line, str, f.Name)
						}

						err = json.Unmarshal([]byte(str), &obj)
						if err != nil {
							log.Warnf("WRONG JSON FORMAT IN LINE(%d) CONTENT => %s, FILE => %s \n", line, str, f.Name)
							log.Error(err)
						}
					}

					err = scanner.Err()
					if err != nil {
						log.Error(err)
					}

					if strings.Contains(f.Name, "commit.json.gz") {
						if !reGPG.Match([]byte(fullContentFile)) {
							log.Warnf("THIS FILE DOES NOT CONTAIN ANY SIGNED COMMITS, THIS MIGHT BE WRONG, FILE => %s\n", f.Name)
						}
					}
				}
			}(f)
		}

		wg.Wait()
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
