package pubsub

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"log"

	ampq "github.com/rabbitmq/amqp091-go"
)

type SimpleQueueType int

const (
	Durable   SimpleQueueType = iota
	Transient                 = 1
)

type AckType int

const (
	Ack         AckType = iota
	NackRequeue         = 1
	NackDiscard         = 2
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

	durable := queueType == Durable
	table := ampq.Table{"x-dead-letter-exchange": "peril_dlx"}

	q, err := ch.QueueDeclare(
		queueName,
		durable,
		!durable,
		!durable,
		false,
		table,
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

func SubscribeJSON[T any](
	conn *ampq.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handle func(T) AckType,
) error {

	channel, _, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		return err
	}

	deliveryChan, err := channel.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	go func() {
		var err error
		for msgBytes := range deliveryChan {

			var msg T
			err = json.Unmarshal(msgBytes.Body, &msg)
			if err != nil {
				log.Println("Something went wrong unmarshalling message\n", err)
				continue
			}

			response := handle(msg)

			switch response {
			case Ack:
				//log.Println("Acknowledging message")
				err = msgBytes.Ack(false)
			case NackRequeue:
				//log.Println("Not Acknowledging message with requeue")
				err = msgBytes.Nack(false, true)
			case NackDiscard:
				//log.Println("Not Acknowledging message with discard")
				err = msgBytes.Nack(false, false)
			}
			if err != nil {
				log.Println("Something went wrong acknowledging message\n", err)
			}
		}
	}()

	return nil
}

func PublishGob[T any](ch *ampq.Channel, exchange, key string, val T) error {

	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(val)
	if err != nil {
		return err
	}

	err = ch.PublishWithContext(context.Background(), exchange, key, false, false, ampq.Publishing{
		ContentType: "application/gob",
		Body:        buffer.Bytes(),
	})

	if err != nil {
		return err
	}

	return nil
}
