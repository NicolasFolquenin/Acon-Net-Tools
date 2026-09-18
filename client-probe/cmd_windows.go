//go:build windows

package main

import (
	"os/exec"
	"syscall"
)

// hideWindow define a flag CREATE_NO_WINDOW para evitar janelas de console piscando no Windows
func hideWindow(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.HideWindow = true
	cmd.SysProcAttr.CreationFlags = 0x08000000 // CREATE_NO_WINDOW
}
