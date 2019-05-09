package main

import (
	"os"
)

// no locking on Windows
func lock(f *os.File) error {
	return nil
}

func unlock(f *os.File) error {
	return nil
}
