//go:build windows

package platform

import (
	"fmt"
	"os/exec"
)

type windowsDesktop struct{}

func newDesktop() Desktop {
	return &windowsDesktop{}
}

func (d *windowsDesktop) OpenURL(url string) error {
	return exec.Command("cmd", "/c", "start", "", url).Start()
}

func (d *windowsDesktop) OpenApp(nameOrPath string) error {
	if nameOrPath == "" {
		return fmt.Errorf("name or path is required")
	}
	return exec.Command("cmd", "/c", "start", "", nameOrPath).Start()
}
