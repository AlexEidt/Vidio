package main

import (
	"errors"
	"os"
)

// Returns true if file exists, false otherwise.
// https://stackoverflow.com/questions/12518876/how-to-check-if-a-file-exists-in-go
func Exists(filename string) bool {
	_, err := os.Stat(filename)
	if err == nil {
		return true
	}
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	return false
}
