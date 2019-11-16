package cmd

import (
	"os"
	"os/exec"

	"github.com/pinpt/go-common/fileutil"
	pstrings "github.com/pinpt/go-common/strings"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// datamodelCmd represents the datamodel command
var datamodelCmd = &cobra.Command{
	Use:   "datamodel",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		projectDest := "agent.next/"
		file := "*"
		fileDest := ""

		// 0 := project
		// 1 := filepath
		switch len(args) {
		case 2:
			file = args[1]
			fileDest = file
			fallthrough
		case 1:
			projectDest = args[0]
		}

		GoPath := os.Getenv("GOPATH")
		if GoPath == "" {
			log.Error("$GOPATH env not defined")
		}

		baseDir := pstrings.JoinURL(GoPath, "/src/github.com/pinpt/")
		baseSrc := pstrings.JoinURL(baseDir, "datamodel/dist/golang/public/")

		finalSrc := "/" + pstrings.JoinURL(baseSrc, file)
		finalDest := "/" + pstrings.JoinURL(baseDir, projectDest, "vendor/github.com/pinpt/integration-sdk/", fileDest)

		if exists := fileutil.FileExists(finalSrc); !exists {
			log.Error("Does not exits path := ", finalSrc)
		}
		if exists := fileutil.FileExists(finalDest); !exists {
			log.Error("Does not exits path := ", finalSrc)
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
