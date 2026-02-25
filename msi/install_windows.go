package msi

import (
	"os"
	"os/exec"
)

func RunAndWait(msiPath string) error {
	cmd := exec.Command("msiexec.exe", "/i", msiPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
