//go:build !windows

package appcore

import "os"

func replaceFile(source, destination string) error {
	return os.Rename(source, destination)
}
