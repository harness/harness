package dockerclient

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

var _ Client = (*DockerClient)(nil)

const (
	// APIVersion is currently hardcoded to v1.15
	// TODO: bump the API version or allow users to choose which API version to
	// use the client with. The current value does not make sense for many
	// methods, such as ContainerStats, StartMonitorStats, and StopAllMonitorStats
	// (v1.17) and
	// ListVolumes, {Remove,Create}Volume, ListNetworks,
	// {Inspect,Create,Connect,Disconnect,Remove}Network (v1.21)
	APIVersion = "v1.15"
)

var (
	ErrImageNotFound     = errors.New("Image not found")
	ErrNotFound          = errors.New("Not found")
	ErrConnectionRefused = errors.New("Cannot connect to the docker engine endpoint")

	defaultTimeout = 30 * time.Second
)

type DockerClient struct {
	URL           *url.URL
	HTTPClient    *http.Client
	TLSConfig     *tls.Config
	monitorStats  int32
	eventStopChan chan (struct{})
}

type Error struct {
	StatusCode int
	Status     string
	msg        string
}

func (e Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Status, e.msg)
}

func NewDockerClient(daemonUrl string, tlsConfig *tls.Config) (*DockerClient, error) {
	return NewDockerClientTimeout(daemonUrl, tlsConfig, time.Duration(defaultTimeout), nil)
}

func NewDockerClientTimeout(daemonUrl string, tlsConfig *tls.Config, timeout time.Duration, setUserTimeout tcpFunc) (*DockerClient, error) {
	u, err := url.Parse(daemonUrl)
	if err != nil {
		return nil, err
	}
	if u.Scheme == "" || u.Scheme == "tcp" {
		if tlsConfig == nil {
			u.Scheme = "http"
		} else {
			u.Scheme = "https"
		}
	}
	httpClient := newHTTPClient(u, tlsConfig, timeout, setUserTimeout)
	return &DockerClient{u, httpClient, tlsConfig, 0, nil}, nil
}

func (client *DockerClient) doRequest(method string, path string, body []byte, headers map[string]string) ([]byte, error) {
	b := bytes.NewBuffer(body)

	reader, err := client.doStreamRequest(method, path, b, headers)
	if err != nil {
		return nil, err
	}

	defer reader.Close()
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (client *DockerClient) doStreamRequest(method string, path string, in io.Reader, headers map[string]string) (io.ReadCloser, error) {
	if (method == "POST" || method == "PUT") && in == nil {
		in = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, client.URL.String()+path, in)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	if headers != nil {
		for header, value := range headers {
			req.Header.Add(header, value)
		}
	}
	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		if !strings.Contains(err.Error(), "connection refused") && client.TLSConfig == nil {
			return nil, fmt.Errorf("%v. Are you trying to connect to a TLS-enabled daemon without TLS?", err)
		}
		if strings.Contains(err.Error(), "connection refused") {
			return nil, ErrConnectionRefused
		}
		return nil, err
	}
	if resp.StatusCode == 404 {
		defer resp.Body.Close()
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, ErrNotFound
		}
		if len(data) > 0 {
			// check if is image not found error
			if strings.Index(string(data), "No such image") != -1 {
				return nil, ErrImageNotFound
			}
			return nil, errors.New(string(data))
		}
		return nil, ErrNotFound
	}
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return nil, Error{StatusCode: resp.StatusCode, Status: resp.Status, msg: string(data)}
	}

	return resp.Body, nil
}

