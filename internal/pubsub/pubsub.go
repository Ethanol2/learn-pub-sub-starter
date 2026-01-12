package pubsub

import (
	"context"
	"encoding/json"

	ampq "github.com/rabbitmq/amqp091-go"
)

type SimpleQueueType int

const (
	Durable   SimpleQueueType = iota
	Transient                 = 1
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

func DeclareAndBind(
	conn *ampq.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
) (*ampq.Channel, ampq.Queue, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, ampq.Queue{}, err
	}

	durable := isDurable(queueType)

	q, err := ch.QueueDeclare(
		queueName,
		durable,
		!durable,
		!durable,
		false,
		nil,
	)
	if err != nil {
		return nil, ampq.Queue{}, err
	}

	err = ch.QueueBind(queueName, key, exchange, false, nil)
	if err != nil {
		return nil, ampq.Queue{}, err
	}

	return ch, q, nil
}

func isDurable(qType SimpleQueueType) bool { return qType == Durable }
