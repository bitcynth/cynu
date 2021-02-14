package main

import (
	"errors"
	"net/http"
)

var (
	errInvalidUploadKey = errors.New("invalid upload key")
	errFailPostFile     = errors.New("failed to get file")
	errFailedRandomName = errors.New("failed to generate random filename")
	errFailedOutputFile = errors.New("error opening output file")
	errFailedWriteFile  = errors.New("failed to write to output file")
)

var errorStatusCodeValues = map[error]int{
	errInvalidUploadKey: http.StatusUnauthorized,
	errFailPostFile:     http.StatusBadRequest,
	errFailedRandomName: http.StatusInternalServerError,
	errFailedOutputFile: http.StatusInternalServerError,
	errFailedWriteFile:  http.StatusInternalServerError,
}

var errorStatusTextValues = map[error]string{
	errInvalidUploadKey: errInvalidUploadKey.Error(),
	errFailPostFile:     errFailPostFile.Error(),
}

func errorStatusCode(e error) int {
	c, ok := errorStatusCodeValues[e]
	if !ok {
		return http.StatusInternalServerError
	}
	return c
}

func errorStatusText(e error) string {
	c, ok := errorStatusTextValues[e]
	if !ok {
		return http.StatusText(errorStatusCode(e))
	}
	return c
}
