package secure

import (
	"testing"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/franela/goblin"
)

func Test_Secure(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Encrypt params", func() {

		key := "9T2tH3qZ8FSPr9uxrhzV4mn2VdVgA56xPVtYvCh0"
		text := "super_duper_secret"
		long := "-----BEGIN RSA PRIVATE KEY-----\nProc-Type: 4,ENCRYPTED\nDEK-Info: DES-EDE3-CBC,32495A90F3FF199D\nlrMAsSjjkKiRxGdgR8p5kZJj0AFgdWYa3OT2snIXnN5+/p7j13PSkseUcrAFyokc\nV9pgeDfitAhb9lpdjxjjuxRcuQjBfmNVLPF9MFyNOvhrprGNukUh/12oSKO9dFEt\ns39F/2h6Ld5IQrGt3gZaBB1aGO+tw3ill1VBy2zGPIDeuSz6DS3GG/oQ2gLSSMP4\nOVfQ32Oajo496iHRkdIh/7Hho7BNzMYr1GxrYTcE9/Znr6xgeSdNT37CCeCH8cmP\naEAUgSMTeIMVSpILwkKeNvBURic1EWaqXRgPRIWK0vNyOCs/+jNoFISnV4pu1ROF\n92vayHDNSVw9wHcdSQ75XSE4Msawqv5U1iI7e2lD64uo1qhmJdrPcXDJQCiDbh+F\nhQhF+wAoLRvMNwwhg+LttL8vXqMDQl3olsWSvWPs6b/MZpB0qwd1bklzA6P+PeAU\nsfOvTqi9edIOfKqvXqTXEhBP8qC7ZtOKLGnryZb7W04SSVrNtuJUFRcLiqu+w/F/\nMSxGSGalYpzIZ1B5HLQqISgWMXdbt39uMeeooeZjkuI3VIllFjtybecjPR9ZYQPt\nFFEP1XqNXjLFmGh84TXtvGLWretWM1OZmN8UKKUeATqrr7zuh5AYGAIbXd8BvweL\nPigl9ei0hTculPqohvkoc5x1srPBvzHrirGlxOYjW3fc4kDgZpy+6ik5k5g7JWQD\nlbXCRz3HGazgUPeiwUr06a52vhgT7QuNIUZqdHb4IfCYs2pQTLHzQjAqvVk1mm2D\nkh4myIcTtf69BFcu/Wuptm3NaKd1nwk1squR6psvcTXOWII81pstnxNYkrokx4r2\n7YVllNruOD+cMDNZbIG2CwT6V9ukIS8tl9EJp8eyb0a1uAEc22BNOjYHPF50beWF\nukf3uc0SA+G3zhmXCM5sMf5OxVjKr5jgcir7kySY5KbmG71omYhczgr4H0qgxYo9\nZyj2wMKrTHLfFOpd4OOEun9Gi3srqlKZep7Hj7gNyUwZu1qiBvElmBVmp0HJxT0N\nmktuaVbaFgBsTS0/us1EqWvCA4REh1Ut/NoA9oG3JFt0lGDstTw1j+orDmIHOmSu\n7FKYzr0uCz14AkLMSOixdPD1F0YyED1NMVnRVXw77HiAFGmb0CDi2KEg70pEKpn3\nksa8oe0MQi6oEwlMsAxVTXOB1wblTBuSBeaECzTzWE+/DHF+QQfQi8kAjjSdmmMJ\nyN+shdBWHYRGYnxRkTatONhcDBIY7sZV7wolYHz/rf7dpYUZf37vdQnYV8FpO1um\nYa0GslyRJ5GqMBfDS1cQKne+FvVHxEE2YqEGBcOYhx/JI2soE8aA8W4XffN+DoEy\nZkinJ/+BOwJ/zUI9GZtwB4JXqbNEE+j7r7/fJO9KxfPp4MPK4YWu0H0EUWONpVwe\nTWtbRhQUCOe4PVSC/Vv1pstvMD/D+E/0L4GQNHxr+xyFxuvILty5lvFTxoAVYpqD\nu8gNhk3NWefTrlSkhY4N+tPP6o7E4t3y40nOA/d9qaqiid+lYcIDB0cJTpZvgeeQ\nijohxY3PHruU4vVZa37ITQnco9az6lsy18vbU0bOyK2fEZ2R9XVO8fH11jiV8oGH\n-----END RSA PRIVATE KEY-----"

		g.It("Should encrypt a string", func() {
			encrypted, err := Encrypt(key, text)
			g.Assert(err == nil).IsTrue()
			decrypted, err := Decrypt(key, encrypted)
			g.Assert(err == nil).IsTrue()
			g.Assert(text).Equal(decrypted)
		})

		g.It("Should encrypt a long string", func() {
			encrypted, err := Encrypt(key, long)
			g.Assert(err == nil).IsTrue()
			decrypted, err := Decrypt(key, encrypted)
			g.Assert(err == nil).IsTrue()
			g.Assert(long).Equal(decrypted)
		})

		g.It("Should decrypt a map", func() {
			params := map[string]string{
				"foo": "dG0H-Kjg4lZ8s-4WwfaeAgAAAAAAAAAAAAAAAAAAAADKUC-q4zHKDHzH9qZYXjGl1S0=",
			}
			err := DecryptMap(key, params)
			g.Assert(err == nil).IsTrue()
			g.Assert(params["foo"]).Equal("super_duper_secret")
		})

		g.It("Should trim a key with blocksize greater than 32 bytes", func() {
			trimmed := trimKey("9T2tH3qZ8FSPr9uxrhzV4mn2VdVgA56x")
			g.Assert(len(key) > 32).IsTrue()
			g.Assert(len(trimmed)).Equal(32)
		})

		g.It("Should decrypt a yaml", func() {
			yaml := `secure: {"foo": "dG0H-Kjg4lZ8s-4WwfaeAgAAAAAAAAAAAAAAAAAAAADKUC-q4zHKDHzH9qZYXjGl1S0="}`
			decrypted, err := Parse(key, yaml)
			g.Assert(err == nil).IsTrue()
			g.Assert(decrypted["foo"]).Equal("super_duper_secret")
		})

		g.It("Should decrypt a yaml with no secure section", func() {
			yaml := `foo: bar`
			decrypted, err := Parse(key, yaml)
			g.Assert(err == nil).IsTrue()
			g.Assert(len(decrypted)).Equal(0)
		})

		g.It("Should encrypt a map", func() {
			params := map[string]string{
				"foo": text,
			}
			err := EncryptMap(key, params)
			g.Assert(err == nil).IsTrue()
			g.Assert(params["foo"] == "super_duper_secret").IsFalse()
			err = DecryptMap(key, params)
			g.Assert(err == nil).IsTrue()
			g.Assert(params["foo"] == "super_duper_secret").IsTrue()
		})
	})
}
