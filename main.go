package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/alecthomas/kong"
)

const headers = "HTTP/1.1 200 OK\r\n" +
	"Cache-Control: no-cache\r\n" +
	"Transfer-Encoding: chunked\r\n" +
	"Content-Type: text/plain; charset=iso-8859-1\r\n" +
	"X-Content-Type-Options: nosniff\r\n" +
	"\r\n"

const port = ":6969"

var cli struct {
	Payload string `arg name:"payload" help:"content to send as a response." type:"string"`
	Port    string `arg name:"port" help:"port to bind to." type:"int" default:8080 optional`
}

func main() {
	kong.Parse(&cli,
		kong.Name("reverse-slowloris"),
		kong.Description("A server that sends a slow HTTP response forever to whoever connects to it."))

	var requestNum = 0

	payload, err := ioutil.ReadFile(cli.Payload)
	if err != nil {
		log.Fatalf("Failed to load %s: %s", cli.Payload, err)
	}
	chunk := []byte(fmt.Sprintf("%x\r\n%s\r\n", len(payload), payload))

	log.Printf("Starting server on port %s", cli.Port)
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", cli.Port))
	if err != nil {
		log.Fatalf("Failed to start server :%s", err)
	}

	// Close the listener when the application closes.
	defer listener.Close()
	for {
		// Listen for an incoming connection.
		conn, err := listener.Accept()
		if err != nil {
			log.Fatalf("Error accepting: ", err.Error())
		}
		// Handle connections in a new goroutine.
		go handleRequest(conn, requestNum, chunk)
		requestNum++
	}
}

func getProbableRemoteIP(request *http.Request, conn net.Conn) string {
	// won't follow back through our proxies in front of CF
	requester := request.Header.Get("CF-Connecting-IP")
	if requester == "" {
		requester = fmt.Sprintf("%s [CF bypassed]", conn.RemoteAddr().String())
	}
	return requester
}

func getParsedRequest(conn net.Conn) (*http.Request, error) {
	buf := make([]byte, 2048)
	_, err := conn.Read(buf)
	if err != nil {
		log.Println("Error reading:", err.Error())
		return nil, errors.New("Error reading headers from socket")
	}
	readRequest, err := http.ReadRequest(bufio.NewReader(strings.NewReader(string(buf))))
	if err != nil {
		log.Println("Error parsing:", err.Error())
		return nil, errors.New("Error parsing headers")
	}
	return readRequest, nil
}

func handleRequest(conn net.Conn, requestNum int, payload []byte) {
	defer conn.Close()
	started := time.Now()
	parsedRequest, err := getParsedRequest(conn)
	if err != nil {
		log.Printf("%d | %s", requestNum, err)
		return
	}
	requester := getProbableRemoteIP(parsedRequest, conn)
	conn.Write([]byte(headers))

	log.Printf(
		"%d | %s connected asking for %s, starting to stream response\n",
		requestNum,
		requester,
		parsedRequest.URL.RequestURI(),
	)
	keepGoing := true
	for {
		if !keepGoing {
			break
		}
		for _, char := range payload {
			_, err := conn.Write([]byte{uint8(char)})
			// A failure here is expected because it is the only way out of this infinite response
			if err != nil {
				keepGoing = false
				break
			}
			time.Sleep(75 * time.Millisecond)
		}
		if keepGoing {
			elapsed := time.Since(started).Round(time.Second)
			log.Printf("%d | %s Has been streaming for %s\n", requestNum, requester, elapsed)
		}
	}
	elapsed := time.Since(started).Round(time.Second)
	log.Printf("%d | %s closed their connection after %s\n", requestNum, requester, elapsed)
}
