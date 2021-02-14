package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

var listenAddr = flag.String("listen", "", "the address to listen on for HTTP (overrides config option)")
var uploadPath = flag.String("upload.path", "", "the place to place the uploaded files (overrides config option)")
var uploadURL = flag.String("upload.url", "", "the base url for the uploaded files (overrides config option)")
var remoteAddrHeader = flag.String("header.remoteaddr", "", "if set, uses this header to get the remote ip addr (overrides config option)")
var configPath = flag.String("config.path", "./config.json", "path to the config file")
var debugMode = flag.Bool("debug", false, "set to enable debug mode (more logging)")

func main() {
	flag.Parse()

	loadConfig()

	http.HandleFunc("/upload", handleUpload)

	// API compat modes
	http.HandleFunc("/compat/imgur/image", handleCompatImgurImage)

	go func() {
		err := http.ListenAndServe(config.ListenAddr, nil)
		if err != nil {
			log.Fatal(err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP)

	for {
		s := <-sig
		if s == syscall.SIGHUP {
			log.Print("received SIGHUP signal, reloading...")
			loadConfig()
		}
	}
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusNotFound) // don't acknowledge this path unless using correct method
		fmt.Fprint(w, "nope!\n")
		return
	}

	r.ParseMultipartForm(32 << 20)

	file, handler, err := r.FormFile("file")
	if err != nil {
		w.WriteHeader(errorStatusCode(errFailPostFile))
		fmt.Fprintf(w, "ERROR: %s\n", errorStatusText(errFailPostFile))
		log.Print("failed to get file from form: ", err)
		return
	}
	defer file.Close()

	useRandomFilename := r.FormValue("randomname") == "true"

	req := uploadRequest{
		UploadKey:      r.FormValue("key"),
		Filename:       handler.Filename,
		RandomFilename: &useRandomFilename,
		ContentType:    handler.Header.Get("Content-Type"),
		InputFile:      file,
	}

	result, err := uploadFile(req)
	if err != nil {
		log.Printf("error [%s]: %v", getRemoteAddr(r), err)
		w.WriteHeader(errorStatusCode(err))
		fmt.Fprintf(w, "ERROR: %s\n", errorStatusText(err))
	}

	res := &uploadResultJSON{
		FileURL: result.FileURL,
	}

	jsonData, err := json.Marshal(res)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		log.Print("failed to generate response json: ", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func uploadFile(req uploadRequest) (*uploadResult, error) {
	if !validateUploadKey(req.UploadKey) {
		log.Print("invalid upload key!")
		return nil, errInvalidUploadKey
	}

	if req.ContentType == "" && req.Filename != "" {
		fnExt := filepath.Ext(req.Filename)
		if fnExt != "" {
			req.ContentType = mime.TypeByExtension(fnExt)
			if *debugMode {
				log.Printf("selecting content type \"%s\" for file ext \"%s\"", req.ContentType, fnExt)
			}
		}
	}

	filename := req.Filename
	if *req.RandomFilename {
		randName, err := randomFilename()
		if err != nil {
			log.Print("error generating random filename: ", err)
			return nil, errFailedRandomName
		}
		fnExt := filepath.Ext(req.Filename)
		if fnExt == "" {
			fnExt, _ = getExtByMimetype(req.ContentType)
			if fnExt == "" {
				fnExt = ".dat"
			}
		}
		filename = randName + fnExt
	}

	f, err := os.OpenFile(filepath.Join(config.UploadPath, filename), os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		log.Print("error opening output file: ", err)
		return nil, errFailedOutputFile
	}
	defer f.Close()

	n, err := io.Copy(f, req.InputFile)
	if err != nil {
		log.Printf("got error at %d bytes written: %v", n, err)
		return nil, errFailedWriteFile
	}

	result := &uploadResult{
		FileURL:     config.UploadURL + filename,
		ContentType: req.ContentType,
	}

	return result, nil
}

func validateUploadKey(key string) bool {
	keys := *uploadKeys
	v, ok := keys[key]
	if ok && v {
		return true
	}
	return false
}

func getRemoteAddr(r *http.Request) string {
	if config.RemoteAddrHeader != "" {
		return r.Header.Get(config.RemoteAddrHeader)
	}
	return r.RemoteAddr
}
