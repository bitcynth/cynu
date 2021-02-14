package main

import "io"

type uploadRequest struct {
	UploadKey      string
	InputFile      io.Reader
	Filename       string
	RandomFilename *bool
	ContentType    string
}

type uploadResult struct {
	FileURL     string
	ContentType string
}

type uploadResultJSON struct {
	FileURL string `json:"file_url"`
}
