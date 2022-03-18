package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
)

var (
	c           = Results{Date: current_date_time(), Result: []Result{}, Count: 0}
	p           = Paths{Paths: []string{}}
	d           = Directories{Directories: []Results{}, Count: 0}
	r           = Result{}
	f           = File{}
	directory   *string
	directories *bool
	exclude     *string
	help        *bool
	stdout      *bool
	arrayList   []string

	list_of_files_to_esclude = []string{
		".DS_Store",
		".git",
		".gitignore",
		".idea",
		".vscode",
		"##Attributes.ini",
		".log",
	}
)

type Directories struct {
	Directories []Results
	Count       int `json:"count"`
}

type Paths struct {
	Paths []string
}

type Result struct {
	Path     string `json:"path"`
	Quantity int    `json:"quantity"`
	Files    []File `json:"files"`
}

type File struct {
	File string `json:"file"`
	Hash string `json:"hash"`
}

type Results struct {
	Date   string   `json:"date"`
	Count  int      `json:"count"`
	Result []Result `json:"result"`
}

// exclude file from list
func exclude_file(file string) bool {
	for _, v := range list_of_files_to_esclude {
		if strings.Contains(file, v) {
			return true
		}
	}
	return false
}

// list all items in a folder and its subfolders
func search_all_files(folder string, paths *Paths) error {
	log.Infof("Searching all files in %s", folder)
	err := filepath.Walk(folder,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && !exclude_file(path) {
				paths.Paths = append(paths.Paths, path)
			}
			return nil
		})
	sort.Strings(paths.Paths)
	return err
}

// generate md5 hash from file
func generate_md5(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// remeve speciphic character from string
func remove_spec_char(str string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return -1
	}, str)
}

// current date time
func current_date_time() string {
	time := time.Now()
	return fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", time.Year(), time.Month(), time.Day(), time.Hour(), time.Minute(), time.Second())
}

// get filename for path
func get_filename(path string) string {
	return filepath.Base(path)
}

// get path from filename
func get_path(path string) string {
	return filepath.Dir(path)
}

// compare path and if the path is same store filename in list
func compute_results(p *Paths, f *File, r *Result, c *Results) {
	log.Infof("compute_results")
	paths := p.Paths
	var t_path string
	for i := range paths {
		filename := get_filename(string(paths[i]))
		path := get_path(string(paths[i]))
		hash, err := generate_md5(string(paths[i]))
		if err != nil {
			log.Errorf("Error generating md5 hash for %s", string(paths[i]))
		}
		if t_path == "" {
			t_path = path
			c.Count++
			r.Files = append(r.Files, File{File: filename, Hash: hash})
		} else if path == t_path {
			c.Count++
			r.Files = append(r.Files, File{File: filename, Hash: hash})
		} else {
			r.Path = t_path
			t_path = path
			r.Quantity = len(r.Files)
			c.Result = append(c.Result, *r)
			r.Files = []File{}
		}
	}
	c.Result = append(c.Result, *r)
}

// save the result to disk
func save_result(c *Results) {
	log.Infof("Saving result to disk")
	date := strings.Replace(c.Date, " ", "_", -1)
	date = strings.Replace(date, ":", ".", -1)
	to_write := fmt.Sprintf("report_%s.json", date)
	file, err := os.Create(to_write)
	if err != nil {
		log.Errorf("Error creating file %s", to_write)
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encoder.Encode(c)
}

// options
func init() {
	r.Files = []File{}
	directory = flag.String("dir", "", "directory to search")
	directories = flag.Bool("dirs", false, "folders to search separated by comma")
	exclude = flag.String("exclude", "", "folders to exclude")
	help = flag.Bool("help", false, "print help")
	stdout = flag.Bool("stdout", false, "print result to stdout")
	arrayList = flag.Args()
	flag.Parse()
}

func task() {
	compute_results(&p, &f, &r, &c)
	if *stdout == true {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(c)
	} else {
		save_result(&c)
	}
	os.Exit(0)
}

func main() {

	if *help == true {
		flag.PrintDefaults()
		os.Exit(0)
	} else if *directories == true && arrayList != nil {
		log.Infof("Searching in directories %s", *directories)
		for i := range arrayList {
			folder := strings.TrimSpace(arrayList[i])
			err := search_all_files(folder, &p)
			if err != nil {
				log.Errorf("Error searching in %s", arrayList[i])
				os.Exit(1)
			}
			d.Count++
			d.Directories = append(d.Directories, c)
			c = Results{Date: current_date_time(), Result: []Result{}, Count: 0}
		}
		task()
		os.Exit(0)
	} else if *directory != ""   {
		err := search_all_files(*directory, &p)
		if err != nil {
			log.Infof("Error searching all files in %s", *directory)
			os.Exit(1)
		}
		task()
	}
}
