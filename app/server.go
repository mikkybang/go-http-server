package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
)

// Ensures gofmt doesn't remove the "net" and "os" imports above (feel free to remove this!)
var (
	_ = net.Listen
	_ = os.Exit
)

type HttpRequest struct {
	method      string
	path        string
	httpVersion string
	headers     interface{}
}

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Logs from your program will appear here!")

	// Uncomment this block to pass the first stage
	l, err := net.Listen("tcp", "0.0.0.0:4221")
	defer l.Close()
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	buff := make([]byte, 1024)

	for {
		_, err := conn.Read(buff)
		if err != nil {
			fmt.Println("Error occured while reading connection buf", err.Error())
			if errors.Is(err, io.EOF) {
				fmt.Println("eof from ParseCommand")
				break
			}
			break
		}
		fmt.Println(string(buff))
		parts := bytes.Split(buff, []byte("\r\n"))
		var requestPath string

		for index, part := range parts {
			if index == 0 {
				request := strings.Split(string(part), " ")
				requestPath = request[1]
				fmt.Println(requestPath)
			}
		}
		if requestPath == "/" {
			conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
		} else {
			conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
		}
		conn.Close()
	}
}
