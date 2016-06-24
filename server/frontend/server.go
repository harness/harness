// +build ignore

// Use this server to run Polymer in standalone mode to avoid having to
// re-compile your Go code every and re-launch Drone every time you make a
// code change.
//
// This server proxies all traffic to a running Drone instance. This can be a
// local drone instance, or a remote drone instance, so you can develop against
// real world data.
//
//     go run server.go --scheme=http --host=drone.server.com --token=<token>
//

package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/drone/drone/client"
	"github.com/koding/websocketproxy"
)

var (
	host   = flag.String("host", "localhost:8000", "drone url")
	scheme = flag.String("scheme", "http", "drone url scheme. http or https")
	token  = flag.String("token", "", "drone api token from your user profile")
)

func main() {
	flag.Parse()

	// create the drone client and get the Drone user
	client := client.NewClientToken(*scheme+"://"+*host, *token)
	user, err := client.Self()
	if err != nil {
		log.Fatal(err)
	}
	userJson, err := json.Marshal(user)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Listening on port 9000")

	// serve the static html page
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		index, err := ioutil.ReadFile("index.html")
		if err != nil {
			log.Println(err)
		}
		if r.URL.Query().Get("guest") != "" {
			w.Write(index)
			return
		}
		out := strings.Replace(
			string(index),
			"window.STATE_FROM_SERVER={}",
			"window.STATE_FROM_SERVER={user:"+string(userJson)+"}", -1)
		w.Write([]byte(out))
	})

	// serve static content from the filesystem
	http.Handle("/src/",
		http.StripPrefix("/src/",
			http.FileServer(
				http.Dir("src/"),
			),
		),
	)
	// serve static content from the filesystem
	http.Handle("/bower_components/",
		http.StripPrefix("/bower_components/",
			http.FileServer(
				http.Dir("bower_components/"),
			),
		),
	)

	// proxy all websockets
	http.HandleFunc("/ws/", func(rw http.ResponseWriter, req *http.Request) {
		target, _ := url.Parse(req.URL.String())
		target.Host = *host
		target.Scheme = "ws"
		if *scheme == "https" {
			target.Scheme = "wss"
		}
		target.RawQuery = "access_token=" + *token
		websocketproxy.NewProxy(target).ServeHTTP(rw, req)
	})

	// proxy all requests to beta.drone.io
	http.Handle("/api/", &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			log.Println(req.Method, req.URL.Path)
			req.URL.Scheme = *scheme
			req.URL.Host = *host
			req.Host = *host
			req.Header.Set("X-Forwarded-For", *host)
			req.Header.Set("X-Forwarded-Proto", *scheme)
			req.Header.Set("Authorization", "Bearer "+*token)
		},
	})

	http.ListenAndServe(":9000", nil)
}
