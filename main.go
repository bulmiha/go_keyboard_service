package main

import (
	"flag"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
	"github.com/jacobsa/go-serial/serial"
)

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

var addr = flag.String("a", "localhost:8080", "http service address")
var toDump = flag.Bool("d", false, "Dump all keypress data")
var serialInterface = flag.String("i", "", "Serial Arduino board interface (required)")
var serialSpeed = flag.Uint("b", 9600, "Serial baud rate")

var upgrader = websocket.Upgrader{} // use default options
var serialPort io.ReadWriteCloser

func home(w http.ResponseWriter, r *http.Request) {
	homeTemplate.Execute(w, "ws://"+r.Host)
}

var homeTemplate *template.Template

func echo(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade:", err)
		return
	}
	defer c.Close()

	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("Read:", err)
			break
		}
		log.Printf("Received: %s", message)
		err = c.WriteMessage(mt, message)
		if err != nil {
			log.Println("Write:", err)
		}

	}
}

func keyAPI(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade:", err)
		return
	}
	defer c.Close()

	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Println("Read:", err)
			break
		}
		if *toDump {
			log.Printf("Received: %02X %d %s", keysToCodes[message[1]], message[1], func() string {
				if message[0] > 0 {
					return "Up"
				}
				return "Down"
			}())
			//log.Println(keysToCodes[message[1]])
			//log.Println("Received:",message[0],message[1])
		}

		if *serialInterface != "demo" {
			serialPort.Write(message)
		}
	}
}

func main() {
	flag.Parse()
	if *serialInterface == "" {
		flag.Usage()
		os.Exit(1)
	}
	var err error
	homeTemplate, err = template.ParseFiles("template.html")
	if err != nil {
		panic(err)
	}
	if *serialInterface != "demo" {
		serialOptions := serial.OpenOptions{
			PortName:        *serialInterface,
			BaudRate:        *serialSpeed,
			DataBits:        8,
			StopBits:        1,
			MinimumReadSize: 4,
		}

		serialPort, err = serial.Open(serialOptions)
		if err != nil {
			panic(err)
		}
		defer serialPort.Close()
	}
	log.SetFlags(0)
	http.HandleFunc("/", home)
	http.HandleFunc("/echo", echo)
	http.HandleFunc("/keys", keyAPI)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
