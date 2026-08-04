package main

import (
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	eventsExchange = "order-events"
	queueName      = "notification-service"
)

// startConsumer connects to RabbitMQ, declares a queue bound to every
// "order.*" routing key on the order-events exchange, and logs each event
// it receives. In a real system this is where you'd send an email or push
// notification instead of a log line. Degrades gracefully if RabbitMQ
// isn't reachable, same as the Order Service's publisher.
func startConsumer(url string) {
	conn, err := amqp.Dial(url)
	if err != nil {
		log.Printf("warning: could not connect to RabbitMQ, notifications disabled: %v", err)
		return
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Printf("warning: could not open RabbitMQ channel: %v", err)
		return
	}

	if err := ch.ExchangeDeclare(eventsExchange, "topic", true, false, false, false, nil); err != nil {
		log.Printf("warning: could not declare exchange: %v", err)
		return
	}

	q, err := ch.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		log.Printf("warning: could not declare queue: %v", err)
		return
	}

	if err := ch.QueueBind(q.Name, "order.*", eventsExchange, false, nil); err != nil {
		log.Printf("warning: could not bind queue: %v", err)
		return
	}

	msgs, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		log.Printf("warning: could not start consuming: %v", err)
		return
	}

	go func() {
		for msg := range msgs {
			handleEvent(msg.RoutingKey, msg.Body)
		}
	}()

	log.Printf("notification: listening for order events (queue=%q)", queueName)
}

func handleEvent(routingKey string, body []byte) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Printf("notification: failed to parse event %s: %v", routingKey, err)
		return
	}

	switch routingKey {
	case "order.completed":
		log.Printf("notification: order #%v confirmed for user %v -- sending confirmation email", payload["orderId"], payload["userId"])
	case "order.failed":
		log.Printf("notification: order #%v failed (%v) -- sending failure notice", payload["orderId"], payload["reason"])
	default:
		log.Printf("notification: unhandled event %s: %v", routingKey, payload)
	}
}
