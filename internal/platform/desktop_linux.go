//go:build linux

package platform

import (
	"fmt"
	"os/exec"
)

type linuxDesktop struct{}

func newDesktop() Desktop {
	return &linuxDesktop{}
}

func (d *linuxDesktop) OpenURL(url string) error {
	return exec.Command("xdg-open", url).Start()
}

func (d *linuxDesktop) OpenApp(nameOrPath string) error {
	if nameOrPath == "" {
		return fmt.Errorf("name or path is required")
	}
	return exec.Command("xdg-open", nameOrPath).Start()
}
