package goui

import (
	"C"
	"fmt"
	"github.com/toqueteos/webbrowser"
)

const (
	WindowTitled = 0
	WindowBorderless = 1 << iota
	WindowClosable = 1 << iota
	WindowResizable = 1 << iota
	WindowMinimizable = 1 << iota
	WindowModal = 1 << iota
)

var (
	nextWindowId = 1
)

type WindowOptions struct {
	//	Required options
	Template string
	StyleFlags int
	
	//	Specify a window title, rather than relying on the HTML <title> tag
	Title string

	//	Specify window size as a fixed pixel size or percent of screen size
	PixelWidth int
	PixelHeight int
	PercentWidth float64
	PercentHeight float64
	
	//	Specify positioning rules; either centered, by pixel, or by percent.
	Centered bool
	PixelLeft int
	PixelTop int
	PercentLeft float64
	PercentTop float64
	
	RememberGeometry bool
	OpenInBrowser bool
}

type Window struct {
	handle int
	closeHandler func(window *Window)
	pushQueue chan *Message
	isBrowser bool
}

var openWindows map[int]*Window;

func init() {
	openWindows = make(map[int]*Window);
}

func OpenWindow(options WindowOptions) *Window {
	windowId := nextWindowId
	nextWindowId++
	
	url := fmt.Sprintf("%sassets/%s#%d", serverAddress, options.Template, windowId)
	
	window := &Window{
		handle: windowId,
		pushQueue: make(chan *Message, 10),
	}	

	if options.OpenInBrowser {
		//	Open in a browser if specified
		webbrowser.Open(url)
	} else {
		//	Open a window
		osOpenWindow(window, url, options.StyleFlags)
		
		//	Give it a title (if one has been specified)
		if options.Title != "" {
			window.SetTitle(options.Title)
		}
	
		//	Resize it.
		screenWidth, screenHeight := GetScreenSize()
		width, height := 100, 50
	
		switch {
			case options.PixelWidth > 0:
				width = options.PixelWidth
			case options.PercentWidth > 0:
				width = int((options.PercentWidth * float64(screenWidth)) / 100)
		}
		
		switch {
			case options.PixelHeight > 0:
				height = options.PixelHeight
			case options.PercentHeight > 0:
				height = int((options.PercentHeight * float64(screenHeight)) / 100)
		}
		
		window.SetSize(width, height)
		
		//	Position it
		top, left := 0, 0
		
		switch {
			case options.PixelLeft > 0:
				left = options.PixelLeft
			case options.PercentLeft > 0:
				left = int((options.PercentLeft * float64(screenWidth)) / 100)
		}
		
		switch {
			case options.PixelTop > 0:
				top = options.PixelTop
			case options.PercentTop > 0:
				top = int((options.PercentTop * float64(screenHeight)) / 100)
		}
		
		if options.Centered {
			left = (screenWidth - width) / 2
			top = (screenHeight - height) / 2
		}
		
		window.SetPosition(left, top)
	
		//	Remember geometry.
		if options.RememberGeometry {
			osRememberGeometry(window, options.Template)
		}
	}
	
	openWindows[window.handle] = window
	
	if options.StyleFlags & WindowModal > 0	{
		osRunModal(window)
	}
	
	return window;
}

func GetWindow(id int) *Window {
	if window, ok := openWindows[id]; ok {
		return window
	}
	return nil
}

func (window *Window) SetTitle(title string) {
	if ! window.isBrowser {
		osSetWindowTitle(window, title)
	}
}

func (window *Window) SetSize(width int, height int) {
	if ! window.isBrowser {
		osSetWindowSize(window, width, height)
	}
}

func (window *Window) SetPosition(left int, top int) {
	if ! window.isBrowser {
		osSetWindowPosition(window, left, top)
	}
}

func (window *Window) SetCloseHandler(handler func(window *Window)) {
	if ! window.isBrowser {
		window.closeHandler = handler;
	}
}

func (window *Window) Send(message Message) {
	window.pushQueue <- &message
}

func (window *Window) Close() {
	if ! window.isBrowser {
		osCloseWindow(window)
		delete(openWindows, window.handle)
	}
}
