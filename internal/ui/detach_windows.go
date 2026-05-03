//go:build windows

package ui

import "syscall"

func detachAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{}
}
