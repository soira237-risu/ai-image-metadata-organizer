//go:build !windows

package mover

import (
	"errors"
	"syscall"
)

func isPlatformCrossDevice(err error) bool {
	return errors.Is(err, syscall.EXDEV)
}
