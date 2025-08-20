package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
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
	cache := newCache()

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
			err = handleConn(conn, cache)
		}()

		if err != nil {
			return fmt.Errorf("failed to handle connection: %w", err)
		}
	}
}

func handleConn(conn net.Conn, cache *cache) error {
	defer func() {
		_ = conn.Close()
	}()

	reader := bufio.NewReader(conn)

	for {
		resp, err := parseResp(reader)
		if err != nil {
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
				_, _ = conn.Write(toBulk(resp.Value[1]))
			}
		case "GET":
			value, found := cache.get(resp.Value[1])
			if !found {
				_, _ = conn.Write([]byte("$-1\r\n"))
			} else {
				_, _ = conn.Write(toBulk(value))
			}
		case "SET":
			var ttl time.Duration
			if len(resp.Value) > 3 {
				if strings.ToUpper(resp.Value[3]) == "PX" {
					wait, _ := strconv.Atoi(resp.Value[4])
					ttl = time.Duration(wait) * time.Millisecond
				}
			}

			cache.set(resp.Value[1], resp.Value[2], ttl)
			_, _ = conn.Write([]byte("+OK\r\n"))

		default:
			return fmt.Errorf("unknown command: %s", command)
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
		return fromArray(trimmed, reader)
	case Bulk:
		return fromBulk(trimmed, reader)
	case Simple:
		return &Resp{
			Type:  "simple",
			Value: []string{trimmed[1:]},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported RESP type: %c", trimmed[0])
	}
}

func fromArray(line string, reader *bufio.Reader) (*Resp, error) {
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

func fromBulk(line string, reader *bufio.Reader) (*Resp, error) {
	var length int
	_, err := fmt.Sscanf(line, "$%d", &length)
	if err != nil {
		return nil, fmt.Errorf("failed to scan bulk size: %w", err)
	}

	value := make([]byte, length)
	_, err = io.ReadFull(reader, value)
	if err != nil {
		return nil, fmt.Errorf("failed to read bulk data: %w", err)
	}

	_, err = reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read CRLF after bulk: %w", err)
	}

	return &Resp{
		Type:  "bulk",
		Value: []string{string(value)},
	}, nil
}

func toBulk(str string) []byte {
	bytes := len(str)
	bulk := fmt.Sprintf("$%d\r\n%s\r\n", bytes, str)

	return []byte(bulk)
}
