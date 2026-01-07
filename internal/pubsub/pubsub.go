package pubsub

import (
	"context"
	"encoding/json"

	ampq "github.com/rabbitmq/amqp091-go"
)

func PublishJson[T any](ch *ampq.Channel, exchange, key string, val T) error {

	valJson, err := json.Marshal(val)
	if err != nil {
		return err
	}

	err = ch.PublishWithContext(context.Background(), exchange, key, false, false, ampq.Publishing{
		ContentType: "application/json",
		Body:        valJson,
	})

	if err != nil {
		return err
	}

	return nil
}
