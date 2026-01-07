package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

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

	pubsub.PublishJson(msgChan, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{IsPaused: true})

	// Wait for exit signal (ctrl+c)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan

	fmt.Println("")
	log.Println("Exiting")
}
