package main

import (
	"fmt"
	"log"

	"github.com/mattnickolaus/learn-pub-sub-starter/internal/gamelogic"
	"github.com/mattnickolaus/learn-pub-sub-starter/internal/pubsub"
	"github.com/mattnickolaus/learn-pub-sub-starter/internal/routing"

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

	ch, err := rmq.Channel()
	if err != nil {
		log.Fatalf("Failed to connect to amqp with given connection stirng: %s", connStr)
	}

	gamelogic.PrintServerHelp()

	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}

		switch words[0] {
		case "pause":
			fmt.Printf("Pausing...\n")
			err := pubsub.PublishJSON(
				ch,
				routing.ExchangePerilDirect,
				routing.PauseKey,
				routing.PlayingState{IsPaused: true},
			)
			if err != nil {
				log.Printf("could not publish time: %v", err)
			}
		case "resume":
			fmt.Printf("Resuming...\n")
			err := pubsub.PublishJSON(
				ch,
				routing.ExchangePerilDirect,
				routing.PauseKey,
				routing.PlayingState{IsPaused: false},
			)
			if err != nil {
				log.Printf("could not publish time: %v", err)
			}
		case "quit":
			log.Printf("Exiting...\n")
			return
		default:
			fmt.Printf("Unknown command, please try again")
		}

	}
}
