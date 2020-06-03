package main

import (
	"flag"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/fcgi"
	"net/http/pprof"
	"os"

	"github.com/gorilla/handlers"
	"github.com/gorilla/websocket"
	"github.com/jacobsa/go-serial/serial"
	"github.com/markbates/pkger"
)

// KeyCode mop to comvert JS KeyCode to PS/2 Scan Code
var keysToCodes = map[byte]byte{

	96:  0x70,
	49:  0x16,
	97:  0x69,
	50:  0x1e,
	98:  0x72,
	51:  0x26,
	99:  0x7A,
	52:  0x25,
	100: 0x6B,
	53:  0x2e,
	101: 0x73,
	54:  0x36,
	102: 0x74,
	55:  0x3d,
	103: 0x6C,
	56:  0x3e,
	104: 75,
	57:  0x46,
	105: 0x7d,
	48:  0x45,
	// 51:  0x15,
}

// Setting up command line flags

// Serving address
var addr = flag.String("a", "127.0.0.1:8080", "Binding address")

// Dumping all keyboard data to stdout
var toDump = flag.Bool("d", false, "Dump all keypress data")

// Serial interface pseudo-file
var serialInterface = flag.String("i", "", "Serial Arduino board interface (required)")

// Serial interface baud-rate
var serialSpeed = flag.Uint("b", 9600, "Serial baud rate")

// Use FastCGI protocol
var asFCGI = flag.Bool("cgi", false, "Start in FCGI mode")

// WebSocket html connection upgrader
var upgrader = websocket.Upgrader{} // use default options

// Serial interface object
var serialPort io.ReadWriteCloser

// Serving main page using a template
func home(w http.ResponseWriter, r *http.Request) {
	homeTemplate.Execute(w, r.Host)
}

// Precompiled home page template object
var homeTemplate *template.Template

// WebSocket pased keyboard API handler
func keyAPI(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true } // Upgrade connetion to WebSocket no matter the origin.
	c, err := upgrader.Upgrade(w, r, nil)                             // Do the connection upgrade.
	if err != nil {
		log.Println("Upgrade:", err)
		return
	}
	defer c.Close()

	for { // Infinite serving loop
		_, message, err := c.ReadMessage() // Get a message
		if err != nil {
			log.Println("Read:", err)
			break
		}

		if *toDump { // If dumping is enabled print the message to stdout
			log.Printf("Received: %02X %d %s", keysToCodes[message[1]], message[1], func() string {
				if message[0] > 0 {
					return "Up"
				}
				return "Down"
			}())
		}

		if *serialInterface != "demo" { // If it's not in demo mode, send the message to arduino.
			serialPort.Write(message)
		}
	}
}

func main() {

	flag.Parse()                // Get the flags set using command line and process them into set variables.
	if *serialInterface == "" { // Serial interface is required. If it's not set display the help and quit.
		flag.Usage()
		os.Exit(1)
	}
	var err error
	templateTextFile, err := pkger.Open("/template.html") // Open template text from embeded data block.
	if err != nil {
		panic(err)
	}
	templateText, err := ioutil.ReadAll(templateTextFile) // Read the template text.
	if err != nil {
		panic(err)
	}
	templateTextFile = nil                                                // Mem cleanup.
	homeTemplate, err = template.New("index").Parse(string(templateText)) // Compile the template.
	if err != nil {
		panic(err)
	}
	templateText = nil              // Mem cleanup.
	if *serialInterface != "demo" { // Set up the serial options.
		serialOptions := serial.OpenOptions{
			PortName:        *serialInterface,
			BaudRate:        *serialSpeed,
			DataBits:        8,
			StopBits:        1,
			MinimumReadSize: 4,
		}

		serialPort, err = serial.Open(serialOptions) // Open serial connection.
		if err != nil {
			panic(err)
		}
		defer serialPort.Close()
	}
	log.SetFlags(0)
	// Set up http server handlers
	r := http.NewServeMux()
	r.Handle("/static/", http.StripPrefix("/static/", http.FileServer(pkger.Dir("/static"))))
	r.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(pkger.Dir("/assets"))))
	r.Handle("/libs/", http.StripPrefix("/libs/", http.FileServer(pkger.Dir("/libs"))))
	r.HandleFunc("/keys", keyAPI)
	r.HandleFunc("/", home)
	r.HandleFunc("/debug/pprof/", pprof.Index)
	r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	r.HandleFunc("/debug/pprof/profile", pprof.Profile)
	r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	r.HandleFunc("/debug/pprof/trace", pprof.Trace)

	if *asFCGI {
		l, err := net.Listen("tcp", *addr)
		if err != nil {
			panic(err)
		}
		log.Fatal(fcgi.Serve(l, handlers.LoggingHandler(os.Stdout, r)))
	} else {
		log.Fatal(http.ListenAndServe(*addr, handlers.LoggingHandler(os.Stdout, r)))
	}
}
