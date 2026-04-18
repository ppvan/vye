package main

import (
	"runtime"

	"github.com/rodrigocfd/windigo/co"
	"github.com/rodrigocfd/windigo/win"
)

const title = "QR generator"

func main() {
	runtime.LockOSThread() // Windows GUI must run on the OS thread

	win.CoInitializeEx(co.COINIT_APARTMENTTHREADED | co.COINIT_DISABLE_OLE1DDE)
	defer win.CoUninitialize()

	ShowMainWindow()
}
