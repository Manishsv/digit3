package clamav

import (
	"bufio"
	"clamav-wrapper/config"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

func Scan(reader io.Reader, fileSize int64) (bool, error) {
	maxBytes := int64(config.ClamAVMaxFileSizeMB) * 1024 * 1024
	if fileSize > maxBytes {
		return false, fmt.Errorf("file too large to scan (%d bytes > max %d bytes)", fileSize, maxBytes)
	}

	address := fmt.Sprintf("%s:%d", config.ClamAVHost, config.ClamAVPort)
	fmt.Printf("Connecting to ClamAV at %s...\n", address)

	conn, err := net.DialTimeout("tcp", address, time.Duration(config.ClamAVDialTimeoutSeconds)*time.Second)
	if err != nil {
		fmt.Printf("Failed to connect to ClamAV: %v\n", err)
		return false, err
	}
	defer conn.Close()

	fmt.Println("Connected. Sending INSTREAM command...")
	_, err = fmt.Fprintf(conn, "zINSTREAM\000")
	if err != nil {
		fmt.Printf("Failed to send INSTREAM command: %v\n", err)
		return false, err
	}

	buf := make([]byte, config.ClamAVChunkSizeKB*1024)
	totalBytes := 0
	chunkCount := 0

	for {
		n, err := reader.Read(buf)
		if n > 0 {
			chunk := []byte{
				byte(n >> 24),
				byte(n >> 16),
				byte(n >> 8),
				byte(n),
			}
			_, err = conn.Write(chunk)
			if err != nil {
				fmt.Printf("Failed to write chunk header: %v\n", err)
				return false, err
			}
			_, err = conn.Write(buf[:n])
			if err != nil {
				fmt.Printf("Failed to write chunk data: %v\n", err)
				return false, err
			}
			totalBytes += n
			chunkCount++
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("Error while reading file: %v\n", err)
			return false, err
		}
	}

	fmt.Printf("Sent %d chunks (%d bytes total)\n", chunkCount, totalBytes)

	_, err = conn.Write([]byte{0, 0, 0, 0}) // EOF marker
	if err != nil {
		fmt.Printf("Failed to send EOF marker: %v\n", err)
		return false, err
	}
	fmt.Println("Sent EOF marker. Awaiting ClamAV response...")

	respBuf := bufio.NewReader(conn)
	var response []byte
	for {
		line, err := respBuf.ReadBytes('\n')
		response = append(response, line...)

		if err == io.EOF {
			break // EOF is expected after response is sent
		}
		if err != nil {
			fmt.Printf("Failed to read ClamAV response: %v\n", err)
			return false, err
		}
	}

	status := string(response)
	fmt.Printf("ClamAV response: %s\n", status)

	if strings.Contains(status, "OK") {
		fmt.Println("File is clean.")
		return true, nil
	}
	if strings.Contains(status, "FOUND") {
		fmt.Println("File is infected!")
		return false, nil
	}

	// Unknown response
	fmt.Println("Unexpected ClamAV response. Treating as infected.")
	return false, fmt.Errorf("unexpected ClamAV response: %s", status)
}
