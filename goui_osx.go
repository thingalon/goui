package goui

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework WebKit -framework IOKit
#import "goui_osx.m"
*/
import "C"

import (
	"runtime"
)

const (
	nsBorderlessWindowMask = 0
	nsTitledWindowMask = 1
	nsClosableWindowMask = 1 << 1
	nsMiniaturizableWindowMask = 1 << 2
	nsReizableWindowMask = 1 << 3
)

func osInit() {
	runtime.LockOSThread()
	C.StartApp()
}

func osOpenWindow(url string, styleFlags int) C.int {
	var nsflags C.int

	//	Translate goui window flags to Cocoa window flags...
	if ( styleFlags & WindowBorderless > 0 ) {
		nsflags = nsBorderlessWindowMask
	} else {
		nsflags = nsTitledWindowMask
		
		if ( styleFlags & WindowClosable > 0 ) {
			nsflags |= nsClosableWindowMask
		}
		
		if ( styleFlags & WindowResizable > 0 ) {
			nsflags |= nsReizableWindowMask
		}
		
		if ( styleFlags & WindowMinimizable > 0 ) {
			nsflags |= nsMiniaturizableWindowMask
		}
	}
	
	return C.OpenWindow(C.CString(url), nsflags, styleFlags & WindowModal > 0)
}

func osStop() {
	C.StopApp()
}

func osCloseWindow(handle C.int) {
	C.CloseWindow(handle)
}

func osSetWindowTitle(handle C.int, title string) {
	C.SetWindowTitle(handle, C.CString(title))
}

func osGetScreenSize() (int, int) {
	var width, height C.int
	C.GetScreenSize(&width, &height)
	return int(width), int(height)
}

func osSetWindowSize(handle C.int, width int, height int) {
	C.SetWindowSize(handle, C.int(width), C.int(height))
}

func osSetWindowPosition(handle C.int, left int, top int) {
	C.SetWindowPosition(handle, C.int(left), C.int(top))
}

func osRememberGeometry(handle C.int, key string) {
	C.RememberWindowGeometry(handle, C.CString(key))
}

func osRunModal(handle C.int) {
	C.RunModal(handle)
}