func (client *DockerClient) Info() (*Info, error) {
	uri := fmt.Sprintf("/%s/info", APIVersion)
	data, err := client.doRequest("GET", uri, nil, nil)
	if err != nil {
		return nil, err
	}
	ret := &Info{}
	err = json.Unmarshal(data, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (client *DockerClient) ListContainers(all bool, size bool, filters string) ([]Container, error) {
	argAll := 0
	if all == true {
		argAll = 1
	}
	showSize := 0
	if size == true {
		showSize = 1
	}
	uri := fmt.Sprintf("/%s/containers/json?all=%d&size=%d", APIVersion, argAll, showSize)

	if filters != "" {
		uri += "&filters=" + filters
	}

	data, err := client.doRequest("GET", uri, nil, nil)
	if err != nil {
		return nil, err
	}
	ret := []Container{}
	err = json.Unmarshal(data, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (client *DockerClient) InspectContainer(id string) (*ContainerInfo, error) {
	uri := fmt.Sprintf("/%s/containers/%s/json", APIVersion, id)
	data, err := client.doRequest("GET", uri, nil, nil)
	if err != nil {
		return nil, err
	}
	info := &ContainerInfo{}
	err = json.Unmarshal(data, info)
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (client *DockerClient) CreateContainer(config *ContainerConfig, name string, auth *AuthConfig) (string, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return "", err
	}
	uri := fmt.Sprintf("/%s/containers/create", APIVersion)
	if name != "" {
		v := url.Values{}
		v.Set("name", name)
		uri = fmt.Sprintf("%s?%s", uri, v.Encode())
	}
	headers := map[string]string{}
	if auth != nil {
		encoded_auth, err := auth.encode()
		if err != nil {
			return "", err
		}
		headers["X-Registry-Auth"] = encoded_auth
	}
	data, err = client.doRequest("POST", uri, data, headers)
	if err != nil {
		return "", err
	}
	result := &RespContainersCreate{}
	err = json.Unmarshal(data, result)
	if err != nil {
		return "", fmt.Errorf(string(data))
	}
	return result.Id, nil
}

func (client *DockerClient) ContainerLogs(id string, options *LogOptions) (io.ReadCloser, error) {
	v := url.Values{}
	v.Add("follow", strconv.FormatBool(options.Follow))
	v.Add("stdout", strconv.FormatBool(options.Stdout))
	v.Add("stderr", strconv.FormatBool(options.Stderr))
	v.Add("timestamps", strconv.FormatBool(options.Timestamps))
	if options.Tail > 0 {
		v.Add("tail", strconv.FormatInt(options.Tail, 10))
	}

	uri := fmt.Sprintf("/%s/containers/%s/logs?%s", APIVersion, id, v.Encode())
	req, err := http.NewRequest("GET", client.URL.String()+uri, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

func (client *DockerClient) ContainerChanges(id string) ([]*ContainerChanges, error) {
	uri := fmt.Sprintf("/%s/containers/%s/changes", APIVersion, id)
	data, err := client.doRequest("GET", uri, nil, nil)
	if err != nil {
		return nil, err
	}
	changes := []*ContainerChanges{}
	err = json.Unmarshal(data, &changes)
	if err != nil {
		return nil, err
	}
	return changes, nil
}

func (client *DockerClient) ContainerStats(id string, stopChan <-chan struct{}) (<-chan StatsOrError, error) {
	uri := fmt.Sprintf("/%s/containers/%s/stats", APIVersion, id)
	resp, err := client.HTTPClient.Get(client.URL.String() + uri)
	if err != nil {
		return nil, err
	}

	decode := func(decoder *json.Decoder) decodingResult {
		var containerStats Stats
		if err := decoder.Decode(&containerStats); err != nil {
			return decodingResult{err: err}
		} else {
			return decodingResult{result: containerStats}
		}
	}
	decodingResultChan := client.readJSONStream(resp.Body, decode, stopChan)
	statsOrErrorChan := make(chan StatsOrError)
	go func() {
		for decodingResult := range decodingResultChan {
			stats, _ := decodingResult.result.(Stats)
			statsOrErrorChan <- StatsOrError{
				Stats: stats,
				Error: decodingResult.err,
			}
		}
		close(statsOrErrorChan)
	}()
	return statsOrErrorChan, nil
}

func (client *DockerClient) readJSONStream(stream io.ReadCloser, decode func(*json.Decoder) decodingResult, stopChan <-chan struct{}) <-chan decodingResult {
	resultChan := make(chan decodingResult)

	go func() {
		decodeChan := make(chan decodingResult)

		go func() {
			decoder := json.NewDecoder(stream)
			for {
				decodeResult := decode(decoder)
				decodeChan <- decodeResult
				if decodeResult.err != nil {
					close(decodeChan)
					return
				}
			}
		}()

		defer close(resultChan)

		for {
			select {
			case <-stopChan:
				stream.Close()
				for range decodeChan {
				}
				return
			case decodeResult := <-decodeChan:
				resultChan <- decodeResult
				if decodeResult.err != nil {
					stream.Close()
					return
				}
			}
		}

	}()

	return resultChan
}

func (client *DockerClient) ExecCreate(config *ExecConfig) (string, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return "", err
	}
	uri := fmt.Sprintf("/%s/containers/%s/exec", APIVersion, config.Container)
	resp, err := client.doRequest("POST", uri, data, nil)
	if err != nil {
		return "", err
	}
	var createExecResp struct {
		Id string
	}
	if err = json.Unmarshal(resp, &createExecResp); err != nil {
		return "", err
	}
	return createExecResp.Id, nil
}

func (client *DockerClient) ExecStart(id string, config *ExecConfig) error {
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}

	uri := fmt.Sprintf("/%s/exec/%s/start", APIVersion, id)
	if _, err := client.doRequest("POST", uri, data, nil); err != nil {
		return err
	}

	return nil
}

func (client *DockerClient) ExecResize(id string, width, height int) error {
	v := url.Values{}

	w := strconv.Itoa(width)
	h := strconv.Itoa(height)

	v.Set("w", w)
	v.Set("h", h)

	uri := fmt.Sprintf("/%s/exec/%s/resize?%s", APIVersion, id, v.Encode())
	if _, err := client.doRequest("POST", client.URL.String()+uri, nil, nil); err != nil {
		return err
	}

	return nil
}

func (client *DockerClient) AttachContainer(id string, options *AttachOptions) (io.ReadCloser, error) {
	v := url.Values{}
	if options != nil {
		if options.Logs {
			v.Set("logs", "1")
		}
		if options.Stream {
			v.Set("stream", "1")
		}
		if options.Stdin {
			v.Set("stdin", "1")
		}
		if options.Stdout {
			v.Set("stdout", "1")
		}
		if options.Stderr {
			v.Set("stderr", "1")
		}
	}
	uri := fmt.Sprintf("/%s/containers/%s/attach?%s", APIVersion, id, v.Encode())
	return client.doStreamRequest("POST", uri, nil, nil)
}

func (client *DockerClient) StartContainer(id string, config *HostConfig) error {
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	uri := fmt.Sprintf("/%s/containers/%s/start", APIVersion, id)
	_, err = client.doRequest("POST", uri, data, nil)
	if err != nil {
		return err
	}
	return nil
}

func (client *DockerClient) StopContainer(id string, timeout int) error {
	uri := fmt.Sprintf("/%s/containers/%s/stop?t=%d", APIVersion, id, timeout)
	_, err := client.doRequest("POST", uri, nil, nil)
	if err != nil {
		return err
	}
	return nil
}

func (client *DockerClient) RestartContainer(id string, timeout int) error {
	uri := fmt.Sprintf("/%s/containers/%s/restart?t=%d", APIVersion, id, timeout)
	_, err := client.doRequest("POST", uri, nil, nil)
	if err != nil {
		return err
	}
	return nil
}

func (client *DockerClient) KillContainer(id, signal string) error {
	uri := fmt.Sprintf("/%s/containers/%s/kill?signal=%s", APIVersion, id, signal)
	_, err := client.doRequest("POST", uri, nil, nil)
	if err != nil {
		return err
	}
	return nil
}

func (client *DockerClient) Wait(id string) <-chan WaitResult {
	ch := make(chan WaitResult)
	uri := fmt.Sprintf("/%s/containers/%s/wait", APIVersion, id)

	go func() {
		data, err := client.doRequest("POST", uri, nil, nil)
		if err != nil {
			ch <- WaitResult{ExitCode: -1, Error: err}
			return
		}

		var result struct {
			StatusCode int `json:"StatusCode"`
		}
		err = json.Unmarshal(data, &result)
		ch <- WaitResult{ExitCode: result.StatusCode, Error: err}
	}()
	return ch
}

func (client *DockerClient) MonitorEvents(options *MonitorEventsOptions, stopChan <-chan struct{}) (<-chan EventOrError, error) {
	v := url.Values{}
	if options != nil {
		if options.Since != 0 {
			v.Add("since", strconv.Itoa(options.Since))
		}
		if options.Until != 0 {
			v.Add("until", strconv.Itoa(options.Until))
		}
		if options.Filters != nil {
			filterMap := make(map[string][]string)
			if len(options.Filters.Event) > 0 {
				filterMap["event"] = []string{options.Filters.Event}
			}
			if len(options.Filters.Image) > 0 {
				filterMap["image"] = []string{options.Filters.Image}
			}
			if len(options.Filters.Container) > 0 {
				filterMap["container"] = []string{options.Filters.Container}
			}
			if len(filterMap) > 0 {
				filterJSONBytes, err := json.Marshal(filterMap)
				if err != nil {
					return nil, err
				}
				v.Add("filters", string(filterJSONBytes))
			}
		}
	}
	uri := fmt.Sprintf("%s/%s/events?%s", client.URL.String(), APIVersion, v.Encode())
	resp, err := client.HTTPClient.Get(uri)
	if err != nil {
		return nil, err
	}

	decode := func(decoder *json.Decoder) decodingResult {
		var event Event
		if err := decoder.Decode(&event); err != nil {
			return decodingResult{err: err}
		} else {
			return decodingResult{result: event}
		}
	}
	decodingResultChan := client.readJSONStream(resp.Body, decode, stopChan)
	eventOrErrorChan := make(chan EventOrError)
	go func() {
		for decodingResult := range decodingResultChan {
			event, _ := decodingResult.result.(Event)
			eventOrErrorChan <- EventOrError{
				Event: event,
				Error: decodingResult.err,
			}
		}
		close(eventOrErrorChan)
	}()
	return eventOrErrorChan, nil
}

func (client *DockerClient) StartMonitorEvents(cb Callback, ec chan error, args ...interface{}) {
	client.eventStopChan = make(chan struct{})

	go func() {
		eventErrChan, err := client.MonitorEvents(nil, client.eventStopChan)
		if err != nil {
			if ec != nil {
				ec <- err
			}
			return
		}

		for e := range eventErrChan {
			if e.Error != nil {
				if ec != nil {
					ec <- e.Error
				}
				return
			}
			cb(&e.Event, ec, args...)
		}
	}()
}

func (client *DockerClient) StopAllMonitorEvents() {
	if client.eventStopChan == nil {
		return
	}
	close(client.eventStopChan)
}

func (client *DockerClient) StartMonitorStats(id string, cb StatCallback, ec chan error, args ...interface{}) {
	atomic.StoreInt32(&client.monitorStats, 1)
	go client.getStats(id, cb, ec, args...)
}

func (client *DockerClient) getStats(id string, cb StatCallback, ec chan error, args ...interface{}) {
	uri := fmt.Sprintf("%s/%s/containers/%s/stats", client.URL.String(), APIVersion, id)
	resp, err := client.HTTPClient.Get(uri)
	if err != nil {
		ec <- err
		return
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	for atomic.LoadInt32(&client.monitorStats) > 0 {
		var stats *Stats
		if err := dec.Decode(&stats); err != nil {
			ec <- err
			return
		}
		cb(id, stats, ec, args...)
	}
}

func (client *DockerClient) StopAllMonitorStats() {
	atomic.StoreInt32(&client.monitorStats, 0)
}

func (client *DockerClient) TagImage(nameOrID string, repo string, tag string, force bool) error {
	v := url.Values{}
	v.Set("repo", repo)
	v.Set("tag", tag)
	if force {
		v.Set("force", "1")
	}
	uri := fmt.Sprintf("/%s/images/%s/tag?%s", APIVersion, nameOrID, v.Encode())
	if _, err := client.doRequest("POST", uri, nil, nil); err != nil {
		return err
	}
	return nil
}

func (client *DockerClient) Version() (*Version, error) {
	uri := fmt.Sprintf("/%s/version", APIVersion)
	data, err := client.doRequest("GET", uri, nil, nil)
	if err != nil {
		return nil, err
	}
	version := &Version{}
	err = json.Unmarshal(data, version)
	if err != nil {
		return nil, err
	}
	return version, nil
}

func (client *DockerClient) PushImage(name string, tag string, auth *AuthConfig) error {
	v := url.Values{}
	if tag != "" {
		v.Set("tag", tag)
	}
	uri := fmt.Sprintf("/%s/images/%s/push?%s", APIVersion, url.QueryEscape(name), v.Encode())
	req, err := http.NewRequest("POST", client.URL.String()+uri, nil)
	if auth != nil {
		if encodedAuth, err := auth.encode(); err != nil {
			return err
		} else {
			req.Header.Add("X-Registry-Auth", encodedAuth)
		}
	}
	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var finalObj map[string]interface{}
	for decoder := json.NewDecoder(resp.Body); err == nil; err = decoder.Decode(&finalObj) {
	}
	if err != io.EOF {
		return err
	}
	if err, ok := finalObj["error"]; ok {
		return fmt.Errorf("%v", err)
	}
	return nil
}

func (client *DockerClient) PullImage(name string, auth *AuthConfig) error {
	v := url.Values{}
	v.Set("fromImage", name)
	uri := fmt.Sprintf("/%s/images/create?%s", APIVersion, v.Encode())
	req, err := http.NewRequest("POST", client.URL.String()+uri, nil)
	if auth != nil {
		encoded_auth, err := auth.encode()
		if err != nil {
			return err
		}
		req.Header.Add("X-Registry-Auth", encoded_auth)
	}
	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return ErrNotFound
	}
	if resp.StatusCode >= 400 {
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("%s", string(data))
	}

	var finalObj map[string]interface{}
	for decoder := json.NewDecoder(resp.Body); err == nil; err = decoder.Decode(&finalObj) {
	}
	if err != io.EOF {
		return err
	}
	if err, ok := finalObj["error"]; ok {
		return fmt.Errorf("%v", err)
	}
	return nil
}

func (client *DockerClient) InspectImage(id string) (*ImageInfo, error) {
	uri := fmt.Sprintf("/%s/images/%s/json", APIVersion, id)
	data, err := client.doRequest("GET", uri, nil, nil)
	if err != nil {
		return nil, err
	}
	info := &ImageInfo{}
	err = json.Unmarshal(data, info)
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (client *DockerClient) LoadImage(reader io.Reader) error {
	uri := fmt.Sprintf("/%s/images/load", APIVersion)
	_, err := client.doStreamRequest("POST", uri, reader, nil)
	return err
}

func (client *DockerClient) RemoveContainer(id string, force, volumes bool) error {
	argForce := 0
	argVolumes := 0
	if force == true {
		argForce = 1
	}
	if volumes == true {
		argVolumes = 1
	}
	args := fmt.Sprintf("force=%d&v=%d", argForce, argVolumes)
	uri := fmt.Sprintf("/%s/containers/%s?%s", APIVersion, id, args)
	_, err := client.doRequest("DELETE", uri, nil, nil)
	return err
}

func (client *DockerClient) ListImages(all bool) ([]*Image, error) {
	argAll := 0
	if all {
		argAll = 1
	}
	uri := fmt.Sprintf("/%s/images/json?all=%d", APIVersion, argAll)
	data, err := client.doRequest("GET", uri, nil, nil)
	if err != nil {
		return nil, err
	}
	var images []*Image
	if err := json.Unmarshal(data, &images); err != nil {
		return nil, err
	}
	return images, nil
}

func (client *DockerClient) RemoveImage(name string, force bool) ([]*ImageDelete, error) {
	argForce := 0
	if force {
		argForce = 1
	}

	args := fmt.Sprintf("force=%d", argForce)
	uri := fmt.Sprintf("/%s/images/%s?%s", APIVersion, name, args)
	data, err := client.doRequest("DELETE", uri, nil, nil)
	if err != nil {
		return nil, err
	}
	var imageDelete []*ImageDelete
	if err := json.Unmarshal(data, &imageDelete); err != nil {
		return nil, err
	}
	return imageDelete, nil
}

func (client *DockerClient) SearchImages(query, registry string, auth *AuthConfig) ([]ImageSearch, error) {
	term := query
	if registry != "" {
		term = registry + "/" + term
	}
	uri := fmt.Sprintf("/%s/images/search?term=%s", APIVersion, term)
	headers := map[string]string{}
	if auth != nil {
		if encodedAuth, err := auth.encode(); err != nil {
			return nil, err
		} else {
			headers["X-Registry-Auth"] = encodedAuth
		}
	}
	data, err := client.doRequest("GET", uri, nil, headers)
	if err != nil {
		return nil, err
	}
	var imageSearches []ImageSearch
	if err := json.Unmarshal(data, &imageSearches); err != nil {
		return nil, err
	}
	return imageSearches, nil
}

func (client *DockerClient) PauseContainer(id string) error {
	uri := fmt.Sprintf("/%s/containers/%s/pause", APIVersion, id)
	_, err := client.doRequest("POST", uri, nil, nil)
	if err != nil {
		return err
	}
	return nil
}
func (client *DockerClient) UnpauseContainer(id string) error {
	uri := fmt.Sprintf("/%s/containers/%s/unpause", APIVersion, id)
	_, err := client.doRequest("POST", uri, nil, nil)
	if err != nil {
		return err
	}
	return nil
}

func (client *DockerClient) RenameContainer(oldName string, newName string) error {
	uri := fmt.Sprintf("/containers/%s/rename?name=%s", oldName, newName)
	_, err := client.doRequest("POST", uri, nil, nil)
	return err
}

func (client *DockerClient) ImportImage(source string, repository string, tag string, tar io.Reader) (io.ReadCloser, error) {
	var fromSrc string
	v := &url.Values{}
	if source == "" {
		fromSrc = "-"
	} else {
		fromSrc = source
	}

	v.Set("fromSrc", fromSrc)
	v.Set("repo", repository)
	if tag != "" {
		v.Set("tag", tag)
	}

	var in io.Reader
	if fromSrc == "-" {
		in = tar
	}
	return client.doStreamRequest("POST", "/images/create?"+v.Encode(), in, nil)
}

func (client *DockerClient) BuildImage(image *BuildImage) (io.ReadCloser, error) {
	v := url.Values{}

	if image.DockerfileName != "" {
		v.Set("dockerfile", image.DockerfileName)
	}
	if image.RepoName != "" {
		v.Set("t", image.RepoName)
	}
	if image.RemoteURL != "" {
		v.Set("remote", image.RemoteURL)
	}
	if image.NoCache {
		v.Set("nocache", "1")
	}
	if image.Pull {
		v.Set("pull", "1")
	}
	if image.Remove {
		v.Set("rm", "1")
	} else {
		v.Set("rm", "0")
	}
	if image.ForceRemove {
		v.Set("forcerm", "1")
	}
	if image.SuppressOutput {
		v.Set("q", "1")
	}

	v.Set("memory", strconv.FormatInt(image.Memory, 10))
	v.Set("memswap", strconv.FormatInt(image.MemorySwap, 10))
	v.Set("cpushares", strconv.FormatInt(image.CpuShares, 10))
	v.Set("cpuperiod", strconv.FormatInt(image.CpuPeriod, 10))
	v.Set("cpuquota", strconv.FormatInt(image.CpuQuota, 10))
	v.Set("cpusetcpus", image.CpuSetCpus)
	v.Set("cpusetmems", image.CpuSetMems)
	v.Set("cgroupparent", image.CgroupParent)
	if image.BuildArgs != nil {
		buildArgsJSON, err := json.Marshal(image.BuildArgs)
		if err != nil {
			return nil, err
		}
		v.Set("buildargs", string(buildArgsJSON))
	}

	headers := make(map[string]string)
	if image.Config != nil {
		encoded_config, err := image.Config.encode()
		if err != nil {
			return nil, err
		}
		headers["X-Registry-Config"] = encoded_config
	}
	if image.Context != nil {
		headers["Content-Type"] = "application/tar"
	}

	uri := fmt.Sprintf("/%s/build?%s", APIVersion, v.Encode())
	return client.doStreamRequest("POST", uri, image.Context, headers)
}

func (client *DockerClient) ListVolumes() ([]*Volume, error) {
	uri := fmt.Sprintf("/%s/volumes", APIVersion)
	data, err := client.doRequest("GET", uri, nil, nil)
	if err != nil {
		return nil, err
	}
	var volumesList VolumesListResponse
	if err := json.Unmarshal(data, &volumesList); err != nil {
		return nil, err
	}
	return volumesList.Volumes, nil
}

func (client *DockerClient) RemoveVolume(name string) error {
	uri := fmt.Sprintf("/%s/volumes/%s", APIVersion, name)
	_, err := client.doRequest("DELETE", uri, nil, nil)
	return err
}

func (client *DockerClient) CreateVolume(request *VolumeCreateRequest) (*Volume, error) {
	data, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	uri := fmt.Sprintf("/%s/volumes/create", APIVersion)
	data, err = client.doRequest("POST", uri, data, nil)
	if err != nil {
		return nil, err
	}
	volume := &Volume{}
	err = json.Unmarshal(data, volume)
	return volume, err
}

func (client *DockerClient) ListNetworks(filters string) ([]*NetworkResource, error) {
	uri := fmt.Sprintf("/%s/networks", APIVersion)

	if filters != "" {
		uri += "&filters=" + filters
	}

	data, err := client.doRequest("GET", uri, nil, nil)
	if err != nil {
		return nil, err
	}
	ret := []*NetworkResource{}
	err = json.Unmarshal(data, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (client *DockerClient) InspectNetwork(id string) (*NetworkResource, error) {
	uri := fmt.Sprintf("/%s/networks/%s", APIVersion, id)

	data, err := client.doRequest("GET", uri, nil, nil)
	if err != nil {
		return nil, err
	}
	ret := &NetworkResource{}
	err = json.Unmarshal(data, ret)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (client *DockerClient) CreateNetwork(config *NetworkCreate) (*NetworkCreateResponse, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	uri := fmt.Sprintf("/%s/networks/create", APIVersion)
	data, err = client.doRequest("POST", uri, data, nil)
	if err != nil {
		return nil, err
	}
	ret := &NetworkCreateResponse{}
	err = json.Unmarshal(data, ret)
	return ret, nil
}

func (client *DockerClient) ConnectNetwork(id, container string) error {
	data, err := json.Marshal(NetworkConnect{Container: container})
	if err != nil {
		return err
	}
	uri := fmt.Sprintf("/%s/networks/%s/connect", APIVersion, id)
	_, err = client.doRequest("POST", uri, data, nil)
	return err
}

func (client *DockerClient) DisconnectNetwork(id, container string, force bool) error {
	data, err := json.Marshal(NetworkDisconnect{Container: container, Force: force})
	if err != nil {
		return err
	}
	uri := fmt.Sprintf("/%s/networks/%s/disconnect", APIVersion, id)
	_, err = client.doRequest("POST", uri, data, nil)
	return err
}

func (client *DockerClient) RemoveNetwork(id string) error {
	uri := fmt.Sprintf("/%s/networks/%s", APIVersion, id)
	_, err := client.doRequest("DELETE", uri, nil, nil)
	return err
}
