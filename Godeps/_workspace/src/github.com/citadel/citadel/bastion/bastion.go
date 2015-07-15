package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/citadel/citadel"
	"github.com/drone/drone/Godeps/_workspace/src/github.com/citadel/citadel/cluster"
	"github.com/drone/drone/Godeps/_workspace/src/github.com/citadel/citadel/scheduler"
	"github.com/gorilla/mux"
)

var (
	configPath     string
	config         *Config
	clusterManager *cluster.Cluster
)

func init() {
	flag.StringVar(&configPath, "conf", "", "config file")
	flag.Parse()
}

func destroy(w http.ResponseWriter, r *http.Request) {
	var container *citadel.Container
	if err := json.NewDecoder(r.Body).Decode(&container); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	if err := clusterManager.Kill(container, 9); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	if err := clusterManager.Remove(container); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func run(w http.ResponseWriter, r *http.Request) {
	var image *citadel.Image
	if err := json.NewDecoder(r.Body).Decode(&image); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	container, err := clusterManager.Start(image, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(container); err != nil {
		log.Println(err)
	}
}

func engines(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	if err := json.NewEncoder(w).Encode(config.Engines); err != nil {
		log.Println(err)
	}
}

func containers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	containers := clusterManager.ListContainers(false)
	if err := json.NewEncoder(w).Encode(containers); err != nil {
		log.Println(err)
	}
}

func main() {
	if err := loadConfig(); err != nil {
		log.Fatal(err)
	}

	tlsConfig, err := getTLSConfig()
	if err != nil {
		log.Fatal(err)
	}

	for _, d := range config.Engines {
		if err := setEngineClient(d, tlsConfig); err != nil {
			log.Fatal(err)
		}
	}

	clusterManager, err = cluster.New(scheduler.NewResourceManager(), config.Engines...)
	if err != nil {
		log.Fatal(err)
	}

	var (
		labelScheduler  = &scheduler.LabelScheduler{}
		uniqueScheduler = &scheduler.UniqueScheduler{}
		hostScheduler   = &scheduler.HostScheduler{}

		multiScheduler = scheduler.NewMultiScheduler(
			labelScheduler,
			uniqueScheduler,
		)
	)

	clusterManager.RegisterScheduler("service", labelScheduler)
	clusterManager.RegisterScheduler("unique", uniqueScheduler)
	clusterManager.RegisterScheduler("multi", multiScheduler)
	clusterManager.RegisterScheduler("host", hostScheduler)

	r := mux.NewRouter()
	r.HandleFunc("/containers", containers).Methods("GET")
	r.HandleFunc("/run", run).Methods("POST")
	r.HandleFunc("/destroy", destroy).Methods("DELETE")
	r.HandleFunc("/engines", engines).Methods("GET")

	log.Printf("bastion listening on %s\n", config.ListenAddr)

	if err := http.ListenAndServe(config.ListenAddr, r); err != nil {
		log.Fatal(err)
	}
}
