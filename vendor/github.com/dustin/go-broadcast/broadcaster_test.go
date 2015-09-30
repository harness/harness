package broadcast

import (
	"sync"
	"testing"
)

func TestBroadcast(t *testing.T) {
	wg := sync.WaitGroup{}

	b := NewBroadcaster(100)
	defer b.Close()

	for i := 0; i < 5; i++ {
		wg.Add(1)

		cch := make(chan interface{})

		b.Register(cch)

		go func() {
			defer wg.Done()
			defer b.Unregister(cch)
			<-cch
		}()

	}

	b.Submit(1)

	wg.Wait()
}

func TestBroadcastCleanup(t *testing.T) {
	b := NewBroadcaster(100)
	b.Register(make(chan interface{}))
	b.Close()
}

func echoer(chin, chout chan interface{}) {
	for m := range chin {
		chout <- m
	}
}

func BenchmarkDirectSend(b *testing.B) {
	chout := make(chan interface{})
	chin := make(chan interface{})
	defer close(chin)

	go echoer(chin, chout)

	for i := 0; i < b.N; i++ {
		chin <- nil
		<-chout
	}
}

func BenchmarkBrodcast(b *testing.B) {
	chout := make(chan interface{})

	bc := NewBroadcaster(0)
	defer bc.Close()
	bc.Register(chout)

	for i := 0; i < b.N; i++ {
		bc.Submit(nil)
		<-chout
	}
}

func BenchmarkParallelDirectSend(b *testing.B) {
	chout := make(chan interface{})
	chin := make(chan interface{})
	defer close(chin)

	go echoer(chin, chout)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			chin <- nil
			<-chout
		}
	})
}

func BenchmarkParallelBrodcast(b *testing.B) {
	chout := make(chan interface{})

	bc := NewBroadcaster(0)
	defer bc.Close()
	bc.Register(chout)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			bc.Submit(nil)
			<-chout
		}
	})
}
