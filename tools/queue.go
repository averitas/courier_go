package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type MessageHandler func([]byte) error

type IQueueManager interface {
	Send(interface{}) error
}

type RabbitMqManager struct {
	ConnString     string
	QueueName      string
	MessageChannel chan interface{}

	queue   *amqp.Queue
	channel *amqp.Channel
	conn    *amqp.Connection
}

func (r *RabbitMqManager) Init() error {
	var err error
	r.conn, err = amqp.Dial(r.ConnString)
	if err != nil {
		return fmt.Errorf("Init error when dial to server: %v", err)
	}
	r.MessageChannel = make(chan interface{}, 5)
	return nil
}

func (r *RabbitMqManager) Send(msg interface{}) error {
	select {
	case r.MessageChannel <- msg:
		return nil
	case <-time.After(time.Second):
		return fmt.Errorf("send message timeout, maybe too busy")
	}
}

func (r *RabbitMqManager) StartSender(ctx context.Context) {
Loop:
	for {
		select {
		case <-ctx.Done():
			break Loop
		case msg := <-r.MessageChannel:
			r.SendMessage(msg)
		}
	}
	fmt.Println("signal received, start to stop queue sender")
	r.reset()
	fmt.Println("background queue sender stopped")
}

func (r *RabbitMqManager) StartReceiver(ctx context.Context, handler MessageHandler) (err error) {
	defer func() {
		if rcy := recover(); rcy != nil {
			err = fmt.Errorf("send error: %v\n!panic: %v", err, rcy)
		}
	}()
	err = r.initQueue()

	msgs, err := r.channel.Consume(
		r.queue.Name, "consumer", true, false, false, false, nil,
	)
RLoop:
	for {
		select {
		case <-ctx.Done():
			break RLoop
		case msg := <-msgs:
			err = r.runWrapHandler(func() error {
				return handler(msg.Body)
			})
			if err != nil {
				fmt.Printf("call wrap handler of message [%v] error: %v\n", string(msg.Body), err)
			}
		}
	}

	fmt.Println("signal received, start to stop queue receiver")
	r.reset()
	fmt.Println("background queue receiver stopped")

	return
}

func (r *RabbitMqManager) runWrapHandler(handler func() error) (err error) {
	defer func() {
		if rcy := recover(); rcy != nil {
			err = fmt.Errorf("handler error: %v\n!panic: %v", err, rcy)
		}
	}()

	err = handler()
	return
}

func (r *RabbitMqManager) SendMessage(msg interface{}) error {
	if r.conn == nil {
		return fmt.Errorf("please init queue first")
	}

	msgBodyBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message to json string error: %v", err)
	}

	for i := 0; i < 3; i += 1 {
		err = r.initQueue()
		if err != nil {
			continue
		}
		err = r.sendInner(msgBodyBytes)
		if err == nil {
			return nil
		}

		r.reset()
	}

	return fmt.Errorf("Send failed with 3 tries %v", err)
}

func (r *RabbitMqManager) sendInner(message []byte) (err error) {
	defer func() {
		if rcy := recover(); rcy != nil {
			err = fmt.Errorf("send error: %v\n!panic: %v", err, rcy)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = r.channel.PublishWithContext(ctx, "", r.queue.Name, false, false, amqp.Publishing{
		ContentType: "text/json",
		Body:        message,
	})
	return
}

func (r *RabbitMqManager) initQueue() (err error) {
	if r.conn == nil {
		r.conn, err = amqp.Dial(r.ConnString)
		if err != nil {
			return fmt.Errorf("Init error when dial to server: %v", err)
		}
	}

	// init queue
	r.channel, err = r.conn.Channel()
	if err != nil {
		return fmt.Errorf("Init error when create channel: %v", err)
	}

	queue, err := r.channel.QueueDeclare(r.QueueName, false, false, false, false, nil)
	r.queue = &queue
	if err != nil {
		return fmt.Errorf("Init error when declare queue: %v", err)
	}
	return nil
}

func (r *RabbitMqManager) reset() {
	if r.channel != nil {
		r.channel.Close()
	}
	if r.conn != nil {
		r.conn.Close()
	}

	r.channel = nil
	r.conn = nil
}
