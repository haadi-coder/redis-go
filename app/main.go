package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
)

const (
	Array  = '*'
	Simple = '+'
	Bulk   = '$'
)

func main() {
	fmt.Println("Logs from your program will appear here!")

	if err := run(); err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
}

func run() error {
	l, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		return fmt.Errorf("failed to bind to port 6379: %w", err)
	}

	defer func() {
		_ = l.Close()
	}()

	for {
		conn, err := l.Accept()
		if err != nil {
			return fmt.Errorf("failed to accept connection: %w", err)
		}

		go func() {
			err = handleConn(conn)
		}()

		if err != nil {
			return fmt.Errorf("failed to handle connection: %w", err)
		}
	}
}

func handleConn(conn net.Conn) error {
	reader := bufio.NewReader(conn)

	for {
		resp, err := parseResp(reader)
		if err != nil {
			_ = conn.Close()
			return fmt.Errorf("failed to parse RESP: %w", err)
		}

		if len(resp.Value) == 0 {
			continue
		}

		command := strings.ToUpper(resp.Value[0])

		switch command {
		case "PING":
			_, _ = conn.Write([]byte("+PONG\r\n"))
		case "ECHO":
			if len(resp.Value) < 2 {
				_, _ = conn.Write([]byte("$-1\r\n"))
			} else {
				resp := fmt.Sprintf("$%d\r\n%s\r\n", len(resp.Value[1]), resp.Value[1])
				_, _ = conn.Write([]byte(resp))
			}
		}
	}
}

type Resp struct {
	Type  string
	Value []string
}

func parseResp(reader *bufio.Reader) (*Resp, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read first line")
	}

	trimmed := strings.TrimSpace(line)

	if len(trimmed) == 0 {
		return nil, fmt.Errorf("empty trimmed")
	}

	switch trimmed[0] {
	case Array:
		return parseRespArray(trimmed, reader)
	case Simple:
		return &Resp{
			Type:  "simple",
			Value: []string{trimmed[1:]},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported RESP type: %c", trimmed[0])
	}
}

func parseRespArray(line string, reader *bufio.Reader) (*Resp, error) {
	var count int
	_, err := fmt.Sscanf(line, "*%d\r\n", &count)
	if err != nil {
		return nil, fmt.Errorf("failed to scan array length: %w", err)
	}

	var parts = make([]string, count)
	for i := range count {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read bulk")
		}

		var length int
		_, err = fmt.Sscanf(line, "$%d\r\n", &length)
		if err != nil {
			return nil, fmt.Errorf("failed to scan bulk length")
		}

		value := make([]byte, length)
		_, err = io.ReadFull(reader, value)
		if err != nil {
			return nil, fmt.Errorf("failed to read bulk data")
		}

		_, err = reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read final CRLF")
		}

		parts[i] = string(value)
	}

	return &Resp{
		Type:  "array",
		Value: parts,
	}, nil
}
