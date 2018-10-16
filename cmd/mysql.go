// Copyright © 2018 NAME HERE <EMAIL ADDRESS>
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
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"time"

	"github.com/yeka/zip"

	_ "github.com/go-sql-driver/mysql"
	"github.com/spf13/cobra"
)

const DB = "db"
const FILE = "file"

const dbSize = `
SELECT 
    SUM(data_length + index_length) / 1024 / 1024 AS 'Size (MB)'
FROM
	information_schema.TABLES
WHERE
	table_schema = ?
GROUP BY table_schema
`

// mysqlCmd represents the mysql command
var mysqlCmd = &cobra.Command{
	Use:   "mysql",
	Short: "Automate import db",
	Long:  `This command will listen for changes in FOLDER and if there is a new DB it will start adding it to mysql`,
	Run: func(cmd *cobra.Command, args []string) {
		host, err := cmd.Flags().GetString("databaseHost")
		checkError(err)
		dbPass, err := cmd.Flags().GetString("databasePassword")
		checkError(err)
		zipPass, err := cmd.Flags().GetString("filePassword")
		checkError(err)
		folder, err := cmd.Flags().GetString("folder")
		checkError(err)
		port, err := cmd.Flags().GetString("databasePort")
		checkError(err)

		files, err := getFiles(folder)
		checkError(err)

		dbs, err := getDBs(host, dbPass, port)
		checkError(err)

		doEvery(5*time.Second, func(t time.Time) {
			files, err = getFiles(folder)
			checkError(err)
			dbs, err = getDBs(host, dbPass, port)
			checkError(err)
			if file, db := different(files, dbs); file != "" {
				fmt.Println("There were changes!!", file)
				fmt.Println("Unzipping file...")
				tmpFile, err := unzip(folder+"/"+file, zipPass)
				checkError(err)

				fmt.Println("Creating db " + db + "...")
				err = createDB(host, port, dbPass, db)
				checkError(err)

				dbs, err = getDBs(host, dbPass, port)
				checkError(err)

				fmt.Println("Importing db " + db + "...")
				start := time.Now()
				err = importDB(&tmpFile, dbPass, host, port, db)
				checkError(err)

				fmt.Printf("Imported %s successfully! Took %s\n", db, time.Since(start))
			} else {
				fmt.Println("Listening for changes!!")
			}
		})
	},
}

func init() {
	rootCmd.AddCommand(mysqlCmd)
	mysqlCmd.Flags().StringP("databaseHost", "d", "localhost", "database host adress")
	mysqlCmd.Flags().StringP("databasePassword", "w", "", "database password")
	mysqlCmd.Flags().StringP("filePassword", "e", "", "zip file password")
	mysqlCmd.Flags().StringP("databasePort", "p", "3306", "database port")
	mysqlCmd.Flags().StringP("folder", "f", "", "folder")
	mysqlCmd.MarkFlagRequired("folder")
	mysqlCmd.MarkFlagRequired("filePassword")
}

func filterStrings(names []string, tp string) (filtered map[string]string, err error) {
	filtered = make(map[string]string)
	for _, name := range names {
		match, err := matchStr(name, tp)
		if err != nil {
			return nil, err
		}
		if match {
			form := formatStr(name, tp)
			filtered[form] = name
		}
	}
	return filtered, nil
}

func formatStr(str string, tp string) string {
	switch tp {
	case DB:
		{
			re := regexp.MustCompile("(bs|ss)(_|)(\\d{4})(\\d{2}|)(\\d{2})(.sql|).zip")
			return re.ReplaceAllString(str, "$1$3$5")
		}
	case FILE:
		{
			re := regexp.MustCompile("(bs|ss)(_|)(\\d{4})(\\d{2}|)(\\d{2})(.sql|).zip")
			return re.ReplaceAllString(str, "$1$3$5")
		}
	}
	return ""
}

func matchStr(str string, tp string) (bool, error) {
	switch tp {
	case DB:
		{
			return regexp.MatchString("(bs|ss)(_|)(\\d{4})(\\d{2}|)(\\d{2})", str)
		}
	case FILE:
		{
			return regexp.MatchString("(bs|ss)(_|)(\\d{4})(\\d{2}|)(\\d{2})(.sql|).zip", str)
		}
	}
	return false, nil
}

