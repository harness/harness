package bitbucketserver

import (
	"net/http"
	"bytes"
	"log"
	"io/ioutil"
	"fmt"
	"strings"
	"crypto/md5"
	log "github.com/Sirupsen/logrus"
)

func avatarLink(email string) (url string) {
	data := []byte(strings.ToLower(email))
	emailHash := md5.Sum(data)
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
