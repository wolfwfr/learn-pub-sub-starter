package pubsub

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	bytes, err := json.Marshal(&val)
	if err != nil {
		return fmt.Errorf("marshalling val: %w", err)
	}
	err = ch.PublishWithContext(context.Background(), exchange, key, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        bytes,
	})
	if err != nil {
		return fmt.Errorf("publishing: %w", err)
	}
	return nil
}

type SimpleQueueType string

const (
	Durable   SimpleQueueType = "durable"
	Transient SimpleQueueType = "transient"
)

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType, // an enum to represent "durable" or "transient"
) (*amqp.Channel, amqp.Queue, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("creating new channel: %w", err)
	}
	isDurable := queueType == Durable
	isTransient := queueType == Transient
	queue, err := ch.QueueDeclare(queueName, isDurable, isTransient, isTransient, false, nil)
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("creating queue")
	}
	ch.QueueBind(queue.Name, key, exchange, false, nil)
	return ch, queue, nil
}

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType, // an enum to represent "durable" or "transient"
	handler func(T),
) error {
	ch, q, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		return fmt.Errorf("declaring & binding queue: %w", err)
	}
	c, err := ch.Consume(q.Name, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("creating consumer: %w", err)
	}
	go func() {
		for item := range c {
			var m T
			if err := json.Unmarshal(item.Body, &m); err != nil {
				fmt.Printf("failed unmarshal: %+v\n", err)
			}
			handler(m)
			if err := item.Ack(false); err != nil {
				fmt.Printf("failed message ack: %+v\n", err)
			}
		}
	}()
	return nil
}
