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
	"log"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/yeka/zip"

	_ "github.com/go-sql-driver/mysql"
	"github.com/spf13/cobra"
)

const DB = "db"
const FILE = "file"

// mysqlCmd represents the mysql command
var mysqlCmd = &cobra.Command{
	Use:   "mysql",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		host, err := cmd.Flags().GetString("databaseHost")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		dbPass, err := cmd.Flags().GetString("databasePassword")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		zipPass, err := cmd.Flags().GetString("filePassword")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		folder, err := cmd.Flags().GetString("folder")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		port, err := cmd.Flags().GetString("databasePort")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		// 1.- leer todos los archivos en la carpeta
		files, err := getFiles(folder)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(files)

		// 2.- leer todas las bases de datos deacuerdo a los parametros de entrada o default
		dbs, err := getDBs(host, port)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(dbs)
		// 2.- volver a leer los archivos y si alguno de ellos se agrego
		doEvery(5*time.Second, func(t time.Time) {
			files, err = getFiles(folder)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			dbs, err = getDBs(host, port)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			// 		2.5 Entonces revisamos que tenga el nombre correspondiente
			if file, db := different(files, dbs); file != "" {
				fmt.Println("There were changes!!", file)
				//			2.7 Entonces descomprimimos el zip en una carpeta temporal
				err := Unzip(folder+"/"+file, zipPass)
				if err != nil {
					os.Exit(1)
				}
				// 2.8 Creamos la base de datos
				createDB(host, port, dbPass, db)

				// actualizamos dbs
				dbs, err = getDBs(host, port)
				if err != nil {
					fmt.Fprintf(os.Stderr, "error: %v\n", err)
					os.Exit(1)
				}

				// 2.8 Iniciamos la importacion (tratando de mostrar un proceso) (quiza leyendo el tamaño actual de la BD)
				// 2.9 Mostramos que la bd se importo correctamente
			}
		})

		fmt.Sprintln(host)
		fmt.Sprintln(zipPass)
		fmt.Sprintln(folder)
		fmt.Println("took", time.Since(start))
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

func getDBs(host, port string) (dbs map[string]string, err error) {
	tmpDbs := make([]string, 0)
	conn, err := sql.Open("mysql", "root@tcp("+host+":"+port+")/information_schema")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return dbs, err
	}
	conn.SetConnMaxLifetime(time.Minute * 5)
	conn.SetMaxIdleConns(50)
	conn.SetMaxOpenConns(50)
	rows, err := conn.Query("SELECT schema_name FROM SCHEMATA")
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
	dbs, err = filterStrings(tmpDbs, DB)
	if err != nil {
		return dbs, err
	}
	return dbs, nil
}

func createDB(host string, port string, pass string, dbName string) error {
	db, err := sql.Open("mysql", "root@tcp("+host+":"+port+")/")
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

func Unzip(file string, passwd string) error {
	r, err := zip.OpenReader(file)
	if err != nil {
		log.Fatal(err)
	}
	defer r.Close()

	for _, f := range r.File {
		if f.IsEncrypted() {
			f.SetPassword(passwd)
		}

		r, err := f.Open()
		if err != nil {
			return err
		}

		buf, err := ioutil.ReadAll(r)
		if err != nil {
			log.Fatal(err)
		}
		defer r.Close()

		tmp, err := ioutil.TempFile(os.TempDir(), "tmp")

		err = ioutil.WriteFile(tmp.Name(), buf, 0644)

		fmt.Printf("Size of %v: %v byte(s)\n", tmp.Name(), len(buf))
	}

	return nil
}
