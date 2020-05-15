package cmd

import (
	"os"
	"os/exec"
	"strings"

	"github.com/pinpt/go-common/fileutil"
	pstrings "github.com/pinpt/go-common/strings"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// datamodelCmd represents the datamodel command
var datamodelCmd = &cobra.Command{
	Use:   "datamodel",
	Short: "move files easier",
	Run: func(cmd *cobra.Command, args []string) {

		projectDest := "agent"
		fileDest := "*"

		// datamodel work // defaults agent
		// datamodel pipeline work
		// datamodel work/issues.go // defaults agent
		// datamodel pipeline work/issues.go

		// 0 := project
		// 1 := filepath
		switch len(args) {
		case 2:
			fileDest := args[1]
			if !strings.Contains(fileDest, ".go") {
				fileDest += pstrings.JoinURL("/", "*")
			}
			projectDest = args[0]
		case 1:
			fileDest = args[0]
		}

		GoPath := os.Getenv("GOPATH")
		if GoPath == "" {
			log.Error("$GOPATH env not defined")
		}

		baseDir := pstrings.JoinURL(GoPath, "/src/github.com/pinpt/")
		baseSrc := pstrings.JoinURL(baseDir, "datamodel/dist/golang/public/")

		finalSrc := "/" + pstrings.JoinURL(baseSrc, fileDest)
		finalDest := "/" + pstrings.JoinURL(baseDir, projectDest, "vendor/github.com/pinpt/integration-sdk/", fileDest)

		if exists := fileutil.FileExists(finalSrc); !exists {
			log.Error("Does not exits path := ", finalSrc)
		}
		if exists := fileutil.FileExists(finalDest); !exists {
			log.Error("Does not exits path := ", finalDest)
		}

		c := exec.Command("cp", "-v", "-R", finalSrc, finalDest)
		bts, err := c.Output()
		if err != nil {
			log.Error(err)
		}

		log.Info(string(bts))
	},
}

func init() {
	rootCmd.AddCommand(datamodelCmd)
}
