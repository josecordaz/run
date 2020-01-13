package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	githubToken string = os.Getenv("PP_GITHUB_TOKEN")
	filePATH    string = os.Getenv("PP_FILE_PATH")
	repos       string = os.Getenv("PP_REPOS")
)

var datamodelStatusCmd = &cobra.Command{
	Use:   "datamodel-status",
	Short: "check datamodel version on all apps",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {

		if githubToken == "" {
			fmt.Println("missing PP_GITHUB_TOKEN env var")
			os.Exit(1)
		}

		if filePATH == "" {
			fmt.Println("missing PP_FILE_PATH env var")
			os.Exit(1)
		}

		if repos == "" {
			fmt.Println("missing PP_REPOS env var")
			os.Exit(1)
		}

		repos := strings.Split(repos, ",")

		for _, repo := range repos {
			bts, err := request("https://api.github.com/repos/pinpt/"+repo+"/contents/"+filePATH+"?ref=master", nil)
			if err != nil {
				panic(err)
			}
			text, err := getDatamodelInfo(bts)
			if err != nil {
				panic(err)
			}
			fmt.Printf("%-12s %s\n", repo, strings.Split(text, ":")[1])
		}

	},
}

func getDatamodelInfo(data []byte) (string, error) {

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "datamodel") {
			return scanner.Text(), nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", nil
}

func init() {
	rootCmd.AddCommand(datamodelStatusCmd)
}

func request(url string, params url.Values) ([]byte, error) {

	if len(params) != 0 {
		url += "?" + params.Encode()
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github.v3.raw")
	req.Header.Set("Authorization", "token "+githubToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bts, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return bts, nil
}
