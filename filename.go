package main

import (
	"crypto/rand"
	"encoding/hex"
	"mime"
)

var mimetypeToExt = map[string]string{
	"text/plain": "txt",
	"image/png":  "png",
	"image/jpeg": "jpg",
}

func getExtByMimetype(mimetype string) (string, error) {
	t, ok := mimetypeToExt[mimetype]
	if ok {
		return "." + t, nil
	}

	types, err := mime.ExtensionsByType(mimetype)
	if err != nil || types == nil || len(types) == 0 {
		return "", err
	}

	return types[0], nil
}

func randomFilename() (string, error) {
	length := 12
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
