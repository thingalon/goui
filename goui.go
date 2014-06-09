package goui

import (
	"C"
	"log"
	"fmt"
	"net"
	"net/http"
	"strings"
	"strconv"
	"io/ioutil"
	"encoding/json"
	"github.com/toqueteos/webbrowser"
)

type Data map[string]interface{}

type Message struct {
	Type string
	Params Data
}

var (
	serverAddress string
	assetPath string = "assets/"
	bindataSource func(name string)([]byte, error)
	
	messageHandlers map[string]func(*Window, *Message) Data
	
	guiReady chan struct{}
);

func init() {
	messageHandlers = make(map[string]func(*Window, *Message) Data)
	SetMessageHandler("goui.checkAlive", handleCheckAlive)
	SetMessageHandler("goui.closeWindow", handleCloseWindow)
}

func Run(readyCallback func()) {
	guiReady = make(chan struct{})
	portChannel := make(chan int)
	
	go startWebServer(portChannel)
	go waitForReady(portChannel, readyCallback)
	osInit();
}

func SetMessageHandler(messageName string, handler func(*Window, *Message) Data) {
	messageHandlers[messageName] = handler
}

func Stop() {
	osStop();
}

func SetBindataSource(assetFunc func(name string)([]byte, error)) {
	bindataSource = assetFunc
}

func SetAssetPath(a string) {
	assetPath = a
	if ! strings.HasSuffix(assetPath, "/") {
		assetPath += "/"
	}
}

func GetScreenSize() (width int, height int) {
	return osGetScreenSize()
}

func OpenInBrowser(template string) {
	webbrowser.Open(serverAddress + "assets/" + template)
}

func waitForReady(portChannel chan int, callback func()) {
	//	Wait for the GUI system to be ready.
	<- guiReady
	
	//	Wait for the webserver to provide a listen port
	listeningPort := <- portChannel
	serverAddress = fmt.Sprintf("http://127.0.0.1:%d/", listeningPort)
	
	//	Try a test message to the webserver to ensure things look sane.
	response := makeRequest(&Message{Type:"goui.checkAlive"})
	if response["alive"] != true {
		panic("Invalid response from webserver!")
	}
	
	callback()
}

func makeRequest(message *Message) Data {
	jsonData, _ := json.Marshal(message)
	dataReader := strings.NewReader(string(jsonData))
	
	response, err := http.Post(serverAddress + "callback", "application/json; charset=utf-8", dataReader)
	if err != nil {
		panic("Failed to contact webserver: " + err.Error())
	}
	defer response.Body.Close()
	
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		panic("Error reading webserver response: " + err.Error())
	}
	
	var parsedResponse Data
	err = json.Unmarshal(body, &parsedResponse)
	if err != nil {
		panic("Error parsing webserver response: " + err.Error())
	}
	
	return parsedResponse
}

func startWebServer(portChannel chan int) {
	//	Listen to an OS-assigned port on 127.0.0.1.
	listener, err := net.Listen("tcp", ":0")	
	if err != nil {
		panic("Failed to open a TCP listener: " + err.Error())
	}
	
	//	Pull the listener address to pieces to find the port number
	address := listener.Addr().String()
	address_pieces := strings.Split(address, ":")
	port, err := strconv.Atoi(address_pieces[len(address_pieces) - 1])
	if err != nil {
		panic("Failed to interpret TCP listener port: " + err.Error())
	}
	portChannel <- port

	//	Start a webserver on the listener
	http.HandleFunc("/goui.js", serveJavascript)
	http.HandleFunc("/callback", handleAjaxRequest)
	http.HandleFunc("/assets/", serveAsset)
	if err = http.Serve(listener, nil); err != nil {
		panic("Failed to open a webserver: " + err.Error())
	}
}

func serveJavascript(responseWriter http.ResponseWriter, request *http.Request) {
	fmt.Fprint(responseWriter, javascript)
}

func serveAsset(responseWriter http.ResponseWriter, request *http.Request) {
	path := assetPath + strings.TrimPrefix(request.URL.Path, "/assets/")
	var data []byte
	var err error

	//	Look for a go-bindata source first
	if bindataSource != nil {
		data, _ = bindataSource(path)
	}
	
	if data == nil || len(data) == 0 {
		data, err = ioutil.ReadFile(path)
		if err != nil {
			http.NotFound(responseWriter, request)
			return
		}
	}
	
	fmt.Fprint(responseWriter, string(data))
}

func handleAjaxRequest(responseWriter http.ResponseWriter, rawRequest *http.Request) {
	body, err := ioutil.ReadAll(rawRequest.Body)
	if err != nil {
		log.Println("Warning: received unreadable request")
		return;
	}
	
	var request Message
	var response Data

	err = json.Unmarshal(body, &request)
	if err != nil {
		log.Println("Warning: received invalid request: " + string(body))
		return;
	}
	

	//	Find the window that sent this message
	var window *Window
	if windowId, ok := request.Params["windowId"].(float64); ok {
		window = GetWindow(int(windowId))
	}

	if ( request.Type == "goui.longPoll" ) {
		//	Special handler; long poll.
		closeNotify := responseWriter.(http.CloseNotifier).CloseNotify()
		select {
			case message := <- window.pushQueue:
				response = Data{
					"Type": message.Type,
					"Params": message.Params,
				}
				
			case <-closeNotify:
				return
		}
	} else {	
		//	Generic handlers; registered via SetMessageHandler.
		if handler, ok := messageHandlers[request.Type]; ok {
			response = handler(window, &request)
			
			if response == nil {
				//	Called a handler, it didn't return anything. Return an empty response.
				response = Data{}
			}
		}
	}

	if response == nil {
		//	Failed to find a handler for a message.
		fmt.Println( "Invalid message received: " + request.Type )
		response = Data{"error": "unknown message"}
	}
	
	if response == nil {
		//	Garbage received.
		fmt.Println("Garbage received: " + string(body))
		response = Data{"error": "invalid request"}
	}

	data, _ := json.Marshal(response)
	fmt.Fprint(responseWriter, string(data));
}

func handleCheckAlive(window *Window, request *Message) Data {
	return Data{"alive": true}
}

func handleCloseWindow(window *Window, request *Message) (reply Data) {
	window.Close()
	return
}

//export guiReadyCallback
func guiReadyCallback() {
	guiReady <- struct{}{}
}

//export guiWindowCloseCallback
func guiWindowCloseCallback(windowId C.int) {
	if window := GetWindow(int(windowId)); window != nil {
		if (window.closeHandler != nil) {
			handler := window.closeHandler
			window.closeHandler = nil
			handler(window)
		}
	}
}
