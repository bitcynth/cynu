package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

// named configuration to not conflict with the config variable
type configuration struct {
	ListenAddr       string            `json:"listen_address"`
	UploadPath       string            `json:"upload_path"`
	UploadURL        string            `json:"upload_url"`
	RemoteAddrHeader string            `json:"remote_addr_header"`
	Keys             map[string]string `json:"upload_keys"` // {"key here": "comment here"}
}

var config configuration

var defaultConfig = configuration{
	ListenAddr: ":8080",
	UploadPath: "./data/",
	UploadURL:  "http://localhost:8081/",
}

// just because env vars and other ways to provide keys
var uploadKeys map[string]bool

// loadConfig tries to load the config file and then apply flags
// loadConfig doesn't fail if it can't read the config file.
func loadConfig() {
	config = defaultConfig

	// populate with values from config file
	configBytes, err := ioutil.ReadFile(*configPath)
	if os.IsNotExist(err) {
		// don't continue if file doesn't exist
	} else if err != nil {
		log.Printf("error loading config file: %v", err)
	} else {
		if err := json.Unmarshal(configBytes, &config); err != nil {
			log.Printf("error parsing config file: %v", err)
		}
	}

	// apply values from flags
	if *listenAddr != "" {
		config.ListenAddr = *listenAddr
	}
	if *uploadPath != "" {
		config.UploadPath = *uploadPath
	}
	if *uploadURL != "" {
		config.UploadURL = *uploadURL
	}
	if *remoteAddrHeader != "" {
		config.RemoteAddrHeader = *remoteAddrHeader
	}

	// just making config.Keys is initialized
	if config.Keys == nil {
		config.Keys = make(map[string]string)
	}

	uploadKeys = make(map[string]bool)

	uploadKeyEnv := os.Getenv("UPLOAD_KEY")
	if uploadKeyEnv != "" {
		envKeys := strings.Split(uploadKeyEnv, ",")
		for _, key := range envKeys {
			uploadKeys[key] = true
		}
	}

	for key := range config.Keys {
		uploadKeys[key] = true
	}
}
