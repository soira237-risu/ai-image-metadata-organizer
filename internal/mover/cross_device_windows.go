//go:build windows

package mover

import (
	"errors"

	"golang.org/x/sys/windows"
)

func isPlatformCrossDevice(err error) bool {
	return errors.Is(err, windows.ERROR_NOT_SAME_DEVICE)
}
