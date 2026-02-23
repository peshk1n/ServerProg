package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	host       = "localhost"
	port       = "8080"
	serverRoot = "./www"
	logFile    = "server.log"
)

var (
	logF  *os.File
	logMu sync.Mutex
)

func main() {
	listener, err := net.Listen("tcp", host+":"+port)
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	logF, err = os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer logF.Close()

	fmt.Println("Server started on", host+":"+port)
	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	requestLine, err := reader.ReadString('\n') // GET /index.html HTTP/1.1
	if err != nil {
		return
	}

	parts := strings.Split(requestLine, " ")
	if len(parts) < 2 || parts[0] != "GET" {
		return
	}

	path := parts[1] // /index.html
	filePath := filepath.Join(serverRoot, filepath.Clean(path))
	clientIP := conn.RemoteAddr().String()
	if fileExists(filePath) {
		sendFile(conn, filePath)
		writeLog(clientIP, path, 200)
	} else {
		send404(conn)
		writeLog(clientIP, path, 404)
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func sendFile(conn net.Conn, path string) {
	file, err := os.Open(path)
	if err != nil {
		send500(conn)
		writeLog(conn.RemoteAddr().String(), path, 500)
		return
	}
	defer file.Close()

	info, _ := file.Stat()
	fmt.Fprintf(conn, "HTTP/1.1 200 OK\r\n")
	fmt.Fprintf(conn, "Content-Length: %d\r\n", info.Size())
	fmt.Fprintf(conn, "Content-Type: text/html\r\n")
	fmt.Fprintf(conn, "\r\n")
	io.Copy(conn, file)
}

func send404(conn net.Conn) {
	body := "404 - File Not Found"
	fmt.Fprintf(conn, "HTTP/1.1 404 Not Found\r\n")
	fmt.Fprintf(conn, "Content-Length: %d\r\n", len(body))
	fmt.Fprintf(conn, "Content-Type: text/plain\r\n")
	fmt.Fprintf(conn, "\r\n")
	fmt.Fprintf(conn, "%s", body)
}

func send500(conn net.Conn) {
	body := "500 - Internal Server Error"
	fmt.Fprintf(conn, "HTTP/1.1 500 Internal Server Error\r\n")
	fmt.Fprintf(conn, "Content-Length: %d\r\n", len(body))
	fmt.Fprintf(conn, "Content-Type: text/plain\r\n")
	fmt.Fprintf(conn, "\r\n")
	fmt.Fprintf(conn, "%s", body)
}

func writeLog(ip, path string, status int) {
	logLine := fmt.Sprintf("%s | %s | %s | %d\n",
		time.Now().Format("2006-01-02 15:04:05"),
		ip,
		path,
		status)

	logMu.Lock()
	defer logMu.Unlock()

	logF.WriteString(logLine)
}
