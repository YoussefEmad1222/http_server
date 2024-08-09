package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

func main() {
	// Start listening on port 4221
	listener, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		logErrorAndExit("Failed to bind to port 4221", err)
	}

	for {
		// Accept a connection
		conn, err := listener.Accept()
		if err != nil {
			logErrorAndExit("Error accepting connection", err)
		}

		// Handle the connection
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	// Read data from the connection
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		logErrorAndExit("Error reading from connection", err)
	}

	// Parse the received message
	headerParts, body := parseMessage(buf, n)
	if len(headerParts) == 0 {
		logErrorAndExit("No header found", nil)
	}

	// Process the request
	processRequest(conn, headerParts, body)
}

func parseMessage(buf []byte, n int) ([]string, []byte) {
	receivedData := string(buf[:n])
	fmt.Println(receivedData)
	parts := strings.SplitN(receivedData, "\r\n\r\n", 2)
	header := parts[0]
	body := []byte(parts[1])
	headerParts := strings.Split(header, "\r\n")
	return headerParts, body
}

func processRequest(conn net.Conn, headerParts []string, body []byte) {
	requestLine := headerParts[0]
	headers := headerParts[1:]

	// Parse the request line
	requestParts := strings.Split(requestLine, " ")
	if len(requestParts) != 3 {
		logErrorAndExit("Invalid request", nil)
	}
	method, path := requestParts[0], "."+requestParts[1]

	// Parse headers into a map
	headerMap := parseHeaders(headers)

	// Handle the request based on the method
	switch method {
	case "GET":
		handleGetRequest(conn, path, headerMap)
	case "POST":
		handlePostRequest(conn, path, body)
	default:
		logErrorAndExit("Invalid method", nil)
	}
}

func handlePostRequest(conn net.Conn, path string, body []byte) {
	pathParts := strings.Split(path, "/")
	switch pathParts[1] {
	case "files":
		dir := os.Args[2]
		os.Create(dir + pathParts[2])
		os.WriteFile(dir+pathParts[2], body, 0644)
		conn.Write([]byte("HTTP/1.1 201 Created\r\n\r\n"))
	}
}

func parseHeaders(headers []string) map[string]string {
	headerMap := make(map[string]string)
	for _, part := range headers {
		keyValue := strings.Split(part, ": ")
		if len(keyValue) != 2 {
			logErrorAndExit("Invalid header", nil)
		}
		headerMap[keyValue[0]] = keyValue[1]
	}
	return headerMap
}

func handleGetRequest(conn net.Conn, path string, headerMap map[string]string) {
	pathParts := strings.Split(path, "/")
	switch pathParts[1] {
	case "user-agent":
		userAgent := headerMap["User-Agent"]
		writeResponse(conn, "200 OK", "text/plain", []byte(userAgent))
	case "echo":
		handleEchoRequest(conn, pathParts, headerMap)
	case "files":
		dir := os.Args[2]
		data, err := os.ReadFile(dir + pathParts[2])
		if err != nil {
			writeResponse(conn, "404 Not Found", "application/octet-stream", nil)
		} else {
			writeResponse(conn, "200 OK", "application/octet-stream", data)
		}
	default:
		if exists(path) {
			writeResponse(conn, "200 OK", "text/plain", nil)
		} else {
			writeResponse(conn, "404 Not Found", "text/plain", nil)
		}
	}
}

func handleEchoRequest(conn net.Conn, parts []string, headerMap map[string]string) {
	if len(parts) != 3 {
		logErrorAndExit("Invalid request", nil)
	}
	response := "HTTP/1.1 200 OK\r\n"
	if strings.Contains(headerMap["Accept-Encoding"], "gzip") {
		response += "Content-Encoding: gzip\r\n"
		var b bytes.Buffer
		zw := gzip.NewWriter(&b)
		zw.Write([]byte(parts[2]))
		zw.Close()
		response += "Content-Type: text/plain\r\nContent-Length: " + strconv.Itoa(b.Len()) + "\r\n\r\n"
		response := []byte(response)
		response = append(response, b.Bytes()...)
		conn.Write(response)
		return
	}
	response += "Content-Type: text/plain\r\nContent-Length: " + strconv.Itoa(len(parts[2])) + "\r\n\r\n" + parts[2]
	conn.Write([]byte(response))
}

func writeResponse(conn net.Conn, status, contentType string, body []byte) {
	response := fmt.Sprintf("HTTP/1.1 %s\r\nContent-Type: %s\r\nContent-Length: %d\r\n\r\n",
		status, contentType, len(body))
	conn.Write([]byte(response))
	if body != nil {
		conn.Write(body)
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || !os.IsNotExist(err)
}

func logErrorAndExit(message string, err error) {
	if err != nil {
		fmt.Println(message+":", err.Error())
	} else {
		fmt.Println(message)
	}
	os.Exit(1)
}
