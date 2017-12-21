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
	"os"

	"github.com/spf13/cobra"
)

// sizeCmd represents the size command
var sizeCmd = &cobra.Command{
	Use:   "size",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		folder := args[0]
		size := getFolderSize(folder)
		switch {
		case size < 1024:
			fmt.Println(size, "b")
		case size < 1048576:
			fmt.Printf("%.2f %s\n", size/1024, "K")
		case size < 1073741824:
			fmt.Printf("%.2f %s\n", size/1048576, "M")
		case size < 1099511627776:
			fmt.Printf("%.2f %s\n", size/1073741824, "G")

		}
	},
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
