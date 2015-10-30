package broadcast

import (
	"sync"
	"testing"
)

func TestMuxBroadcast(t *testing.T) {
	wg := sync.WaitGroup{}

	mo := NewMuxObserver(0, 0)
	defer mo.Close()

	b1 := mo.Sub()
	defer b1.Close()

	b2 := mo.Sub()
	defer b2.Close()

	for i := 0; i < 5; i++ {
		wg.Add(2)

		cch1 := make(chan interface{})
		b1.Register(cch1)
		cch2 := make(chan interface{})
		b2.Register(cch2)

		go func() {
			defer wg.Done()
			defer b1.Unregister(cch1)
			<-cch1
		}()
		go func() {
			defer wg.Done()
			defer b2.Unregister(cch2)
			<-cch2
		}()

	}

	go b1.Submit(1)
	go b2.Submit(1)

	wg.Wait()
}

func TestMuxBroadcastCleanup(t *testing.T) {
	mo := NewMuxObserver(0, 0)
	b := mo.Sub()
	b.Register(make(chan interface{}))
	b.Close()
	mo.Close()
}

func BenchmarkMuxBrodcast(b *testing.B) {
	chout := make(chan interface{})

	mo := NewMuxObserver(0, 0)
	defer mo.Close()
	bc := mo.Sub()
	bc.Register(chout)

	for i := 0; i < b.N; i++ {
		bc.Submit(nil)
		<-chout
	}
}
