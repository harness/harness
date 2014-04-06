package bitbucket

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
)

// Bitbucket POSTs to the service URL you specify. The service
// receives an POST whenever user pushes to the repository.
const BrokerTypePost = "POST"

const BrokerTypePullRequestPost = "Pull Request POST"

type Broker struct {
	// A Bitbucket assigned integer representing a unique
	// identifier for the service.
	Id int `json:"id"`

	// A profile describing the service.
	Profile *BrokerProfile `json:"service"`
}

type BrokerProfile struct {
	// One of the supported services. The type is a
	// case-insensitive value.
	Type string `json:"type"`

	// A parameter array containing a name and value pair
	// for each parameter associated with the service.
	Fields []*BrokerField `json:"fields"`
}

type BrokerField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Bitbucket offers integration with external services by allowing a set of
// services, or brokers, to run at certain events.
//
// The Bitbucket services resource provides functionality for adding,
// removing, and configuring brokers on your repositories.  All the methods
// on this resource require the caller to authenticate.
//
// https://confluence.atlassian.com/display/BITBUCKET/services+Resource
type BrokerResource struct {
	client *Client
}

func (r *BrokerResource) List(owner, slug string) ([]*Broker, error) {
	brokers := []*Broker{}
	path := fmt.Sprintf("/repositories/%s/%s/services", owner, slug)

	if err := r.client.do("GET", path, nil, nil, &brokers); err != nil {
		return nil, err
	}

	return brokers, nil
}

func (r *BrokerResource) Find(owner, slug string, id int) (*Broker, error) {
	brokers := []*Broker{}
	path := fmt.Sprintf("/repositories/%s/%s/services/%v", owner, slug, id)

	if err := r.client.do("GET", path, nil, nil, &brokers); err != nil {
		return nil, err
	}

	if len(brokers) == 0 {
		return nil, ErrNotFound
	}

	return brokers[0], nil
}

func (r *BrokerResource) FindUrl(owner, slug, link, brokerType string) (*Broker, error) {
	brokers, err := r.List(owner, slug)
	if err != nil {
		return nil, err
	}

	//fmt.Println("Total borkers:", len(brokers))

	// iterate though list of brokers
	for _, broker := range brokers {
		if broker.Profile != nil && broker.Profile.Fields != nil {
			//fmt.Printf("Compare: %v == %v is %v\n", broker.Profile.Type, brokerType, broker.Profile.Type == brokerType)
			if broker.Profile.Type != brokerType {
				continue
			}
			// iterate through list of fields
			for _, field := range broker.Profile.Fields {
				//fmt.Printf("-->%v==%v\n", field.Name, field.Value)
				if field.Name == "URL" && field.Value == link {
					//	fmt.Println("Found match")
					return broker, nil
				}
			}
			//fmt.Println("Skipping...")
		}
	}

	//fmt.Println("Not match found")
	return nil, ErrNotFound
}

func (r *BrokerResource) Create(owner, slug, link, brokerType string) (*Broker, error) {
	values := url.Values{}
	values.Add("type", brokerType)
	values.Add("URL", link)

	if brokerType == BrokerTypePullRequestPost {
		values.Add("comments", "off")
		// TODO: figure out how to only set the hook for pull request create and update
		// This is the sample string from the docs...it does not work becuase of
		// the stupid slashes
		// Check out the docs: https://confluence.atlassian.com/display/BITBUCKET/Pull+Request+POST+hook+management
		// Sample string:
		// -data "type=POST&URL=https://www.test.comcreate%2Fedit%2Fmerge%2Fdecline=on&comments=on&approve%2Funapprove=on"
	}

	b := Broker{}
	path := fmt.Sprintf("/repositories/%s/%s/services", owner, slug)
	if err := r.client.do("POST", path, nil, values, &b); err != nil {
		return nil, err
	}

	return &b, nil
}

func (r *BrokerResource) Update(owner, slug, link, brokerType string, id int) (*Broker, error) {
	values := url.Values{}
	values.Add("type", brokerType)
	values.Add("URL", link)

	if brokerType == BrokerTypePullRequestPost {
		values.Add("comments", "off")
		// TODO: figure out how to also shutoff other events!
	}

	b := Broker{}
	path := fmt.Sprintf("/repositories/%s/%s/services/%v", owner, slug, id)
	if err := r.client.do("PUT", path, nil, values, &b); err != nil {
		return nil, err
	}

	return &b, nil
}

