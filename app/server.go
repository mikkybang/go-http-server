package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
)

// Ensures gofmt doesn't remove the "net" and "os" imports above (feel free to remove this!)
var (
	_ = net.Listen
	_ = os.Exit
)

const CRLF = "\r\n"

type HttpRequest struct {
	Method      string
	Path        string
	HttpVersion string
	Headers     map[string]string
}

type HttpResponse struct {
	Status  string
	Headers string
	Body    string
	Message string
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

	_, err := conn.Read(buff)
	if err != nil {
		fmt.Println("Error occured while reading connection buf", err.Error())
		if errors.Is(err, io.EOF) {
			fmt.Println("eof from ParseCommand")
			return
		}
		return
	}

	request, err := parseRequest(string(buff))
	if err != nil {
		fmt.Println("Error occured while parsing request")
	}

	response := HttpResponse{
		Status:  "200",
		Message: "OK",
	}

	headers := make(map[string]string)
	switch {
	case request.Path == "/":
		response.Status = "200"
		response.Message = "OK"
	case strings.Contains(request.Path, "echo"):
		params := strings.SplitN(request.Path, "/", 3)
		headers["Content-Type"] = "text/plain"
		headers["Content-Length"] = strconv.Itoa(len(params[2]))
		response.Body = params[2]
	case request.Path == "/user-agent":
		userAgent := request.Headers["User-Agent"]
		fmt.Println(userAgent)
		headers["Content-Type"] = "text/plain"
		headers["Content-Length"] = strconv.Itoa(len(userAgent))
		response.Body = userAgent
	default:
		response.Status = "404"
		response.Message = "Not Found"
	}

	response.Headers = parseResponseHeaders(headers)
	finalResponse := "HTTP/1.1" + " " + response.Status + " " + response.Message + CRLF + response.Headers + CRLF + response.Body
	fmt.Println(finalResponse)
	conn.Write([]byte(finalResponse))

	conn.Close()
}

func parseRequest(request string) (HttpRequest, error) {
	fmt.Println(request)
	parsedRequest := HttpRequest{}

	parts := strings.Split(request, CRLF)

	requestLine := strings.Split(parts[0], " ")
	parsedRequest.Method = requestLine[0]
	parsedRequest.Path = requestLine[1]
	parsedRequest.HttpVersion = requestLine[2]

	headers := make(map[string]string)
	for _, data := range parts[1:] {
		if data == "" {
			break
		}
		singleHeader := strings.SplitN(data, ":", 2)
		headers[singleHeader[0]] = strings.Trim(singleHeader[1], " ")
	}
	parsedRequest.Headers = headers

	return parsedRequest, nil
}

func parseResponseHeaders(headers map[string]string) string {
	b := new(bytes.Buffer)
	for key, value := range headers {
		fmt.Fprintf(b, "%s: %s"+CRLF, key, value)
	}
	return b.String()
}
