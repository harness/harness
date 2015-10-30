package crypto

import (
	"testing"

	"github.com/franela/goblin"
	"github.com/square/go-jose"
)

func TestKeys(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Generate Key", func() {

		g.It("Generates a private key", func() {
			_, err := GeneratePrivateKey()
			g.Assert(err == nil).IsTrue()
		})
	})
}

func Test_Encrypt(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Secure", func() {

		g.It("Should encrypt a string", func() {
			ciphertext, err := Encrypt("top_secret", fakePriv)
			g.Assert(err == nil).IsTrue()

			object, _ := jose.ParseEncrypted(ciphertext)
			privKey, _ := decodePrivateKey(fakePriv)
			plaintext, _ := object.Decrypt(privKey)
			g.Assert(string(plaintext)).Equal("top_secret")
		})

	})
}

var fakePriv = `
-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEA71FaA+otDak2rXF/4h69Tz+OxS6NOWaOc/n7dinHXnlo3Toy
ZzvwweJGQKIOfPNBMncz+8h6oLOByFvb95Z1UEM0d+KCFCCutOeN9NNMw4fkUtSZ
7sm6T35wQUkDOiO1YAGy27hQfT7iryhPwA8KmgZmt7toNNf+WymPR8DMwAAYeqHA
5DIEWWsg+RLohOJ0itIk9q6Us9WYhng0sZ9+U+C87FospjKRMyAinSvKx0Uan4ap
YGbLjDQHimWtimfT4XWCGTO1cWno378Vm/newUN6WVaeZ2CSHcWgD2fWcjFixX2A
SvcvfuCo7yZPUPWeiYKrc5d1CC3ncocu43LhSQIDAQABAoIBAQDIbYKM+sfmxAwF
8KOg1gvIXjuNCrK+GxU9LmSajtzpU5cuiHoEGaBGUOJzaQXnQbcds9W2ji2dfxk3
my87SShRIyfDK9GzV7fZzIAIRhrpO1tOv713zj0aLJOJKcPpIlTZ5jJMcC4A5vTk
q0c3W6GOY8QNJohckXT2FnVoK6GPPiaZnavkwH33cJk0j1vMsbADdKF7Jdfq9FBF
Lx+Za7wo79MQIr68KEqsqMpmrawIf1T3TqOCNbkPCL2tu5EfoyGIItrH33SBOV/B
HbIfe4nJYZMWXhe3kZ/xCFqiRx6/wlc5pGCwCicgHJJe/l8Y9OticDCCyJDQtD8I
6927/j2NAoGBAPNRRY8r5ES5f8ftEktcLwh2zw08PNkcolTeqsEMbWAQspV/v+Ay
4niEXIN3ix2yTnMgrtxRGO7zdPnMaTN8E88FsSDKQ97lm7m3jo7lZtDMz16UxGmd
AOOuXwUtpngz7OrQ25NXhvFYLTgLoPsv3PbFbF1pwbhZqPTttTdg5so3AoGBAPvK
ta/n7DMZd/HptrkdkxxHaGN19ZjBVIqyeORhIDznEYjv9Z90JvzRxCmUriD4fyJC
/XSTytORa34UgmOk1XFtxWusXhnYqCTIHG/MKCy9D4ifzFzii9y/M+EnQIMb658l
+edLyrGFla+t5NS1XAqDYjfqpUFbMvU1kVoDJ/B/AoGBANBQe3o5PMSuAD19tdT5
Rnc7qMcPFJVZE44P2SdQaW/+u7aM2gyr5AMEZ2RS+7LgDpQ4nhyX/f3OSA75t/PR
PfBXUi/dm8AA2pNlGNM0ihMn1j6GpaY6OiG0DzwSulxdMHBVgjgijrCgKo66Pgfw
EYDgw4cyXR1k/ec8gJK6Dr1/AoGBANvmSY77Kdnm4E4yIxbAsX39DznuBzQFhGQt
Qk+SU6lc1H+Xshg0ROh/+qWl5/17iOzPPLPXb0getJZEKywDBTYu/D/xJa3E/fRB
oDQzRNLtuudDSCPG5wc/JXv53+mhNMKlU/+gvcEUPYpUgIkUavHzlI/pKbJOh86H
ng3Su8rZAn9w/zkoJu+n7sHta/Hp6zPTbvjZ1EijZp0+RygBgiv9UjDZ6D9EGcjR
ZiFwuc8I0g7+GRkgG2NbfqX5Cewb/nbJQpHPO31bqJrcLzU0KurYAwQVx6WGW0He
ERIlTeOMxVo6M0OpI+rH5bOLdLLEVhNtM/4HUFi1Qy6CCMbN2t3H
-----END RSA PRIVATE KEY-----
`
