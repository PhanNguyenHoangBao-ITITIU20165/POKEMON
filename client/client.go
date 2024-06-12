package main

import (
    "bufio"
    "fmt"
    "log"
    "net"
    "os"
    "strings"
)

func main() {
    // Establish Connection
    conn, err := net.Dial("tcp", "localhost:8080")  
    if err != nil {
        log.Fatalf("Could not connect to server: %v", err)
    }
    defer conn.Close()

    // Setup Input/Output
    reader := bufio.NewReader(os.Stdin)       // For reading player input from the console
    serverReader := bufio.NewReader(conn)  // For reading messages from the server

    // Goroutine to continuously receive server messages
    go func() {
        for {
            message, err := serverReader.ReadString('\n')
            if err != nil {
                log.Fatalf("Error reading from server: %v", err) 
            }
            fmt.Print(message) // Display server messages on the console (e.g., battle status)
        }
    }()

    // Main Loop: Read and Send User Input
    for {
        fmt.Print(">> ")
        input, _ := reader.ReadString('\n')      
        input = strings.TrimSpace(input)    // Remove leading/trailing whitespace
        fmt.Fprintln(conn, input)          // Send player's input (attack choice, etc.) to the server
    }
}
