package poller

import (
	"bytes"
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/drone/drone/model"
	"github.com/drone/drone/shared/envconfig"
	"github.com/drone/drone/store"
	"github.com/jasonlvhit/gocron"
	"net/http"
	"strings"
)

var (
	instance *Poller
)

//Params poll parameters
type Params struct {
	Owner string `json:"owner"   binding:"required, gte=1"`
	Name  string `json:"name"    binding:"required, gte=1"`
	Force bool   `json:"force"`
}

//Poller git poller
type Poller struct {
	serverURL string
	store     store.Store
	scheduler *gocron.Scheduler
}

//Init initialize poller
func Init(env envconfig.Env, store store.Store) {
	log.Infoln("init... poller")
	serverAddr := env.String("SERVER_ADDR", ":8000")
	instance = &Poller{
		serverURL: formURL(serverAddr),
		store:     store,
		scheduler: gocron.NewScheduler(),
	}
	go func() {
		stopped := instance.scheduler.Start()
		instance.loadPolls()
		log.Infoln("poller started")
		<-stopped
		Init(env, store)
	}()
}

//Ref get scheduler reference
func Ref() *Poller {
	if instance == nil {
		log.Panicln("not initialized")
	}
	return instance
}

//Schedule poll
func (poller *Poller) AddPoll(repo *model.Repo, seconds uint64) {
	poll := model.Poll{
		Owner:  repo.Owner,
		Name:   repo.Name,
		Period: seconds,
	}
	err := poller.store.Polls().Create(&poll)
	if err != nil {
		log.Errorln("can't create poll", err)
	}
	poller.Schedule(repo.Owner, repo.Name, seconds)
}

func (poller *Poller) Schedule(owner, name string, seconds uint64) {
	poller.scheduler.Every(seconds).Seconds().Do(poller.pollGit, owner, name)
	log.Infoln("scheduled poll", owner, "-", name, "-", seconds)
}

//preload all polls from db
func (poller *Poller) loadPolls() {
	polls, err := poller.store.Polls().GetPollList()
	if err != nil {
		log.Errorln("get poll list err", err)
		return
	}
	for _, poll := range polls {
		poller.Schedule(poll.Owner, poll.Name, poll.Period)
		log.Infof("scheduled poll %q\n", poll)
	}
}

func (poller *Poller) pollGit(owner, name string) {
	url := fmt.Sprintf("%s/api/hook?owner=%s&name=%s", poller.serverURL, owner, name)
	params := Params{
		Owner: owner,
		Name:  name,
	}
	buf, err := json.Marshal(params)
	if err != nil {
		log.Errorln("can't encode params", err)
		return
	}
	bufReader := bytes.NewReader(buf)

	req, err := http.NewRequest("POST", url, bufReader)
	req.Header.Add("Content-Type", "application/json")
	client := &http.Client{}

	//resp, err := http.Post(url, "application/json", nil)
	log.Infoln("polling", url)
	log.Infof("polling req %q\n", req)
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		log.Infoln("hook is not 200", resp.StatusCode)
	}
	defer resp.Body.Close()
}

func formURL(addr string) string {
	slices := strings.Split(addr, ":")
	if len(slices) == 2 {
		if len(slices[0]) == 0 {
			return fmt.Sprintf("http://localhost:%s", slices[1])
		} else {
			return fmt.Sprintf("http://%s:%s", slices[0], slices[1])
		}
	} else {
		log.Panicln("bad addr", addr)
	}
	return ""
}
