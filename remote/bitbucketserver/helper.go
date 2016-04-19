package bitbucketserver

import (
	"net/http"
	"bytes"
	"io/ioutil"
	"fmt"
	"strings"
	"crypto/md5"
	log "github.com/Sirupsen/logrus"
	"encoding/hex"
)

func avatarLink(email string) (url string) {
	hasher := md5.New()
	hasher.Write([]byte(strings.ToLower(email)))
	emailHash := fmt.Sprintf("%v", hex.EncodeToString(hasher.Sum(nil)))
	avatarURL := fmt.Sprintf("http://www.gravatar.com/avatar/%s.jpg",emailHash)
	log.Info(avatarURL)
	return avatarURL
}

func doPut(client *http.Client, url string, body []byte) {
	request, err := http.NewRequest("PUT", url, bytes.NewBuffer(body))
	request.Header.Add("Content-Type","application/json")
	response, err := client.Do(request)
	if err != nil {
		log.Fatal(err)
	} else {
		defer response.Body.Close()
		contents, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("The calculated length is:", len(string(contents)), "for the url:", url)
		fmt.Println("   ", response.StatusCode)
		hdr := response.Header
		for key, value := range hdr {
			fmt.Println("   ", key, ":", value)
		}
		fmt.Println(string(contents))
	}
}

func doDelete(client *http.Client, url string) {
	request, err := http.NewRequest("DELETE", url, nil)
	response, err := client.Do(request)
	if err != nil {
		log.Fatal(err)
	} else {
		defer response.Body.Close()
		contents, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("The calculated length is:", len(string(contents)), "for the url:", url)
		fmt.Println("   ", response.StatusCode)
		hdr := response.Header
		for key, value := range hdr {
			fmt.Println("   ", key, ":", value)
		}
		fmt.Println(string(contents))
	}
}
