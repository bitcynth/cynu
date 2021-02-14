package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/h2non/filetype"
)

type compatImgurImageResultJSON struct {
	Success bool `json:"success"`
	Status  int  `json:"status"` // HTTP status code(?)
	Data    struct {
		ID          *string     `json:"id"`
		Title       *string     `json:"title"`
		Description *string     `json:"description"`
		Datetime    *int64      `json:"datetime"`
		MIMEType    *string     `json:"type"`
		Antimated   *bool       `json:"animated"`
		Width       *int        `json:"width"`
		Height      *int        `json:"height"`
		Size        *int        `json:"size"`
		Views       *int        `json:"views"`
		Bandwidth   *int        `json:"bandwidth"`
		Vote        interface{} `json:"vote"` // no clue what datatype this actually is
		Favorite    *bool       `json:"favorite"`
		NSFW        *bool       `json:"nsfw"`
		Section     *string     `json:"section"`
		AccountURL  *string     `json:"account_url"`
		AccountID   *int        `json:"account_id"`
		IsAd        *bool       `json:"is_ad"`
		InMostViral *bool       `json:"in_most_viral"`
		Tags        *[]string   `json:"tags"`
		AdType      *int        `json:"ad_type"`
		AdURL       *string     `json:"ad_url"`
		InGallery   *bool       `json:"in_gallery"`
		DeleteHash  *string     `json:"deletehash"`
		Name        *string     `json:"name"`
		Link        *string     `json:"link"`
	} `json:"data"`
}

// for compat with `https://api.imgur.com/3/upload`
// currently ignoring the parameters: album, title, description, disable_audio
// based on latest docs as of 2021-02-14 at https://apidocs.imgur.com/#de179b6a-3eda-4406-a8d7-1fb06c17cb9c
func handleCompatImgurImage(w http.ResponseWriter, r *http.Request) {
	// helper function to respond with an error
	returnErr := func(e error) {
		res := compatImgurImageResultJSON{
			Status:  errorStatusCode(e),
			Success: false,
		}

		jsonData, err := json.Marshal(res)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			log.Print("failed to generate response json: ", err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(errorStatusCode(e))
		w.Write(jsonData)
	}

	if r.Method != "POST" {
		w.WriteHeader(http.StatusNotFound) // don't acknowledge this path unless using correct method
		fmt.Fprint(w, "nope!\n")
		return
	}

	useRandomFilename := true // always use random file names with imgur compat

	req := uploadRequest{
		RandomFilename: &useRandomFilename,
	}

	isMultipart := strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data;")
	if isMultipart {
		r.ParseMultipartForm(256 << 20)
	} else {
		r.ParseForm()
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		returnErr(errInvalidUploadKey)
		return
	}
	authHeaderParts := strings.SplitN(authHeader, " ", 2)
	if len(authHeaderParts) != 2 {
		returnErr(errInvalidUploadKey)
		return
	}
	if authHeaderParts[0] == "Bearer" {
		req.UploadKey = authHeaderParts[1]
	} else {
		// There is also a "Client-ID" auth type but that is for anonymous use, aka not supported.
		returnErr(errInvalidUploadKey)
		return
	}

	// fileType can be any of "file", "url", "base64"
	fileType := r.FormValue("type")
	if fileType == "" {
		if isMultipart {
			fileType = "file"
		} else {
			fileType = "url"
		}
	}

	if fileType == "file" {
		file, handler, err := r.FormFile("image")
		if err == http.ErrMissingFile {
			// if there was no image, try video
			file, handler, err = r.FormFile("video")
			// err might still be ErrMissingFile but we don't have anything more to try
		}
		if err != nil {
			returnErr(errFailPostFile)
			log.Print("failed to get file from form: ", err)
			return
		}
		defer file.Close()

		req.InputFile = file
		req.Filename = handler.Filename
		req.ContentType = handler.Header.Get("Content-Type")
	} else if fileType == "base64" {
		imageValue := r.FormValue("image")
		imageBytes, err := base64.StdEncoding.DecodeString(imageValue)
		if err != nil {
			returnErr(errFailPostFile)
			log.Print("failed to get file from form: ", err)
			return
		}
		req.InputFile = bytes.NewBuffer(imageBytes)
		t, _ := filetype.Match(imageBytes)
		if t == filetype.Unknown {
			returnErr(errFailPostFile)
			log.Print("no filename or mime type given")
			return
		}
		req.ContentType = t.MIME.Value
	} else if fileType == "url" {
		imageValue := r.FormValue("image")
		parsedURL, err := url.Parse(imageValue)
		if err != nil {
			returnErr(errFailPostFile)
			log.Print("failed to parse URL from form: ", err)
			return
		}
		client := &http.Client{}
		httpReq, err := http.NewRequest("GET", imageValue, nil)
		if err != nil {
			returnErr(errFailPostFile)
			log.Print("failed to create request: ", err)
			return
		}
		httpReq.Header.Set("User-Agent", "cynu/1.0")
		resp, err := client.Do(httpReq)
		if err != nil {
			returnErr(errFailPostFile)
			log.Print("failed to exec request: ", err)
			return
		}
		if resp.StatusCode != http.StatusOK {
			returnErr(errFailPostFile)
			log.Print("got non-200 status code: ", resp.StatusCode)
			return
		}
		defer resp.Body.Close()

		req.Filename = parsedURL.Path[strings.LastIndex(parsedURL.Path, "/")+1:]
		req.InputFile = resp.Body
		req.ContentType = resp.Header.Get("Content-Type")
	}

	if r.FormValue("name") != "" {
		req.Filename = r.FormValue("name")
	}

	if req.ContentType == "" {
		req.ContentType = mime.TypeByExtension(filepath.Ext(req.Filename))
	}

	uploadRes, err := uploadFile(req)
	if err != nil {
		log.Printf("error [%s]: %v", getRemoteAddr(r), err)
		returnErr(err)
		return
	}

	nowTime := time.Now().UTC().Unix()
	result := compatImgurImageResultJSON{
		Success: true,
		Status:  http.StatusOK,
	}
	result.Data.Link = &uploadRes.FileURL
	result.Data.ID = &uploadRes.FileURL
	result.Data.Datetime = &nowTime
	result.Data.MIMEType = &uploadRes.ContentType

	jsonData, err := json.Marshal(result)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		log.Print("failed to generate response json: ", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}
