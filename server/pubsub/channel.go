package pubsub

import (
	"log"
	"time"
)

type Channel struct {
	record        bool
	history       []interface{}
	timeout       time.Duration
	closed        chan bool
	broadcast     chan interface{}
	subscribe     chan *Subscription
	unsubscribe   chan *Subscription
	subscriptions map[*Subscription]bool
}

func NewChannel(opts *Opts) *Channel {
	return &Channel{
		timeout:       opts.Timeout,
		record:        opts.Record,
		history:       make([]interface{}, 0),
		closed:        make(chan bool),
		broadcast:     make(chan interface{}),
		subscribe:     make(chan *Subscription),
		unsubscribe:   make(chan *Subscription),
		subscriptions: make(map[*Subscription]bool),
	}
}

func (c *Channel) Publish(data interface{}) {
	go func() { c.broadcast <- data }()
}

func (c *Channel) Subscribe() *Subscription {
	s := NewSubscription(c)
	c.subscribe <- s
	return s
}

func (c *Channel) Close() {
	go func() { c.closed <- true }()
}

func (c *Channel) start() {
	// make sure we don't bring down the application
	// if somehow we encounter a nil pointer or some
	// other unexpected behavior.
	defer func() {
		if r := recover(); r != nil {
			log.Println("recoved from panic", r)
		}
	}()

	// timeout the channel after N duration
	// ignore the timeout if set to 0
	var timeout <-chan time.Time
	if c.timeout > 0 {
		timeout = time.After(c.timeout)
	}

	for {
		select {

		case sub := <-c.unsubscribe:
			delete(c.subscriptions, sub)
			close(sub.send)

		case sub := <-c.subscribe:
			c.subscriptions[sub] = true

			// if we are recording the output
			// we should send it to the subscriber
			// upon first connecting.
			if c.record && len(c.history) > 0 {
				history := make([]interface{}, len(c.history))
				copy(history, c.history)
				go replay(sub, history)
			}

		case msg := <-c.broadcast:
			// if we are recording the output, append
			// the message to the history
			if c.record {
				c.history = append(c.history, msg)
			}

			// loop through each subscription and
			// send the message.
			for sub := range c.subscriptions {
				select {
				case sub.send <- msg:
					// do nothing
					//default:
					//	log.Println("subscription closed in inner select")
					//	sub.Close()
				}
			}

		case <-timeout:
			log.Println("subscription's timedout channel received message")
			c.Close()

		case <-c.closed:
			log.Println("subscription's close channel received message")
			c.stop()
			return
		}
	}
}

func replay(s *Subscription, history []interface{}) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("recoved from panic", r)
		}
	}()

	for _, msg := range history {
		s.send <- msg
	}
}

func (c *Channel) stop() {
	for sub := range c.subscriptions {
		sub.Close()
	}
}
