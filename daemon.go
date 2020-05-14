package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
)

func cleanUpExistingSocket(options ExecutionOptions) {
	_, err := os.Stat(options.SocketPath)
	if err != nil {
		if os.IsNotExist(err) {
			// File not found, good!
			return
		}

		// Any error other than file not found
		log.Fatalf("Error reading socket path '%s': %s", options.SocketPath, err)
	}

	if options.ForceSocketOverwrite {
		// Allow us to overwrite existing files
		if err := os.RemoveAll(options.SocketPath); err == nil {
			// Successfully deleted
			return
		}
		log.Fatalf("Could not remove existing file at '%s': %s", options.SocketPath, err)
	}
}

func writeResponse(connection net.Conn, response Response) {

	output, _ := json.MarshalIndent(response, "", " ")

	writer := bufio.NewWriter(connection)
	_, err := writer.WriteString(string(output) + "\n")
	if err == nil {
		_ = writer.Flush()
	} else {
		log.Printf("Error writing response: %s", err)
	}
}

func handleConnection(connection net.Conn) {
	//noinspection GoUnhandledErrorResult
	defer connection.Close()

	decoder := json.NewDecoder(connection)

	var req Request
	err := decoder.Decode(&req)
	if err != nil {
		writeResponse(connection, Response{ExitCode: 100, Content: fmt.Sprintf("Error decoding request: %s\n", err)})
		return
	}

	if req.StatusCheck {
		// All we need to do is say we're up
		writeResponse(connection, Response{ExitCode: 0, Content: "OK\n"})
		return
	}

	// Load repo
	repo := loadRepo(req)

	// Build response
	response := buildResponse(req, repo)
	writeResponse(connection, response)
}

func daemonMain(options ExecutionOptions) {
	cleanUpExistingSocket(options)

	// Handle shutdown better
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// At this point we should be clear to create a socket
	listener, err := net.Listen("unix", options.SocketPath)
	if err != nil {
		log.Fatal(err)
	}

	// Try to have clean shutdowns
	done := false
	shutdown := func() {
		log.Printf("Shutting down...")
		done = true

		// Cleanup
		log.Printf("Closing listener")
		err := listener.Close()
		if err != nil {
			log.Printf("Failed to close listener: %s", err)
		}
	}

	// Cleanup on signal
	go func() {
		sig := <-sigs
		log.Printf("Received signal: %s", sig)
		shutdown()
	}()

	log.Printf("Listening on: %s", options.SocketPath)

	for !done {
		connection, err := listener.Accept()
		if err != nil {
			if done {
				break
			}

			log.Printf("Error accepting: %s", err)
			continue
		}

		go handleConnection(connection)
	}
}

func daemonCheckMain(options ExecutionOptions) {
	connection, err := net.Dial("unix", options.SocketPath)
	if err != nil {
		log.Fatalf("Failed to connect to socket: '%s': %s", options.SocketPath, err)
	}
	//noinspection GoUnhandledErrorResult
	defer connection.Close()

	req := Request{
		StatusCheck: true,
	}
	encoder := json.NewEncoder(connection)
	err = encoder.Encode(req)
	if err != nil {
		log.Fatalf("Error encoding request: %s", err)
	}

	decoder := json.NewDecoder(connection)
	var response Response
	err = decoder.Decode(&response)
	if err != nil {
		log.Fatalf("Error decoding response: %s\n", err)
	}

	os.Exit(response.ExitCode)
}
