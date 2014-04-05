package main

import (
	"math/rand"
	"time"
)

// randomString generates a pseudo-random alpha-numeric string with given length.
func randomString(length int) string {
	rand.Seed(time.Now().UnixNano())
	k := make([]rune, length)
	for i := 0; i < length; i++ {
		c := rand.Intn(35)
		if c < 10 {
			c += 48 // numbers (0-9) (0+48 == 48 == '0', 9+48 == 57 == '9')
		} else {
			c += 87 // lower case alphabets (a-z) (10+87 == 97 == 'a', 35+87 == 122 = 'z')
		}
		k[i] = rune(c)
	}
	return string(k)
}
