package poller

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/carlescere/scheduler"
	"github.com/drone/drone/model"
	"github.com/drone/drone/shared/envconfig"
	"github.com/drone/drone/store"

	"net/http"
	"reflect"
	"runtime"
	"strings"
)

var (
	instance *Poller
)

var (
	//ErrBadPoll
	ErrBadPoll = errors.New("bad poll")
)

//Params poll parameters
type Params struct {
	Owner string `json:"owner"   binding:"required,gte=1"`
	Name  string `json:"name"    binding:"required,gte=1"`
	Force bool   `json:"force"`
}

//Poller git poller
type Poller struct {
	serverURL string
	store     store.Store
	jobs      map[int64]*scheduler.Job
}

//Init initialize poller
func Init(env envconfig.Env, store store.Store) {
	log.Infoln("init... poller")
	serverAddr := env.String("SERVER_ADDR", ":8000")
	instance = &Poller{
		serverURL: formURL(serverAddr),
		store:     store,
		jobs:      make(map[int64]*scheduler.Job, 256),
	}
	instance.loadPolls()
	log.Infoln("poller started")
}

//Ref get scheduler reference
func Ref() *Poller {
	if instance == nil {
		log.Panicln("not initialized")
	}
	return instance
}

//AddPoll add git poll
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
	poller.Schedule(&poll)
}

//Schedule schedule git poll
func (poller *Poller) Schedule(poll *model.Poll) {
	jobFn := func() {
		poller.pollGit(poll.Owner, poll.Name)
	}
	job, err := scheduler.Every(int(poll.Period)).Seconds().NotImmediately().Run(jobFn)
	if err != nil {
		log.Errorln("can't schedule job", err)
		return
	}

	poller.jobs[poll.ID] = job

	log.Infof("scheduled poll %q\n", *poll)
	log.Infof("scheduled jobs %q\n", poller.jobs)
	log.Infof("scheduled jobs len %d\n", len(poller.jobs))
}

//UpdatePoll update poll period
func (poller *Poller) UpdatePoll(repo *model.Repo, period uint64) error {
	store := poller.store.Polls()

	if period >= 300 {
		poll, err := store.Get(&model.Poll{Owner: repo.Owner, Name: repo.Name})
		if err != nil {
			log.Errorln("can't load poll", err)
			return err
		}
		if poll.Period != period {
			poll.Period = period
			err = store.Update(poll)
			if err != nil {
				log.Errorln("can't update poll", err)
				return err
			}

			if err = poller.RemoveJob(poll); err == nil {
				poller.Schedule(poll)
			}
		}
	}
	return nil
}

//RemoveJob remove scheduled job
func (poller *Poller) RemoveJob(poll *model.Poll) error {
	if poll == nil || poll.ID < 1 {
		return ErrBadPoll
	}

	if job, ok := poller.jobs[poll.ID]; ok {
		log.Infoln("removing job", poll.ID)
		job.Quit <- true
		delete(poller.jobs, poll.ID)
		return nil
	} else {
		log.Errorf("not found job in map %q\n", poller.jobs)
	}
	log.Infof("removed job %q\n", *poll)

	return nil
}

//DeletePoll delete git poll
func (poller *Poller) DeletePoll(repo *model.Repo) error {
	store := poller.store.Polls()

	poll, err := store.Get(&model.Poll{Owner: repo.Owner, Name: repo.Name})
	if err != nil {
		log.Errorln("can't load poll", err)
		return err
	}

	err = store.Delete(poll)
	if err != nil {
		log.Errorln("can't delete poll", err)
		return err
	}

	err = poller.RemoveJob(poll)
	if err != nil {
		return err
	}
	return nil
}

//preload all polls from db
func (poller *Poller) loadPolls() {
	polls, err := poller.store.Polls().List()
	if err != nil {
		log.Errorln("get poll list err", err)
		return
	}
	for _, poll := range polls {
		poller.Schedule(poll)
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

// for given function fn , get the name of funciton.
func fnName(fn interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf((fn)).Pointer()).Name()
}
