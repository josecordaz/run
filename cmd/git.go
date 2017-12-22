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
	"bytes"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

// gitCmd represents the git command
var gitCmd = &cobra.Command{
	Use:   "git [MESSAGE]",
	Short: "add, commit and push",
	Long: `This will execute
		git add .
		git commit -m MESSAGE
		git push`,
	Run: func(cmd *cobra.Command, args []string) {
		var out bytes.Buffer
		var stderr bytes.Buffer

		comm := exec.Command("git", "add", ".")
		comm.Stdout = &out
		comm.Stderr = &stderr
		err := comm.Run()
		if err != nil {
			fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
			os.Exit(1)
		}
		// fmt.Println(out.String())
		// out.Reset()

		comm = exec.Command("git", "commit", "-m", args[0])
		comm.Stdout = &out
		comm.Stderr = &stderr
		err = comm.Run()
		if err != nil {
			fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
			os.Exit(1)
		}
		// fmt.Println(out.String())
		// out.Reset()

		comm = exec.Command("git", "push")
		comm.Stdout = &out
		comm.Stderr = &stderr
		err = comm.Run()
		if err != nil {
			fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
			os.Exit(1)
		}
		fmt.Println(out.String())
	},
}

func init() {
	rootCmd.AddCommand(gitCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// gitCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// gitCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
