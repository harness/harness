package main

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/harness/gitness/internal/gitrpc/lock"
)

func main() {
	c := &lock.Mutex{}
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		log.Printf("1. Try to lock key: simple")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		lock, err := c.AcquireLock(ctx, "simple")
		if err != nil {
			log.Println(err)
			return
		}
		log.Printf("1. Lock key: simple")
		time.Sleep(time.Second * 10)
		lock.Release()
		log.Printf("1. Release key: simple")
	}()

	go func() {
		defer wg.Done()
		log.Printf("2. Try to lock key: simple")
		lock, err := c.AcquireLock(context.Background(), "simple")
		if err != nil {
			log.Println(err)
			return
		}
		log.Printf("2. Lock key: simple")
		time.Sleep(time.Second * 20)
		lock.Release()
		log.Printf("2. Release key: simple")
	}()

	go func() {
		defer wg.Done()
		log.Printf("3. Try to lock key: simple")
		lock, err := c.AcquireLock(context.Background(), "simple")
		if err != nil {
			log.Println(err)
			return
		}

		log.Printf("4. Try to lock key: simple")
		lck, err := c.AcquireLock(context.Background(), "simple")
		if err != nil {
			log.Println(err)
			return
		}
		log.Printf("4. Lock key: simple")
		time.Sleep(time.Second * 10)
		lck.Release()
		log.Printf("4. Release key: simple")

		log.Printf("3. Lock key: simple")
		time.Sleep(time.Second * 20)
		lock.Release()
		log.Printf("3. Release key: simple")
	}()

	wg.Wait()
}
