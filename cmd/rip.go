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
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/josecordaz/run/ripsrc"
	"github.com/pkg/profile"
	"github.com/spf13/cobra"
)

// ripCmd represents the rip command
var ripCmd = &cobra.Command{
	Use:   "rip",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		errors := make(chan error, 1)
		go func() {
			for err := range errors {
				cancel()
				fmt.Println(err)
				os.Exit(1)
			}
		}()
		var dir string
		p := "mem"
		dir, _ = ioutil.TempDir("", "profile")
		defer func() {
			fn := filepath.Join(dir, p+".pprof")
			abs, _ := filepath.Abs(os.Args[0])
			fmt.Printf("to view profile, run `go tool pprof --pdf %s %s`\n", abs, fn)
		}()
		defer profile.Start(profile.MemProfile, profile.ProfilePath(dir), profile.Quiet).Stop()
		go func() {
			var s runtime.MemStats
			var c int
			os.MkdirAll("profile", 0755)
			var menor uint64
			dump := func() {
				c++
				f, _ := os.Create(fmt.Sprintf("profile/profile.%d.pb.gz", c))
				defer f.Close()
				// debug.FreeOSMemory()
				// runtime.GC()
				runtime.ReadMemStats(&s)
				fmt.Println(strings.Repeat("-", 120))
				fmt.Println(strings.Repeat("-", 120))
				fmt.Printf("fragment %d MB\n", (s.HeapInuse-s.HeapAlloc)/1024/1024)
				fmt.Printf("alive %d MB\n", (s.Mallocs-s.Frees)/1024/1024)
				current := (s.HeapAlloc) / 1024 / 1024
				fmt.Printf("alloc %d MB\n", current)
				if current < menor {
					fmt.Printf("FINALLY DID IT !!!! %d %d \n", current, menor)
					os.Exit(1)
				}
				menor = current
				fmt.Println(strings.Repeat("-", 120))
				fmt.Println(strings.Repeat("-", 120))
				// fmt.Println("Alloc       : ", s.Alloc)
				// fmt.Println("Total Alloc : ", s.TotalAlloc)
				pprof.WriteHeapProfile(f)
			}
			for {
				select {
				case <-time.After(1 * time.Second):
					dump()
				case <-ctx.Done():
					dump()
				}
			}
		}()
		var filter *ripsrc.Filter
		var count int
		started := time.Now()
		Rip(ctx, args[0], errors, filter)
		fmt.Printf("finished processing %d entries from %d directories in %v\n", count, len(args), time.Since(started))
	},
}

