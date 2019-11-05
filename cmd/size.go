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
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strconv"

	"github.com/spf13/cobra"
)

type item struct {
	name string
	size float64
}

type bySize []item

func (a bySize) Len() int           { return len(a) }
func (a bySize) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a bySize) Less(i, j int) bool { return a[i].size > a[j].size }

// sizeCmd represents the size command
var sizeCmd = &cobra.Command{
	Use:   "size [FOLDER]",
	Short: "Prints FOLDER size",
	Long:  `Checks folder size resursively`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			items := make([]item, 0)
			dir, err := os.Getwd()
			if err != nil {
				log.Fatal(err)
			}
			files, err := ioutil.ReadDir(dir)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			for _, file := range files {
				if file.IsDir() {
					items = append(items, item{file.Name(), getFolderSize(dir + "/" + file.Name())})
				} else {
					items = append(items, item{file.Name(), float64(file.Size())})
				}
			}
			longestName := len(items[0].name)
			for item := range items {
				if len(items[item].name) > longestName {
					longestName = len(items[item].name)
				}
			}
			condition := fmt.Sprintf("%s%d%s %s%d%s\n", "%-", longestName, "s", "%-", longestName, "s")
			sort.Sort(bySize(items))
			for item := range items {
				name := items[item].name
				size := items[item].size
				fmt.Printf(condition, name, getSize(size))
			}
		} else {
			folder := args[0]
			fmt.Println(checkFolderSize(folder))
		}
	},
}

func checkFolderSize(folder string) string {
	size := getFolderSize(folder)
	return getSize(size)
}

func getSize(size float64) string {
	switch {
	case size < 1024:
		return formatFloat(size, 'b')
	case size < 1048576:
		return formatFloat(size/1024, 'K')
	case size < 1073741824:
		return formatFloat(size/1048576, 'M')
	case size < 1099511627776:
		return formatFloat(size/1073741824, 'G')
	}
	return "-1"
}

func formatFloat(size float64, unit rune) string {
	s := strconv.FormatFloat(size, 'f', 0, 64)
	return fmt.Sprintf("%-6s %s", s, string(unit))
}

func getFolderSize(folder string) float64 {
	files, err := ioutil.ReadDir(folder)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	var size float64
	for _, file := range files {
		if file.IsDir() {
			size += getFolderSize(folder + "/" + file.Name())
		} else {
			size += float64(file.Size())
		}
	}
	return size
}

func init() {
	rootCmd.AddCommand(sizeCmd)
}