// CreateUpdate will attempt to Create a Broker (Server Hook) if
// it doesn't already exist in the Bitbucket.
func (r *BrokerResource) CreateUpdate(owner, slug, link, brokerType string) (*Broker, error) {
	if found, err := r.FindUrl(owner, slug, link, brokerType); err == nil {
		// if the Broker already exists, just return it
		// ... not need to re-create
		//fmt.Println("Broker already found, skipping!", brokerType)
		return found, nil
	}

	return r.Create(owner, slug, link, brokerType)
}

func (r *BrokerResource) Delete(owner, slug string, id int) error {
	path := fmt.Sprintf("/repositories/%s/%s/services/%v", owner, slug, id)
	return r.client.do("DELETE", path, nil, nil, nil)
}

func (r *BrokerResource) DeleteUrl(owner, slug, url, brokerType string) error {
	broker, err := r.FindUrl(owner, slug, url, brokerType)
	if err != nil {
		return err
	}

	return r.Delete(owner, slug, broker.Id)
}

// patch := bitbucket.GetPatch(repo, p.Id, u.BitbucketToken, u.BitbucketSecret)
func (r *BrokerResource) GetPatch(owner, slug string, id int) (string, error) {
	data := []byte{}
	// uri, err := url.Parse("https://api.bitbucket.org/1.0" + path)
	// https://bitbucket.org/!api/2.0/repositories/tdburke/test_mymysql/pullrequests/1/patch

	path := fmt.Sprintf("/repositories/tdburke/test_mymysql/pullrequests/1/patch")

	fmt.Println(path)

	if err := r.client.do("GET", path, nil, nil, &data); err != nil {
		fmt.Println("Get error:", err)
		return "", err
	}

	fmt.Println(data)

	if len(data) == 0 {
		return "", ErrNotFound
	}

	return "", nil
}

// -----------------------------------------------------------------------------
// Post Receive Hook Functions

// -----------------------------------------------------------------------------

var ErrInvalidPostReceiveHook = errors.New("Invalid Post Receive Hook")

type PullRequestHook struct {
	Id string `json:"id"`
}

type PostReceiveHook struct {
	Repo    *Repo     `json:"repository"`
	User    string    `json:"user"`
	Url     string    `json:"canon_url"`
	Commits []*Commit `json:"commits"`
}

type Commit struct {
	Message string `json:"message"`
	Author  string `json:"author"`
	Branch  string `json:"branch"`
	Hash    string `json:"raw_node"`
	Files []*File `json:"files"`
}

type File struct {
	Name string `json:"file"`
	Type string `json:"type"`
}

func ParseHook(raw []byte) (*PostReceiveHook, error) {
	hook := PostReceiveHook{}
	if err := json.Unmarshal(raw, &hook); err != nil {
		return nil, err
	}

	// it is possible the JSON was parsed, however,
	// was not from Bitbucket (maybe was from Google Code)
	// So we'll check to be sure certain key fields
	// were populated
	switch {
	case hook.Repo == nil:
		return nil, ErrInvalidPostReceiveHook
	case hook.Commits == nil:
		return nil, ErrInvalidPostReceiveHook
	case len(hook.User) == 0:
		return nil, ErrInvalidPostReceiveHook
	case len(hook.Commits) == 0:
		return nil, ErrInvalidPostReceiveHook
	}

	return &hook, nil
}

func ParsePullRequestHook(raw []byte) (*PullRequestHook, error) {
	data := make(map[string]map[string]interface{})
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}

	hook := PullRequestHook{}

	if p, ok := data["pullrequest_created"]; ok {
		fmt.Println(p["id"])
		hook.Id = fmt.Sprintf("%v", p["id"])
		if hook.Id == "" {
			return nil, errors.New("Could not parse bitbucket pullrequest_created message")
		}

		/*
			brokers := []*Broker{}
			path := fmt.Sprintf("/repositories/%s/%s/services/%v", owner, slug, id)

			if err := r.client.do("GET", path, nil, nil, &brokers); err != nil {
				return nil, err
			}

		*/

		return &hook, nil

	}

	//

	// How do we get the diff file?
	return nil, errors.New("Could not parse bitbucket pull request hook")
}

var ips = map[string]bool{
	"63.246.22.222": true,
}

// Check's to see if the Post-Receive Build Hook is coming
// from a valid sender (IP Address)
func IsValidSender(ip string) bool {
	return ips[ip]
}