func init() {
	rootCmd.AddCommand(ripCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// ripCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// ripCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

type CommitStatus string

func (s CommitStatus) String() string {
	return string(s)
}

const (
	// GitFileCommitStatusAdded is the added status
	GitFileCommitStatusAdded = CommitStatus("added")
	// GitFileCommitStatusModified is the modified status
	GitFileCommitStatusModified = CommitStatus("modified")
	// GitFileCommitStatusRemoved is the removed status
	GitFileCommitStatusRemoved = CommitStatus("removed")
)

type Commit struct {
	Dir            string
	SHA            string
	AuthorEmail    string
	CommitterEmail string
	Files          map[string]*CommitFile
	Date           time.Time
	Ordinal        int64
	Message        string
	Parent         *string
	Signed         bool
	callback       Callback
}

type CommitFile struct {
	Filename    string
	Status      CommitStatus
	Renamed     bool
	RenamedFrom string
	RenamedTo   string
	Additions   int
	Deletions   int
	Binary      bool
}

type BlameLine struct {
	Name    string
	Email   string
	Date    time.Time
	Comment bool
	Code    bool
	Blank   bool

	// private, only used internally
	line *string
}

type BlameResult struct {
	Commit             *Commit
	Language           string
	Filename           string
	Lines              []*BlameLine
	Size               int64
	Loc                int64
	Sloc               int64
	Comments           int64
	Blanks             int64
	Complexity         int64
	WeightedComplexity float64
	Skipped            string
	License            *License
	Status             CommitStatus
}

var (
	tabSplitter        = regexp.MustCompile("\\t")
	spaceSplitter      = regexp.MustCompile("[ ]")
	whitespaceSplitter = regexp.MustCompile("\\s+")
)

type License struct {
	Name       string  `json:"license"`
	Confidence float32 `json:"confidence"`
}

// Callback for handling the commit job
type Callback func(err error, result *BlameResult, total int)

var (
	lend               = []byte("\n")
	commitPrefix       = []byte("commit ")
	authorPrefix       = []byte("Author: ")
	committerPrefix    = []byte("Committer: ")
	signedEmailPrefix  = []byte("Signed-Email: ")
	messagePrefix      = []byte("Message: ")
	parentPrefix       = []byte("Parent: ")
	emailRegex         = regexp.MustCompile("<(.*)>")
	emailBracketsRegex = regexp.MustCompile("^\\[(.*)\\]$")
	datePrefix         = []byte("Date: ")
	space              = []byte(" ")
	tab                = []byte("\t")
	rPrefix            = []byte("R")
	renameRe           = regexp.MustCompile("(.*)\\{(.*) => (.*)\\}(.*)")
)

func Rip(ctx context.Context, dir string, errors chan<- error, filter *ripsrc.Filter) {
	commits := make(chan *Commit, 1)
	gitdirs := [1]string{"/Users/developer/go/src/github.com/pinpt/worker"}
	fmt.Println("Starting...")
	after := make(chan bool, 1)
	go func() {
		for commit := range commits {
			fmt.Sprintln(commit)
		}
		after <- true
	}()
	for _, gitdir := range gitdirs {
		var sha string
		var limit int
		if err := streamCommits(ctx, gitdir, sha, limit, commits, errors); err != nil {
			errors <- fmt.Errorf("error streaming commits from git dir from %v. %v", gitdir, err)
			return
		}
	}
	close(commits)
	<-after
}

func getFilename(fn string) (string, string, bool) {
	if renameRe.MatchString(fn) {
		match := renameRe.FindStringSubmatch(fn)
		// use path.Join to remove empty directories and to correct join paths
		// must be path not filepath since it's always unix style in git and on windows
		// filepath will use \
		oldfn := path.Join(match[1], match[2], match[4])
		newfn := path.Join(match[1], match[3], match[4])
		return newfn, oldfn, true
	}
	// straight rename without parts
	s := strings.Split(fn, " => ")
	if len(s) > 1 {
		return s[1], s[0], true
	}
	return fn, fn, false
}

func parseDate(d string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, d)
	if err != nil {
		return time.Now(), fmt.Errorf("error parsing commit date `%v`. %v", d, err)
	}
	return t.UTC(), nil
}

func regSplit(text string, splitter *regexp.Regexp) []string {
	indexes := splitter.FindAllStringIndex(text, -1)
	laststart := 0
	result := make([]string, len(indexes)+1)
	for i, element := range indexes {
		result[i] = text[laststart:element[0]]
		laststart = element[1]
	}
	result[len(indexes)] = text[laststart:len(text)]
	return result
}

func parseEmail(email string) string {
	// strip out the angle brackets
	if emailRegex.MatchString(email) {
		m := emailRegex.FindStringSubmatch(email)
		s := m[1]
		// attempt to strip out square brackets if found
		if emailBracketsRegex.MatchString(s) {
			m = emailBracketsRegex.FindStringSubmatch(s)
			return m[1]
		}
		return s
	}
	return ""
}

func toCommitStatus(name []byte) CommitStatus {
	switch string(name) {
	case "A":
		{
			return GitFileCommitStatusAdded
		}
	case "D":
		{
			return GitFileCommitStatusRemoved
		}
	case "M", "R", "C", "MM":
		{
			return GitFileCommitStatusModified
		}
	}
	return GitFileCommitStatusModified
}

// buffer pool to reduce GC
var bufferPool = sync.Pool{
	// New is called when a new instance is needed
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

// getBuffer fetches a buffer from the pool
func getBuffer() *bytes.Buffer {
	return bufferPool.Get().(*bytes.Buffer)
}

// putBuffer returns a buffer to the pool
func putBuffer(buf *bytes.Buffer) {
	buf.Reset()
	bufferPool.Put(buf)
}

func streamCommits(ctx context.Context, dir string, sha string, limit int, commits chan<- *Commit, errors chan<- error) error {
	errout := getBuffer()
	defer putBuffer(errout)
	var cmd *exec.Cmd
	args := []string{
		"log",
		"--raw",
		"--reverse",
		"--numstat",
		"--pretty=format:commit %H%nCommitter: %ce%nAuthor: %ae%nSigned-Email: %GS%nDate: %aI%nParent: %P%nMessage: %s%n",
		"--no-merges",
	}
	// if provided, we need to start streaming after this commit forward
	if sha != "" {
		args = append(args, sha+"...")
	}
	// fmt.Println(args)
	cmd = exec.CommandContext(ctx, "git", args...)
	out, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	defer out.Close()
	cmd.Dir = dir
	cmd.Stderr = errout
	if err := cmd.Start(); err != nil {
		if strings.Contains(errout.String(), "does not have any commits yet") {
			return fmt.Errorf("no commits found in repo at %s", dir)
		}
		if strings.Contains(errout.String(), "Not a git repository") {
			return fmt.Errorf("not a valid git repo found in repo at %s", dir)
		}
		return fmt.Errorf("error running git log in dir %s, %v", dir, err)
	}
	done := make(chan bool)
	var total int
	go func() {
		defer func() {
			done <- true
		}()
		var commit *Commit
		r := bufio.NewReader(out)
		ordinal := time.Now().Unix()
		for {
			buf, err := r.ReadBytes(lend[0])
			if err != nil {
				if err == io.EOF {
					break
				}
				errors <- err
				return
			}

			// see if our context is cancelled
			select {
			case <-ctx.Done():
				return
			default:
				break
			}

			buf = buf[0 : len(buf)-1]
			if len(buf) == 0 {
				continue
			}
			if bytes.HasPrefix(buf, commitPrefix) {
				sha := string(buf[len(commitPrefix):])
				i := strings.Index(sha, " ")
				if i > 0 {
					// trim off stuff after the sha since we can get tag info there
					sha = sha[0:i]
				}
				// send the old commit and create a new one
				if commit != nil { // because we send when we detect the next commit
					commits <- commit
				}
				if limit > 0 && total >= limit {
					commit = nil
					break
				}
				commit = &Commit{
					Dir:     dir,
					SHA:     string(sha),
					Files:   make(map[string]*CommitFile, 0),
					Ordinal: ordinal,
				}
				ordinal++
				total++
				continue
			}
			if bytes.HasPrefix(buf, datePrefix) {
				d := bytes.TrimSpace(buf[len(datePrefix):])
				t, err := parseDate(string(d))
				if err != nil {
					errors <- fmt.Errorf("error parsing commit %s in %s. %v", commit.SHA, dir, err)
					return
				}
				commit.Date = t.UTC()
				continue
			}
			if bytes.HasPrefix(buf, authorPrefix) {
				commit.AuthorEmail = string(buf[len(authorPrefix):])
				continue
			}
			if bytes.HasPrefix(buf, committerPrefix) {
				commit.CommitterEmail = string(buf[len(committerPrefix):])
				continue
			}
			if bytes.HasPrefix(buf, signedEmailPrefix) {
				signedCommitLine := string(buf[len(signedEmailPrefix):])
				if signedCommitLine != "" {
					commit.Signed = true
					signedEmail := parseEmail(signedCommitLine)
					if signedEmail != "" {
						// if signed, mark it as such as use this as the preferred email
						commit.AuthorEmail = signedEmail
					}
				}
				continue
			}
			if bytes.HasPrefix(buf, messagePrefix) {
				commit.Message = string(buf[len(messagePrefix):])
				continue
			}
			if bytes.HasPrefix(buf, parentPrefix) {
				parent := string(buf[len(parentPrefix):])
				commit.Parent = &parent
				continue
			}
			if buf[0] == ':' {
				// :100644␠100644␠d1a02ae0...␠a452aaac...␠M␉·pandora/pom.xml
				tok1 := bytes.Split(buf, space)
				tok2 := bytes.Split(bytes.Join(tok1[4:], space), tab)
				action := tok2[0]
				paths := tok2[1:]
				if len(action) == 1 {
					fn := string(bytes.TrimLeft(paths[0], " "))
					commit.Files[fn] = &CommitFile{
						Filename: fn,
						Status:   toCommitStatus(action),
					}
				} else if bytes.HasPrefix(action, rPrefix) {
					fromFn := string(bytes.TrimLeft(paths[0], " "))
					toFn := string(bytes.TrimLeft(paths[1], " "))
					commit.Files[fromFn] = &CommitFile{
						Status:      GitFileCommitStatusRemoved,
						Filename:    fromFn,
						Renamed:     true,
						RenamedFrom: fromFn,
						RenamedTo:   toFn,
					}
					commit.Files[toFn] = &CommitFile{
						Status:      GitFileCommitStatusAdded,
						Filename:    toFn,
						Renamed:     true,
						RenamedFrom: fromFn,
						RenamedTo:   toFn,
					}
				} else {
					fn := string(bytes.TrimLeft(paths[0], " "))
					commit.Files[fn] = &CommitFile{
						Status:   toCommitStatus(action),
						Filename: fn,
					}
				}
				continue
			}
			tok := bytes.Split(buf, tab)
			// handle the file stats output
			if len(tok) == 3 {
				tok := regSplit(string(buf), tabSplitter)
				fn, oldfn, renamed := getFilename(tok[2])
				file := commit.Files[fn]
				if file == nil {
					panic("logic error. cannot determine commit file named: " + fn + " for commit " + sha + " in " + dir)
				}
				if renamed {
					file.RenamedFrom = oldfn
					file.Renamed = true
				}
				if tok[0] == "-" {
					file.Binary = true
				} else {
					adds, _ := strconv.ParseInt(tok[0], 10, 32)
					dels, _ := strconv.ParseInt(tok[1], 10, 32)
					file.Additions = int(adds)
					file.Deletions = int(dels)
				}
			}
		}
		if commit != nil {
			commits <- commit
		}
	}()
	<-done
	return nil
}
