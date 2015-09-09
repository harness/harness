package jwt_test

import (
	"crypto/ecdsa"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/dgrijalva/jwt-go"
)

var ecdsaTestData = []struct {
	name        string
	keys        map[string]string
	tokenString string
	alg         string
	claims      map[string]interface{}
	valid       bool
}{
	{
		"Basic ES256",
		map[string]string{"private": "test/ec256-private.pem", "public": "test/ec256-public.pem"},
		"eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJmb28iOiJiYXIifQ.MEQCIHoSJnmGlPaVQDqacx_2XlXEhhqtWceVopjomc2PJLtdAiAUTeGPoNYxZw0z8mgOnnIcjoxRuNDVZvybRZF3wR1l8w",
		"ES256",
		map[string]interface{}{"foo": "bar"},
		true,
	},
	{
		"Basic ES384",
		map[string]string{"private": "test/ec384-private.pem", "public": "test/ec384-public.pem"},
		"eyJhbGciOiJFUzM4NCIsInR5cCI6IkpXVCJ9.eyJmb28iOiJiYXIifQ.MGUCMQCHBr61FXDuFY9xUhyp8iWQAuBIaSgaf1z2j_8XrKcCfzTPzoSa3SZKq-m3L492xe8CMG3kafRMeuaN5Aw8ZJxmOLhkTo4D3-LaGzcaUWINvWvkwFMl7dMC863s0gov6xvXuA",
		"ES384",
		map[string]interface{}{"foo": "bar"},
		true,
	},
	{
		"Basic ES512",
		map[string]string{"private": "test/ec512-private.pem", "public": "test/ec512-public.pem"},
		"eyJhbGciOiJFUzUxMiIsInR5cCI6IkpXVCJ9.eyJmb28iOiJiYXIifQ.MIGIAkIAmVKjdJE5lG1byOFgZZVTeNDRp6E7SNvUj0UrvpzoBH6nrleWVTcwfHzbwWuooNpPADDSFR_Ql3ze-Vwwi8hBqQsCQgHn-ZooL8zegkOVeEEsqd7WHWdhb8UekFCYw3X8JnNP-D3wvZQ1-tkkHakt5gZ2-xO29TxfSPun4ViGkMYa7Q4N-Q",
		"ES512",
		map[string]interface{}{"foo": "bar"},
		true,
	},
	{
		"basic ES256 invalid: foo => bar",
		map[string]string{"private": "test/ec256-private.pem", "public": "test/ec256-public.pem"},
		"eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJmb28iOiJiYXIifQ.MEQCIHoSJnmGlPaVQDqacx_2XlXEhhqtWceVopjomc2PJLtdAiAUTeGPoNYxZw0z8mgOnnIcjoxRuNDVZvybRZF3wR1l8W",
		"ES256",
		map[string]interface{}{"foo": "bar"},
		false,
	},
}

func TestECDSAVerify(t *testing.T) {
	for _, data := range ecdsaTestData {
		var err error

		key, _ := ioutil.ReadFile(data.keys["public"])

		var ecdsaKey *ecdsa.PublicKey
		if ecdsaKey, err = jwt.ParseECPublicKeyFromPEM(key); err != nil {
			t.Errorf("Unable to parse ECDSA public key: %v", err)
		}

		parts := strings.Split(data.tokenString, ".")

		method := jwt.GetSigningMethod(data.alg)
		err = method.Verify(strings.Join(parts[0:2], "."), parts[2], ecdsaKey)
		if data.valid && err != nil {
			t.Errorf("[%v] Error while verifying key: %v", data.name, err)
		}
		if !data.valid && err == nil {
			t.Errorf("[%v] Invalid key passed validation", data.name)
		}
	}
}

func TestECDSASign(t *testing.T) {
	for _, data := range ecdsaTestData {
		var err error
		key, _ := ioutil.ReadFile(data.keys["private"])

		var ecdsaKey *ecdsa.PrivateKey
		if ecdsaKey, err = jwt.ParseECPrivateKeyFromPEM(key); err != nil {
			t.Errorf("Unable to parse ECDSA private key: %v", err)
		}

		if data.valid {
			parts := strings.Split(data.tokenString, ".")
			method := jwt.GetSigningMethod(data.alg)
			sig, err := method.Sign(strings.Join(parts[0:2], "."), ecdsaKey)
			if err != nil {
				t.Errorf("[%v] Error signing token: %v", data.name, err)
			}
			if sig == parts[2] {
				t.Errorf("[%v] Identical signatures\nbefore:\n%v\nafter:\n%v", data.name, parts[2], sig)
			}
		}
	}
}
