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

type GameLog struct {
	Message string
	AckType pubsub.AckType
}

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
		log.Println("Something went wrong while creating direct queue\n", err)
		return
	}

	movesKey := fmt.Sprintf("%s.%s", routing.ArmyMovesPrefix, username)

	movesCh, _, err := pubsub.DeclareAndBind(
		conn,
		routing.ExchangePerilTopic,
		movesKey,
		fmt.Sprintf("%s.*", routing.ArmyMovesPrefix),
		pubsub.Transient,
	)
	if err != nil {
		log.Println("Something went wrong while creating moves queue\n", err)
		return
	}

	logsCh, _, err := pubsub.DeclareAndBind(
		conn,
		routing.ExchangePerilTopic,
		"game_logs",
		routing.GameLogSlug+".*",
		pubsub.Durable,
	)
	if err != nil {
		log.Println("Something went wrong while creating logs queue\n", err)
		return
	}

	gameState := gamelogic.NewGameState(username)

	err = pubsub.SubscribeJSON(conn, routing.ExchangePerilDirect, fmt.Sprintf("pause.%s", username), routing.PauseKey, pubsub.Transient, handlerPause(gameState))
	if err != nil {
		log.Println("Something went wrong subscribing to the pause channel\n", err)
		return
	}

	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		fmt.Sprintf("%s.%s", routing.ArmyMovesPrefix, username),
		fmt.Sprintf("%s.*", routing.ArmyMovesPrefix),
		pubsub.Transient,
		handlerMove(gameState, movesCh),
	)
	if err != nil {
		log.Println("Something went wrong subscribing to the move channel\n", err)
		return
	}

	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		"war",
		routing.WarRecognitionsPrefix+".*",
		pubsub.Durable,
		handlerWar(gameState, logsCh),
	)
	if err != nil {
		log.Println("Something went wrong subscribing to the war channel\n", err)
		return
	}

	for {
		input := gamelogic.GetInput()
		if len(input) > 0 {

			switch strings.ToLower(input[0]) {
			case "spawn":
				err = gameState.CommandSpawn(input)

			case "move":
				move, err := gameState.CommandMove(input)
				if err == nil {
					fmt.Println("Move was successful")
				}

				err = pubsub.PublishJson(movesCh, routing.ExchangePerilTopic, movesKey, move)
				if err == nil {
					fmt.Println("Move published")
				} else {
					log.Println("Something went wrong publishing move\n", err)
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

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.AckType {
	return func(state routing.PlayingState) pubsub.AckType {
		defer fmt.Print("> ")
		gs.HandlePause(state)
		return pubsub.Ack
	}
}
func handlerMove(gs *gamelogic.GameState, movesCh *ampq.Channel) func(gamelogic.ArmyMove) pubsub.AckType {
	return func(move gamelogic.ArmyMove) pubsub.AckType {
		defer fmt.Print("> ")
		outcome := gs.HandleMove(move)

		switch outcome {
		case gamelogic.MoveOutcomeMakeWar:
			//log.Println("Move outcome is war")

			err := pubsub.PublishJson(movesCh, routing.ExchangePerilTopic, fmt.Sprintf("%s.%s", routing.WarRecognitionsPrefix, gs.Player.Username), gamelogic.RecognitionOfWar{
				Attacker: move.Player,
				Defender: gs.GetPlayerSnap(),
			})
			if err != nil {
				log.Println("Something went wrong publishing the declaration of war... requeuing")
				return pubsub.NackRequeue
			}

			return pubsub.Ack

		case gamelogic.MoveOutComeSafe:
			//log.Println("Move outcome is safe")
			return pubsub.Ack

		case gamelogic.MoveOutcomeSamePlayer:
			//log.Println("Move outcome is same player")
			return pubsub.NackDiscard
		}

		return pubsub.NackDiscard
	}
}
func handlerWar(gs *gamelogic.GameState, logsCh *ampq.Channel) func(gamelogic.RecognitionOfWar) pubsub.AckType {

	return func(warRec gamelogic.RecognitionOfWar) pubsub.AckType {
		defer fmt.Print("> ")

		outcome, winner, loser := gs.HandleWar(warRec)

		outcomeLog := GameLog{}
		var ackType pubsub.AckType

		switch outcome {
		case gamelogic.WarOutcomeNotInvolved:
			outcomeLog.Message = fmt.Sprintf("A war happened between %s, who won, and %s", winner, loser)
			ackType = pubsub.NackRequeue

		case gamelogic.WarOutcomeNoUnits:
			outcomeLog.Message = fmt.Sprintf("Nothing happened in a war between %s and %s because there were no units", winner, loser)
			ackType = pubsub.NackDiscard

		case gamelogic.WarOutcomeOpponentWon:
			outcomeLog.Message = fmt.Sprintf("%s won a war against %s", winner, loser)
			ackType = pubsub.Ack

		case gamelogic.WarOutcomeYouWon:
			outcomeLog.Message = fmt.Sprintf("%s won a war against %s", winner, loser)
			ackType = pubsub.Ack

		case gamelogic.WarOutcomeDraw:
			outcomeLog.Message = fmt.Sprintf("A war between %s and %s resulted in a draw", winner, loser)
			ackType = pubsub.Ack

		default:
			log.Println("Unhandled war result")
			outcomeLog.Message = fmt.Sprintf("Unhandled war result in war between %s and %s", winner, loser)
			ackType = pubsub.NackDiscard
		}

		err := outcomeLog.publishLog(logsCh, routing.ExchangePerilTopic, gs.GetUsername())
		if err != nil {
			log.Println("Something went wrong while publishing a log\n", err)
			return pubsub.NackRequeue
		}

		return ackType
	}
}

func (log GameLog) publishLog(ch *ampq.Channel, exchange, username string) error {
	return pubsub.PublishGob(ch, exchange, routing.GameLogSlug+"."+username, log)
}
