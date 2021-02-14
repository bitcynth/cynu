package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

type resultJSON struct {
	FileURL string `json:"file_url"`
}

var listenAddr = flag.String("listen", "", "the address to listen on for HTTP (overrides config option)")
var uploadPath = flag.String("upload.path", "", "the place to place the uploaded files (overrides config option)")
var uploadURL = flag.String("upload.url", "", "the base url for the uploaded files (overrides config option)")
var configPath = flag.String("config.path", "./config.json", "path to the config file")

func main() {
	flag.Parse()

	loadConfig()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP)

	go func() {
		for {
			s := <-sig
			if s == syscall.SIGHUP {
				log.Print("received SIGHUP signal, reloading...")
				loadConfig()
			}
		}
	}()

	http.HandleFunc("/upload", handleUpload)

	err := http.ListenAndServe(config.ListenAddr, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(404) // don't acknowledge this path unless using correct method
		fmt.Fprint(w, "nope!")
		return
	}

	r.ParseMultipartForm(32 << 20)

	key := r.FormValue("key")
	if !validateUploadKey(key) {
		w.WriteHeader(401)
		fmt.Fprint(w, "ERROR: invalid upload key!")
		log.Println("Invalid upload key!")
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		w.WriteHeader(500)
		log.Print("failed to get file from form: ", err)
		fmt.Fprint(w, "ERROR: failed to get file!")
		return
	}

	defer file.Close()

	filename := handler.Filename
	useRandomName := r.FormValue("randomname")
	if useRandomName == "true" {
		randstr, err := randomString()
		filename = randstr + filepath.Ext(handler.Filename)
		if err != nil {
			w.WriteHeader(500)
			log.Print(err)
			fmt.Fprint(w, "ERROR: failed generate filename!")
			return
		}
	}

	f, err := os.OpenFile(config.UploadPath+filename, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		w.WriteHeader(500)
		log.Print(err)
		fmt.Fprint(w, "ERROR: failed to open file")
		return
	}
	defer f.Close()

	io.Copy(f, file)

	res := &resultJSON{
		FileURL: config.UploadURL + filename,
	}

	jsonData, err := json.Marshal(res)
	if err != nil {
		w.WriteHeader(500)
		log.Print(err)
		fmt.Fprint(w, "ERROR: failed to create json")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func validateUploadKey(key string) bool {
	v, ok := uploadKeys[key]
	if ok && v {
		return true
	}
	return false
}

func randomString() (string, error) {
	length := 8
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
