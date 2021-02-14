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

var config *configuration

var oldConfig configuration
var newConfig configuration

var defaultConfig = configuration{
	ListenAddr: ":8080",
	UploadPath: "./data/",
	UploadURL:  "http://localhost:8081/",
}

// just because env vars and other ways to provide keys
var uploadKeys *map[string]bool

var oldUploadKeys map[string]bool
var newUploadKeys map[string]bool

// loadConfig tries to load the config file and then apply flags
// loadConfig doesn't fail if it can't read the config file.
func loadConfig() {
	oldUploadKeys = newUploadKeys
	uploadKeys = &oldUploadKeys
	oldConfig = newConfig
	config = &oldConfig
	newConfig = defaultConfig

	// populate with values from config file
	configBytes, err := ioutil.ReadFile(*configPath)
	if os.IsNotExist(err) {
		// don't continue if file doesn't exist
	} else if err != nil {
		log.Printf("error loading config file: %v", err)
	} else {
		if err := json.Unmarshal(configBytes, &newConfig); err != nil {
			log.Printf("error parsing config file: %v", err)
		}
	}

	// apply values from flags
	if *listenAddr != "" {
		newConfig.ListenAddr = *listenAddr
	}
	if *uploadPath != "" {
		newConfig.UploadPath = *uploadPath
	}
	if *uploadURL != "" {
		newConfig.UploadURL = *uploadURL
	}
	if *remoteAddrHeader != "" {
		newConfig.RemoteAddrHeader = *remoteAddrHeader
	}

	// just making config.Keys is initialized
	if newConfig.Keys == nil {
		newConfig.Keys = make(map[string]string)
	}

	newUploadKeys = make(map[string]bool)

	uploadKeyEnv := os.Getenv("UPLOAD_KEY")
	if uploadKeyEnv != "" {
		envKeys := strings.Split(uploadKeyEnv, ",")
		for _, key := range envKeys {
			newUploadKeys[key] = true
		}
	}

	for key := range newConfig.Keys {
		newUploadKeys[key] = true
	}

	uploadKeys = &newUploadKeys
	config = &newConfig
}
