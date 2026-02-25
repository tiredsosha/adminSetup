package exe

import (
	"os"
	"os/exec"
)

func RunAndWait(exePath string, args ...string) error {
	cmd := exec.Command(exePath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func RunCommand(dir string, command string) error {
	cmd := exec.Command("cmd", "/C", command)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
