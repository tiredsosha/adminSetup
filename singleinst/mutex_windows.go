package singleinst

import "golang.org/x/sys/windows"

func Acquire(name string) (release func(), alreadyRunning bool, err error) {
	n, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, false, err
	}
	h, err := windows.CreateMutex(nil, false, n)
	if err != nil {
		return nil, false, err
	}
	if windows.GetLastError() == windows.ERROR_ALREADY_EXISTS {
		_ = windows.CloseHandle(h)
		return func() {}, true, nil
	}
	return func() { _ = windows.CloseHandle(h) }, false, nil
}
