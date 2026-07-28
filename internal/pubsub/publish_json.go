package pubsub

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

func PublishGob[T any](ch *amqp.Channel, exchange, key string, val T) error {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(val)
	if err != nil {
		return err
	}

	err = ch.PublishWithContext(
		context.Background(),
		exchange,
		key,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/gob",
			Body:        buf.Bytes(),
		},
	)
	if err != nil {
		return err
	}

	return nil
}

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	data, err := json.Marshal(val)
	if err != nil {
		return err
	}

	err = ch.PublishWithContext(
		context.Background(),
		exchange,
		key,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        data,
		},
	)
	if err != nil {
		return err
	}

	return nil
}

type SimpleQueueType struct {
	Durable   bool
	Transient bool
}

type AckType int

const (
	Ack         AckType = iota // 0 and then increments
	NackRequeue                // 1
	NackDiscard                // 2
)

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType, // SimpleQueueType is an "enum" type I made to represent "durable" or "transient"
) (*amqp.Channel, amqp.Queue, error) {

	ch, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	args := make(amqp.Table)
	args["x-dead-letter-exchange"] = "peril_dlx"

	q, err := ch.QueueDeclare(queueName, queueType.Durable, queueType.Transient, queueType.Transient, false, args)
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	err = ch.QueueBind(queueName, key, exchange, false, nil)
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	return ch, q, nil
}

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) AckType,
) error {
	err := subscribe(
		conn,
		exchange,
		queueName,
		key,
		queueType,
		handler,
		JSONUnmarshaller,
	)
	if err != nil {
		return err
	}

	return nil
}

func SubscribeGob[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) AckType,
) error {
	err := subscribe(
		conn,
		exchange,
		queueName,
		key,
		queueType,
		handler,
		gobUnmarshaller,
	)
	if err != nil {
		return err
	}

	return nil
}

func gobUnmarshaller[T any](data []byte) (T, error) {
	reader := bytes.NewBuffer(data)
	dec := gob.NewDecoder(reader)

	var value T

	err := dec.Decode(&value)
	if err != nil {
		return value, err
	}

	return value, nil
}

func JSONUnmarshaller[T any](data []byte) (T, error) {
	var value T

	err := json.Unmarshal(data, &value)
	if err != nil {
		return value, err
	}

	return value, nil
}

func subscribe[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) AckType,
	unmarshaller func([]byte) (T, error),
) error {
	channel, queue, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		return err
	}

	err = channel.Qos(10, 0, true)
	if err != nil {
		return err
	}

	deliveryChan, err := channel.Consume(queue.Name, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	go func() {
		for message := range deliveryChan {
			data, err := unmarshaller(message.Body)
			if err != nil {
				fmt.Printf("Could not Unmarshal message: %v\n", err)
				continue
			}

			ack := handler(data)

			switch ack {
			case Ack:
				err = message.Ack(false)
				if err != nil {
					fmt.Printf("Could not Ack(nowledge) message: %v\n> ", err)
					return
				}
			case NackRequeue:
				err = message.Nack(false, true)
				if err != nil {
					fmt.Printf("Could not Nack message for requeue: %v\n> ", err)
					return
				}
			case NackDiscard:
				err = message.Nack(false, false)
				if err != nil {
					fmt.Printf("Could not Nack message and discard: %v\n> ", err)
					return
				}
			}
		}
	}()

	return nil
}
