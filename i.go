package main

import (
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
	// the password to upload files
	key = "password"
	// the directory to save the images in
	root = "/var/www/i.fourtf.com/"
	// the root of the link that will be generated
	webRoot = "http://i.fourtf.com/"

	// maximum age for the files
	// the program will delete the files older than maxAge every 2 hours
	maxAge = time.Hour * 24 * 30
	// files to be ignored when delting old files
	deleteIgnoreRegexp = regexp.MustCompile("index\\.html|favicon\\.ico")

	// length of the random filename
	randomFilenameLength = 8
	// characters to use for the random filename
	randomFilenameLetters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
)

func main() {
	rand.Seed(time.Now().UnixNano())

	// collect old files every 2 hours
	go func() {
		for {
			<-time.After(time.Hour * 2)
			collectGarbage()
		}
	}()

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
	infile, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error parsing uploaded file: "+err.Error(), http.StatusBadRequest)
		return
	}

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

	var savePath string
	var link string

	// find a random filename that doesn't exist already
	for i := 0; i < 100; i++ {
		b := make([]rune, randomFilenameLength)
		for i := range b {
			b[i] = randomFilenameLetters[rand.Intn(len(randomFilenameLetters))]
		}

		random := string(b)

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
	http.Error(w, link, 200)

	// do this or it doesn't work
	io.Copy(ioutil.Discard, r.Body)
}

func collectGarbage() {
	files, err := ioutil.ReadDir(root)

	if err != nil {
		return
	}

	for _, file := range files {
		if file.IsDir() || deleteIgnoreRegexp.MatchString(file.Name()) {
			continue
		}

		if time.Now().Sub(file.ModTime()) > maxAge {
			os.Remove(file.Name())
		}
	}
}
