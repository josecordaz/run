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
	"strings"

	"github.com/spf13/cobra"
)

type Item struct {
	name string
	size float64
}

type BySize []Item

func (a BySize) Len() int           { return len(a) }
func (a BySize) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a BySize) Less(i, j int) bool { return a[i].size > a[j].size }

// sizeCmd represents the size command
var sizeCmd = &cobra.Command{
	Use:   "size [FOLDER]",
	Short: "Prints FOLDER size",
	Long:  `Checks folder size resursively`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			items := make([]Item, 0)
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
					items = append(items, Item{file.Name(), getFolderSize(dir + "/" + file.Name())})
				} else {
					items = append(items, Item{file.Name(), float64(file.Size())})
				}
			}
			longestName := len(items[0].name)
			for item := range items {
				if len(items[item].name) > longestName {
					longestName = len(items[item].name)
				}
			}
			sort.Sort(BySize(items))
			for item := range items {
				name := items[item].name
				size := items[item].size
				fmt.Printf("%s %s %s\n", name, strings.Repeat(" ", longestName-len(name)), getSize(size))
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
	//strconv.FormatFloat(input_num, 'f', 6, 64)
	switch {
	case size < 1024:
		s := strconv.FormatFloat(size, 'f', 0, 64)
		return fmt.Sprint(s, strings.Repeat(" ", 6-len(s)), "b")
	case size < 1048576:
		s := strconv.FormatFloat(size/1024, 'f', 2, 64)
		return fmt.Sprint(s, strings.Repeat(" ", 6-len(s)), "K")
	case size < 1073741824:
		s := strconv.FormatFloat(size/1048576, 'f', 2, 64)
		return fmt.Sprint(s, strings.Repeat(" ", 6-len(s)), "M")
	case size < 1099511627776:
		s := strconv.FormatFloat(size/1073741824, 'f', 2, 64)
		return fmt.Sprint(s, strings.Repeat(" ", 6-len(s)), "G")
	}
	return "-1"
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

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// sizeCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// sizeCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
