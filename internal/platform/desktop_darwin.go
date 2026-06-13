//go:build darwin

package platform

import (
	"fmt"
	"os/exec"
	"strings"
)

type darwinDesktop struct{}

func newDesktop() Desktop {
	return &darwinDesktop{}
}

func (d *darwinDesktop) OpenURL(url string) error {
	return exec.Command("open", url).Start()
}

func (d *darwinDesktop) OpenApp(nameOrPath string) error {
	if nameOrPath == "" {
		return fmt.Errorf("name or path is required")
	}
	if strings.Contains(nameOrPath, "/") || strings.Contains(nameOrPath, `\`) {
		return exec.Command("open", nameOrPath).Start()
	}
	return exec.Command("open", "-a", nameOrPath).Start()
}
