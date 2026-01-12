package main

import (
	"fmt"
	"log"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	ampq "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril server...")

	connStr := "amqp://guest:guest@localhost:5672/"
	conn, err := ampq.Dial(connStr)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	log.Println("Connection Successful!")

	msgChan, err := conn.Channel()
	if err != nil {
		log.Println("Something went wrong while creating the message channel")
		log.Println(err)
		return
	}

	gamelogic.PrintServerHelp()

	for {
		input := gamelogic.GetInput()
		if len(input) > 0 {
			switch input[0] {
			case "pause":
				fmt.Println("Sending pause message")
				err = pubsub.PublishJson(msgChan, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{IsPaused: true})

			case "resume":
				fmt.Println("Sending resume message")
				err = pubsub.PublishJson(msgChan, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{IsPaused: false})

			case "quit":
				fmt.Println("Exiting")
				return

			default:
				fmt.Println("Unknown command")
			}

			if err != nil {
				fmt.Println("Something went wrong while publishing json\n", err)
			}
		}
	}
}
