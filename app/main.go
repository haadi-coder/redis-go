package main

import (
	"fmt"
	"io"
	"net"
	"os"
)

func main() {
	fmt.Println("Logs from your program will appear here!")

	l, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}

	defer func() {
		_ = l.Close()
	}()

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		go handleClient(conn)
	}
}

func handleClient(conn net.Conn) {
	buffer := make([]byte, 1024)
	for {
		_, err := conn.Read(buffer)
		if err != nil {
			if err == io.EOF {
				err = conn.Close()
				if err != nil {
					fmt.Println("failed to close connection: %w", err)
				}
				return
			}
			fmt.Println("failed to read: %w", err)

			err = conn.Close()
			if err != nil {
				fmt.Println("failed to close connection: %w", err)
			}
			return
		}

		_, _ = conn.Write([]byte("+PONG\r\n"))
	}
}
