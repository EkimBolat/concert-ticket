package main

import (
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

const eventsExchange = "order-events"

// eventPublisher wraps a RabbitMQ connection. If the broker isn't
// reachable at startup, it degrades gracefully: the order saga still
// works, it just doesn't publish events (useful for local dev without
// RabbitMQ running).
type eventPublisher struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

func newEventPublisher(url string) *eventPublisher {
	conn, err := amqp.Dial(url)
	if err != nil {
		log.Printf("warning: could not connect to RabbitMQ, events will not be published: %v", err)
		return &eventPublisher{}
	}
	ch, err := conn.Channel()
	if err != nil {
		log.Printf("warning: could not open RabbitMQ channel: %v", err)
		return &eventPublisher{conn: conn}
	}
	if err := ch.ExchangeDeclare(eventsExchange, "topic", true, false, false, false, nil); err != nil {
		log.Printf("warning: could not declare exchange: %v", err)
	}
	return &eventPublisher{conn: conn, ch: ch}
}

func (p *eventPublisher) publish(routingKey string, payload any) {
	if p.ch == nil {
		return
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return
	}
	if err := p.ch.Publish(eventsExchange, routingKey, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        b,
	}); err != nil {
		log.Printf("warning: failed to publish event %s: %v", routingKey, err)
	}
}
