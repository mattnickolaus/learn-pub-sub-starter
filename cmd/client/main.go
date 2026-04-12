package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/mattnickolaus/learn-pub-sub-starter/internal/gamelogic"
	"github.com/mattnickolaus/learn-pub-sub-starter/internal/pubsub"
	"github.com/mattnickolaus/learn-pub-sub-starter/internal/routing"

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril client...")
	connStr := "amqp://guest:guest@localhost:5672/"

	rmq, err := amqp.Dial(connStr)
	if err != nil {
		log.Fatalf("Failed to connect to amqp with given connection stirng: %s", connStr)
	}
	defer rmq.Close()
	println("Connection successful!")

	userName, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalf("%v", err)
	}

	_, queue, err := pubsub.DeclareAndBind(
		rmq,
		routing.ExchangePerilDirect,
		routing.PauseKey+"."+userName,
		routing.PauseKey,
		pubsub.SimpleQueueType{
			Transient: true,
		},
	)
	if err != nil {
		log.Fatalf("Error during declare and bind: %v", err)
	}
	fmt.Printf("Queue %v declared and bound!\n", queue.Name)

	gameState := gamelogic.NewGameState(userName)

	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}

		switch words[0] {
		case "spawn":
			// "usage: spawn <location> <rank>"
			err := gameState.CommandSpawn(words)
			if err != nil {
				log.Printf("%v\n", err)
				log.Printf("\tPossible locations: americas, europe, africa, asia, antarctica, australia\n")
				log.Printf("\tPossible ranks: infantry, cavalry, artillery\n")
				continue
			}
		case "move":
			mv, err := gameState.CommandMove(words)
			if err != nil {
				log.Printf("%v\n", err)
				continue
			}
			if mv.ToLocation != "" {
				fmt.Printf("Success\n")
			}
			// TODO: publish the move
		case "status":
			gameState.CommandStatus()
		case "help":
			gamelogic.PrintClientHelp()
		case "spam":
			fmt.Printf("Spamming not allowed yet!\n")
			// TODO: publish a bunch of commands
		case "quit":
			gamelogic.PrintQuit()
			return
		default:
			fmt.Printf("Unknown command, please try again\n")
		}

	}

}
