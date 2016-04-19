package bitbucketserver

import (
	"net/http"
	"crypto/tls"
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	"encoding/pem"
	"crypto/x509"
	"github.com/mrjones/oauth"
)


func NewClient(ConsumerKey string, URL string) *oauth.Consumer{
	privateKeyFileContents, err := ioutil.ReadFile("/private_key.pem")
	log.Info("Tried to read the key")
	if err != nil {
		log.Fatal(err)
	}

	block, _ := pem.Decode([]byte(privateKeyFileContents))
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		log.Fatal(err)
	}

	c := oauth.NewRSAConsumer(
		ConsumerKey,
		privateKey,
		oauth.ServiceProvider{
			RequestTokenUrl:   URL + "/plugins/servlet/oauth/request-token",
			AuthorizeTokenUrl: URL + "/plugins/servlet/oauth/authorize",
			AccessTokenUrl:    URL + "/plugins/servlet/oauth/access-token",
			HttpMethod:        "POST",
		})
	c.HttpClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	return c
}

func NewClientWithToken(ConsumerKey string, URL string, AccessToken string) *http.Client{
	NewClient(ConsumerKey, URL)
	c := NewClient(ConsumerKey, URL)

	var token oauth.AccessToken
	token.Token = AccessToken
	client, err := c.MakeHttpClient(&token)
	if err != nil {
		log.Fatal(err)
	}
	return client
}






