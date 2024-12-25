package main

import (
	"bytes"
	"compress/gzip"
	"errors"
	"flag"
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
	Body        string
}

type HttpResponse struct {
	Status  string
	Headers string
	Body    string
	Message string
}

var defaultFileDir = "/tmp/"

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Logs from your program will appear here!")

	fileDirFlag := flag.String("directory", "", "This allows you to specify a directory for files")

	flag.Parse()

	if *fileDirFlag != "" {
		fmt.Println(*fileDirFlag)
		defaultFileDir = *fileDirFlag
	}

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
		headers["Content-Type"] = "text/plain"
		headers["Content-Length"] = strconv.Itoa(len(userAgent))
		response.Body = userAgent
	case strings.Contains(request.Path, "files"):
		params := strings.SplitN(request.Path, "/", 3)
		fileName := strings.Join(params[2:], "/")
		switch request.Method {
		case "GET":
			if fileName == "" {
				response.Status = "404"
				response.Message = "Not Found"
			} else {
				fileBuff, err := os.ReadFile(defaultFileDir + fileName)
				if err != nil {
					fmt.Println(err)
					response.Status = "404"
					response.Message = "Not Found"
				} else {
					response.Body = string(fileBuff)
					response.Status = "200"
					headers["Content-Type"] = "application/octet-stream"
					headers["Content-Length"] = strconv.Itoa(len(fileBuff))
				}
			}
		case "POST":
			if fileName == "" {
				response.Status = "404"
				response.Message = "Not Found"
			} else {
				fmt.Println(request.Body)
				fileBuffer := []byte(request.Body)
				file, err := os.Create(defaultFileDir + fileName)
				file.Write(bytes.Trim(fileBuffer, "\x00"))
				file.Close()
				if err != nil {
					fmt.Println("Error Occured while saving file")
					fmt.Println(err)
					response.Status = "400"
					response.Message = "Error Occured while saving file"
				} else {
					response.Status = "201"
					response.Message = "Created"
				}
			}
		default:
			response.Status = "405"
			response.Body = "Method not allowed"
		}
	default:
		response.Status = "404"
		response.Message = "Not Found"
	}
	if request.Headers["Accept-Encoding"] != "" {
		fmt.Println("Compression Types")
		fmt.Println(request.Headers["Accept-Encoding"])
		for _, value := range strings.Split(request.Headers["Accept-Encoding"], ",") {
			if strings.TrimSpace(value) == "gzip" {
				headers["Content-Encoding"] = "gzip"
				var writerBuffer bytes.Buffer
				_, err := gzip.NewWriter(&writerBuffer).Write([]byte(response.Body))
				if err != nil {
					fmt.Println("Error Occured while compressing data")
					response.Status = "500"
					response.Message = "Internal Server Error"
				}
				response.Body = writerBuffer.String()
				break
			}
		}
	}

	response.Headers = parseResponseHeaders(headers)
	finalResponse := "HTTP/1.1" + " " + response.Status + " " + response.Message + CRLF + response.Headers + CRLF + response.Body
	fmt.Println(finalResponse)
	conn.Write([]byte(finalResponse))

	conn.Close()
}

func parseRequest(request string) (HttpRequest, error) {
	// fmt.Println(request)
	parsedRequest := HttpRequest{}

	parts := strings.Split(request, CRLF)

	requestLine := strings.Split(parts[0], " ")
	parsedRequest.Method = requestLine[0]
	parsedRequest.Path = requestLine[1]
	parsedRequest.HttpVersion = requestLine[2]

	headers := make(map[string]string)

	for index, data := range parts[1:] {
		if data == "" {
			if len(parts) >= index+2 {
				parsedRequest.Body = requestBodyParser(parts[index+2:], headers["Content-Type"])
			}
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

func requestBodyParser(body []string, contentType string) string {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	switch contentType {
	case "application/octet-stream":
		return strings.TrimRight(strings.Join(body, CRLF), CRLF)
	default:
		return ""
	}
}
