package goui

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework WebKit -framework IOKit
#import "goui_osx.m.txt"
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

func osOpenWindow(window *Window, url string, styleFlags int) {
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
	
	C.OpenWindow(C.int(window.handle), C.CString(url), nsflags, styleFlags & WindowModal > 0)
}

func osStop() {
	C.StopApp()
}

func osCloseWindow(window *Window) {
	C.CloseWindow(C.int(window.handle))
}

func osSetWindowTitle(window *Window, title string) {
	C.SetWindowTitle(C.int(window.handle), C.CString(title))
}

func osGetScreenSize() (int, int) {
	var width, height C.int
	C.GetScreenSize(&width, &height)
	return int(width), int(height)
}

func osSetWindowSize(window *Window, width int, height int) {
	C.SetWindowSize(C.int(window.handle), C.int(width), C.int(height))
}

func osSetWindowPosition(window *Window, left int, top int) {
	C.SetWindowPosition(C.int(window.handle), C.int(left), C.int(top))
}

func osRememberGeometry(window *Window, key string) {
	C.RememberWindowGeometry(C.int(window.handle), C.CString(key))
}

func osRunModal(window *Window) {
	C.RunModal(C.int(window.handle))
}

