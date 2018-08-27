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
		dbs, err := getDBs(host, dbPass, port)
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
			dbs, err = getDBs(host, dbPass, port)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			// 		2.5 Entonces revisamos que tenga el nombre correspondiente
			if file, db := different(files, dbs); file != "" {
				fmt.Println("There were changes!!", file)
				//			2.7 Entonces descomprimimos el zip en una carpeta temporal
				fmt.Println("Unzipping file...")
				tmpFile, err := unzip(folder+"/"+file, zipPass)
				if err != nil {
					fmt.Fprintf(os.Stderr, "error: %v\n", err)
					os.Exit(1)
				}
				fmt.Println("tmpFile", tmpFile)
				// 2.8 Creamos la base de datos
				fmt.Println("Creating db " + db + "...")
				err = createDB(host, port, dbPass, db)
				if err != nil {
					fmt.Fprintf(os.Stderr, "error: %v\n", err)
					os.Exit(1)
				}

				// actualizamos dbs
				fmt.Println("Importing db " + db + "...")
				dbs, err = getDBs(host, dbPass, port)
				if err != nil {
					fmt.Fprintf(os.Stderr, "error: %v\n", err)
					os.Exit(1)
				}
				// 2.8 Iniciamos la importacion (tratando de mostrar un proceso) (quiza leyendo el tamaño actual de la BD)
				err = importDB(&tmpFile, dbPass, host, port, db)
				if err != nil {
					fmt.Fprintf(os.Stderr, "error: %v\n", err)
					os.Exit(1)
				}
				// 2.9 Mostramos que la bd se importo correctamente
				fmt.Println("Imported " + db + " succesfully!")
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

func getDBs(host, pass, port string) (dbs map[string]string, err error) {
	tmpDbs := make([]string, 0)
	conn, err := sql.Open("mysql", "root:"+pass+"@tcp("+host+":"+port+")/sys")
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
	dbs, err = filterStrings(tmpDbs, DB)
	if err != nil {
		return dbs, err
	}
	return dbs, nil
}

func createDB(host string, port string, pass string, dbName string) error {
	db, err := sql.Open("mysql", "root:"+pass+"@tcp("+host+":"+port+")/")
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
func importDB(tmp *os.File, pass string, host string, port string, dbName string) error {
	// mysql -u root -p DB_NAME < PATH_FILE

	// thepath, err := filepath.Abs(filepath.Dir(tmp.Name()))
	// if err != nil {
	// 	return err
	// }

	// cmd := "mysql -u root --password=" + pass + " --host=" + host + " --port=" + port + " " + dbName + " -e " + tmp.Name()

	// exec.Command("mysql", "-u", "root", "-p{db password}", "{db name}",
	// 	"-e", "source {file abs path}")

	// c := exec.Command("mysql", "-u", "root", "--password=", pass, "--host=", host, "--port=", port, dbName, "-e", tmp.Name())
	// c.Run()
	// out, err := c.Output()
	// if err != nil {
	// 	fmt.Println(string(out))
	// 	return err
	// }
	// fmt.Println(string(out))

	tmp2, err := ioutil.TempFile(os.TempDir(), "tmp")
	fmt.Println("tmpFile:=" + tmp2.Name())
	if err != nil {
		return err
	}
	str := "#!/bin/sh\n" + "mysql" + " -u " + "root" + " --password=" + pass + " -h " + host + " -P " + port + " " + dbName + " < " + tmp.Name()
	fmt.Println("Command used := ", str)
	tmp2.Write([]byte(str))
	err = os.Chmod(tmp2.Name(), 0777)
	if err != nil {
		return err
	}

	// args := []string{"-u", "root", "--password=", pass, "--host=", host, "--port=", port, dbName, "-e", tmp.Name()}
	// args := []string{"hi"}
	var cmdOut []byte
	// var err error
	if cmdOut, err = exec.Command("/bin/sh", tmp2.Name()).Output(); err != nil {
		fmt.Fprintln(os.Stderr, "There was an error running git rev-parse command: ", err)
		return err
	}

	fmt.Println("cmdOut", string(cmdOut))

	// c := exec.Command("/bin/sh", tmp2.Name())
	// err := c.Run()
	// if err != nil {
	// 	fmt.Println(err)
	// 	return err
	// }
	// err = c.Wait()
	// if err != nil {
	// 	out, err := c.Output()
	// 	fmt.Println("2" + string(out))
	// 	return err
	// }
	// out, err := c.Output()
	// if err != nil {
	// 	fmt.Println("3" + string(out))
	// 	return err
	// }
	// fmt.Println("4" + string(out))

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

		fmt.Printf("Size of %v: %v byte(s)\n", tmp.Name(), len(buf))
	}

	return *tmp, nil
}