func getDBs(host, pass, port string) (dbs map[string]string, err error) {
	tmpDbs := make([]string, 0)
	conn, err := sql.Open("mysql", "root:"+pass+"@tcp("+host+":"+port+")/sys")
	defer conn.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return dbs, err
	}
	conn.SetConnMaxLifetime(time.Minute * 5)
	conn.SetMaxIdleConns(50)
	conn.SetMaxOpenConns(50)
	rows, err := conn.Query("SHOW DATABASES")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return dbs, err
	}
	for rows.Next() {
		var db sql.NullString
		if err := rows.Scan(&db); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return dbs, err
		}
		tmpDbs = append(tmpDbs, db.String)
	}
	rows.Close()
	if rows.Err() != nil {
		return dbs, rows.Err()
	}
	dbs, err = filterStrings(tmpDbs, DB)
	if err != nil {
		return dbs, err
	}
	return dbs, nil
}

func createDB(host string, port string, pass string, dbName string) error {
	db, err := sql.Open("mysql", "root:"+pass+"@tcp("+host+":"+port+")/")
	defer db.Close()
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec("CREATE DATABASE " + dbName)
	if err != nil {
		return err
	}

	return nil
}
func importDB(tmp *os.File, pass string, host string, port string, dbName string) (err error) {
	var cmdOut []byte

	var status bool
	go func(status *bool) {
		var size float64
		for {
			if *status {
				break
			}
			db, err := sql.Open("mysql", "root:"+pass+"@tcp("+host+":"+port+")/"+dbName)
			if err != nil {
				fmt.Println("err0", err)
			}

			row := db.QueryRow(dbSize, dbName)

			err = row.Scan(&size)
			if err != nil && err != sql.ErrNoRows {
				fmt.Println("err1", err)
			}

			fmt.Print(fmt.Sprintf("%-.2f %s\n", size, "MB loaded"))

			db.Close()

			time.Sleep(20 * time.Second)
		}
	}(&status)
	if cmdOut, err = exec.Command("/bin/bash", "-c", "mysql -u root --password="+pass+" -h "+host+" -P "+port+" "+dbName+" < "+tmp.Name()).Output(); err != nil {
		status = true
		return err
	}
	status = true

	fmt.Println(string(cmdOut))

	return nil
}

func getFiles(folder string) (files map[string]string, err error) {
	tmpFiles := make([]string, 0)
	err = filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
		tmpFiles = append(tmpFiles, info.Name())
		return nil
	})
	if err != nil {
		return files, err
	}

	files, err = filterStrings(tmpFiles, FILE)
	if err != nil {
		return files, err
	}

	return files, nil
}

func doEvery(d time.Duration, f func(time.Time)) {
	for x := range time.Tick(d) {
		f(x)
	}
}

func different(files map[string]string, dbs map[string]string) (file string, db string) {
	for file, ori := range files {
		var ban bool
		for db := range dbs {
			if file == db {
				ban = true
			}
		}
		if !ban {
			return ori, file
		}
	}
	return "", ""
}

func unzip(file string, passwd string) (os.File, error) {
	var tmp *os.File
	var err error
	r, err := zip.OpenReader(file)
	if err != nil {
		return *tmp, err
	}
	defer r.Close()

	for _, f := range r.File {
		if f.IsEncrypted() {
			f.SetPassword(passwd)
		}

		r, err := f.Open()
		if err != nil {
			return *tmp, err
		}

		buf, err := ioutil.ReadAll(r)
		if err != nil {
			return *tmp, err
		}
		defer r.Close()

		t := os.TempDir()

		tmp, err = ioutil.TempFile(t, "tmp")
		if err != nil {
			return *tmp, err
		}

		err = ioutil.WriteFile(tmp.Name(), buf, 0644)
		if err != nil {
			return *tmp, err
		}

		fmt.Printf("Size of %v: %v MB\n", tmp.Name(), len(buf)/1024/1024)
	}

	return *tmp, nil
}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
