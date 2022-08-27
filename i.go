package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

var (
	// the address to listen on
	address = "127.0.0.1:9005"
	// the directory to save the images in
	root = "/var/www/i.fourtf.com/"
	// the root of the link that will be generated
	webRoot = "https://i.fourtf.com/"

	// maximum age for the files
	// the program will delete the files older than maxAge every 2 hours
	maxAge = time.Hour * 24 * 365
	// files to be ignored when deleting old files
	deleteIgnoreRegexp = regexp.MustCompile("index\\.html|favicon\\.ico")

	// length of the random filename
	randomAdjectivesCount = 2
	adjectives            = make([]string, 0)
	filetypes             = make(map[string]string)
)

func main() {
	rand.Seed(time.Now().UnixNano())

	b, err := ioutil.ReadFile("./filetypes.json")

	if err == nil {
		data := make(map[string][]string)

		if err = json.Unmarshal(b, &data); err != nil {
			fmt.Println(err)
		} else {
			for val, keys := range data {
				for _, key := range keys {
					filetypes["."+strings.TrimLeft(key, ".")] = val
				}
			}
		}
	}

	fmt.Println(filetypes)

	file, err := os.Open("./adjectives1.txt")

	if err != nil {
		panic(err)
	}

	r := bufio.NewReader(file)

	for {
		line, _, err := r.ReadLine()

		if err != nil {
			break
		}

		adjectives = append(adjectives, string(line))
	}

	// uncomment to collect old files
	// go func() {
	// 	for {
	// 		<-time.After(time.Hour * 2)
	// 		collectGarbage()
	// 	}
	// }()

	// create server with read and write timeouts and the desired address
	server := &http.Server{
		ReadTimeout:  time.Minute,
		WriteTimeout: time.Minute,
		Addr:         address,
	}

	// open http server
	http.HandleFunc("/", handleUpload)
	server.ListenAndServe()
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	infile, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error parsing uploaded file: "+err.Error(), http.StatusBadRequest)
		return
	}

	defer infile.Close()

	filename := header.Filename
	var ext string

	// get extension from file name
	index := strings.LastIndex(filename, ".")

	if index == -1 {
		ext = ""
	} else {
		ext = filename[index:]
		filename = filename[:index]
	}

	lastWord := "File"

	fmt.Println(ext)

	if val, ok := filetypes[ext]; ok {
		lastWord = strings.Title(val)
	}

	var savePath string
	var link string

	// find a random filename that doesn't exist already
	for i := 0; i < 100; i++ {
		random := ""

		for j := 0; j < randomAdjectivesCount; j++ {
			random += strings.TrimSpace(strings.Title(adjectives[rand.Intn(len(adjectives))]))
		}

		random += lastWord

		// fuck with link
		savePath = root + random + ext
		link = webRoot + random + ext

		if _, err := os.Stat(savePath); os.IsNotExist(err) {
			break
		}
	}

	// save the file
	outfile, err := os.Create(savePath)
	if err != nil {
		http.Error(w, "error while saving file: "+err.Error(), http.StatusBadRequest)
		return
	}

	_, err = io.Copy(outfile, infile)
	if err != nil {
		http.Error(w, "error while saving file: "+err.Error(), http.StatusBadRequest)
		return
	}
	outfile.Close()

	// return the link as the http body
	w.Write([]byte(link))

	// do this or it doesn't work
	io.Copy(ioutil.Discard, r.Body)
}

func collectGarbage() {
	files, err := ioutil.ReadDir(root)

	if err != nil {
		return
	}

	for _, file := range files {
		fname := file.Name()

		if file.IsDir() || deleteIgnoreRegexp.MatchString(fname) {
			continue
		}

		if time.Since(file.ModTime()) > maxAge {
			err := os.Remove(root + fname)

			if err != nil {
				fmt.Println(err)
				continue
			}

			fmt.Printf("Removed %s \n", fname)
		}
	}
}
