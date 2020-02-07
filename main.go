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
	"path/filepath"
)

var listenAddr = flag.String("listen", ":8080", "the address to listen on for HTTP")
var uploadPath = flag.String("upload.path", "./data/", "the place to place the uploaded files")
var uploadURL = flag.String("upload.url", "http://localhost:8081/", "the base url for the uploaded files")

type resultJSON struct {
	FileURL string `json:"file_url"`
}

func main() {
	flag.Parse()

	http.HandleFunc("/upload", handleUpload)

	err := http.ListenAndServe(*listenAddr, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
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
		log.Print(err)
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

	f, err := os.OpenFile(*uploadPath+filename, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		w.WriteHeader(500)
		log.Print(err)
		fmt.Fprint(w, "ERROR: failed to open file")
		return
	}
	defer f.Close()

	io.Copy(f, file)

	res := &resultJSON{
		FileURL: *uploadURL + filename,
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

// TODO: implement multi-key system
func validateUploadKey(key string) bool {
	if key == os.Getenv("UPLOAD_KEY") {
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
