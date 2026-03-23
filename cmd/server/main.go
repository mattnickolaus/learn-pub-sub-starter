package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril server...")

	connStr := "amqp://guest:guest@localhost:5672/"
	rmq, err := amqp.Dial(connStr)
	if err != nil {
		log.Fatalf("Failed to connect to amqp with given connection stirng: %s", connStr)
	}
	defer rmq.Close()
	println("Connection successful!")

	// Create a channel to read the os.Signal to wait for Ctrl+C interrupt
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	s := <-signalChan
	fmt.Printf("\nReceived signal: %s\n", s)
	fmt.Printf("Program shutting down...\n")
	os.Exit(0)
}
