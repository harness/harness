package pubsub

type Subscription struct {
	channel *Channel
	closed  chan bool
	send    chan interface{}
}

func NewSubscription(channel *Channel) *Subscription {
	return &Subscription{
		channel: channel,
		closed:  make(chan bool),
		send:    make(chan interface{}),
	}
}

func (s *Subscription) Read() <-chan interface{} {
	return s.send
}

func (s *Subscription) Close() {
	go func() { s.channel.unsubscribe <- s }()
	go func() { s.closed <- true }()
}

func (s *Subscription) CloseNotify() <-chan bool {
	return s.closed
}
