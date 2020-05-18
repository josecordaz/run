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
		var itemDestination string

		// datamodel work // defaults agent
		// datamodel pipeline work
		// datamodel work/issues.go // defaults agent
		// datamodel pipeline work/issues.go

		// 0 := project
		// 1 := filepath
		switch len(args) {
		case 2:
			itemDestination = args[1]
			projectDest = args[0]
		case 1:
			itemDestination = args[0]
		}

		goPath := os.Getenv("GOPATH")
		if goPath == "" {
			log.Error("$GOPATH env not defined")
		}

		goPinpointPath := pstrings.JoinURL(goPath, "/src/github.com/pinpt/")
		sourceFolder := pstrings.JoinURL(goPinpointPath, "datamodel/dist/golang/public/")

		finalSrc := "/" + pstrings.JoinURL(sourceFolder, itemDestination)
		finalDest := "/" + pstrings.JoinURL(goPinpointPath, projectDest, "vendor/github.com/pinpt/integration-sdk/", itemDestination)

		if exists := fileutil.FileExists(finalSrc); !exists {
			log.Error("Does not exits path := ", finalSrc)
		}
		if exists := fileutil.FileExists(finalDest); !exists {
			log.Error("Does not exits path := ", finalDest)
		}

		if !strings.Contains(itemDestination, ".go") {
			finalSrc += "/*"
		}

		c := exec.Command("sh", "-c", "cp -v -R "+finalSrc+" "+finalDest)
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
