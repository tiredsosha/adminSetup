package elevate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

type tokenElevation struct {
	TokenIsElevated uint32
}

func IsElevated() bool {
	var token windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &token); err != nil {
		return false
	}
	defer token.Close()

	var elev tokenElevation
	var outLen uint32
	err := windows.GetTokenInformation(
		token,
		windows.TokenElevation,
		(*byte)(unsafe.Pointer(&elev)),
		uint32(unsafe.Sizeof(elev)),
		&outLen,
	)
	if err != nil {
		return false
	}
	return elev.TokenIsElevated != 0
}

func RelaunchAsAdmin() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, err = filepath.Abs(exe)
	if err != nil {
		return err
	}

	argLine := strings.Join(quoteArgs(os.Args[1:]), " ")

	verb, _ := windows.UTF16PtrFromString("runas")
	file, _ := windows.UTF16PtrFromString(exe)
	params, _ := windows.UTF16PtrFromString(argLine)

	if err := windows.ShellExecute(0, verb, file, params, nil, 1); err != nil {
		if errno, ok := err.(windows.Errno); ok && errno == 1223 {
			return fmt.Errorf("UAC cancelled by user")
		}
		return err
	}
	return nil
}

func quoteArgs(args []string) []string {
	out := make([]string, 0, len(args))
	for _, a := range args {
		if a == "" {
			out = append(out, `""`)
			continue
		}
		if !strings.ContainsAny(a, " \t\"") {
			out = append(out, a)
			continue
		}
		a = strings.ReplaceAll(a, `"`, `\"`)
		out = append(out, `"`+a+`"`)
	}
	return out
}
