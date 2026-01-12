package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	ampq "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril client...")

	connStr := "amqp://guest:guest@localhost:5672/"
	conn, err := ampq.Dial(connStr)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	log.Printf("Connection Successful!")

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Println("Something went wrong getting client username\n", err)
		return
	}

	_, _, err = pubsub.DeclareAndBind(
		conn,
		routing.ExchangePerilDirect,
		fmt.Sprintf("%s.%s", routing.PauseKey, username),
		routing.PauseKey,
		pubsub.Transient,
	)
	if err != nil {
		log.Println("Something went wrong declaring and binding queue\n", err)
		return
	}

	gameState := gamelogic.NewGameState(username)

	for {
		input := gamelogic.GetInput()
		if len(input) > 0 {

			switch strings.ToLower(input[0]) {
			case "spawn":
				err = gameState.CommandSpawn(input)

			case "move":
				_, err = gameState.CommandMove(input)
				if err == nil {
					fmt.Println("Move was successful")
				}

			case "status":
				gameState.CommandStatus()

			case "help":
				gamelogic.PrintClientHelp()

			case "spam":
				fmt.Println("Spamming not allowed yet!")

			case "quit":
				gamelogic.PrintQuit()
				return

			default:
				fmt.Println("Unknown command. Use 'help' for a list of commands")
			}

			if err != nil {
				fmt.Println(err)
				err = nil
			}
		}
	}
}
