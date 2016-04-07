package dockerclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/pkg/stdcopy"
)

func assertEqual(t *testing.T, a interface{}, b interface{}, message string) {
	if a == b {
		return
	}
	if len(message) == 0 {
		message = fmt.Sprintf("%v != %v", a, b)
	}
	t.Fatal(message)
}

func testDockerClient(t *testing.T) *DockerClient {
	client, err := NewDockerClient(testHTTPServer.URL, nil)
	if err != nil {
		t.Fatal("Cannot init the docker client")
	}
	return client
}

func ExampleDockerClient_AttachContainer() {
	docker, err := NewDockerClient("unix:///var/run/docker.sock", nil)
	if err != nil {
		panic(err)
	}
	cID, err := docker.CreateContainer(&ContainerConfig{
		Cmd:   []string{"echo", "hi"},
		Image: "busybox",
	}, "", nil)
	if err != nil {
		panic(err)
	}
	done := make(chan struct{})
	if body, err := docker.AttachContainer(cID, &AttachOptions{
		Stream: true,
		Stdout: true,
	}); err != nil {
		panic(err)
	} else {
		go func() {
			defer body.Close()
			if _, err := stdcopy.StdCopy(os.Stdout, os.Stderr, body); err != nil {
				panic(err)
			}
			close(done)
		}()
	}

	if err := docker.StartContainer(cID, nil); err != nil {
		panic(err)
	}
	<-done
}

func TestInfo(t *testing.T) {
	client := testDockerClient(t)
	info, err := client.Info()
	if err != nil {
		t.Fatal("Cannot get server info")
	}
	assertEqual(t, info.Images, int64(1), "")
	assertEqual(t, info.Containers, int64(2), "")
}

func TestKillContainer(t *testing.T) {
	client := testDockerClient(t)
	if err := client.KillContainer("23132acf2ac", "5"); err != nil {
		t.Fatal("cannot kill container: %s", err)
	}
}

func TestWait(t *testing.T) {
	client := testDockerClient(t)

	// This provokes an error on the server.
	select {
	case wr := <-client.Wait("1234"):
		assertEqual(t, wr.ExitCode, int(-1), "")
	case <-time.After(2 * time.Second):
		t.Fatal("Timed out!")
	}

	// Valid case.
	select {
	case wr := <-client.Wait("valid-id"):
		assertEqual(t, wr.ExitCode, int(0), "")
	case <-time.After(2 * time.Second):
		t.Fatal("Timed out!")
	}
}

func TestPullImage(t *testing.T) {
	client := testDockerClient(t)
	err := client.PullImage("busybox", nil)
	if err != nil {
		t.Fatal("unable to pull busybox")
	}

	err = client.PullImage("haproxy", nil)
	if err != nil {
		t.Fatal("unable to pull haproxy")
	}

	err = client.PullImage("wrongimg", nil)
	if err == nil {
		t.Fatal("should return error when it fails to pull wrongimg")
	}
}

func TestListContainers(t *testing.T) {
	client := testDockerClient(t)
	containers, err := client.ListContainers(true, false, "")
	if err != nil {
		t.Fatal("cannot get containers: %s", err)
	}
	assertEqual(t, len(containers), 1, "")
	cnt := containers[0]
	assertEqual(t, cnt.SizeRw, int64(0), "")
}

func TestContainerChanges(t *testing.T) {
	client := testDockerClient(t)
	changes, err := client.ContainerChanges("foobar")
	if err != nil {
		t.Fatal("cannot get container changes: %s", err)
	}
	assertEqual(t, len(changes), 3, "unexpected number of changes")
	c := changes[0]
	assertEqual(t, c.Path, "/dev", "unexpected")
	assertEqual(t, c.Kind, 0, "unexpected")
}

func TestListContainersWithSize(t *testing.T) {
	client := testDockerClient(t)
	containers, err := client.ListContainers(true, true, "")
	if err != nil {
		t.Fatal("cannot get containers: %s", err)
	}
	assertEqual(t, len(containers), 1, "")
	cnt := containers[0]
	assertEqual(t, cnt.SizeRw, int64(123), "")
}

func TestListContainersWithFilters(t *testing.T) {
	client := testDockerClient(t)
	containers, err := client.ListContainers(true, true, "{'id':['332375cfbc23edb921a21026314c3497674ba8bdcb2c85e0e65ebf2017f688ce']}")
	if err != nil {
		t.Fatal("cannot get containers: %s", err)
	}
	assertEqual(t, len(containers), 1, "")

	containers, err = client.ListContainers(true, true, "{'id':['332375cfbc23edb921a21026314c3497674ba8bdcb2c85e0e65ebf2017f688cf']}")
	if err != nil {
		t.Fatal("cannot get containers: %s", err)
	}
	assertEqual(t, len(containers), 0, "")
}

func TestContainerLogs(t *testing.T) {
	client := testDockerClient(t)
	containerId := "foobar"
	logOptions := &LogOptions{
		Follow:     true,
		Stdout:     true,
		Stderr:     true,
		Timestamps: true,
		Tail:       10,
	}
	logsReader, err := client.ContainerLogs(containerId, logOptions)
	if err != nil {
		t.Fatal("cannot read logs from server")
	}

	stdoutBuffer := new(bytes.Buffer)
	stderrBuffer := new(bytes.Buffer)
	if _, err = stdcopy.StdCopy(stdoutBuffer, stderrBuffer, logsReader); err != nil {
		t.Fatal("cannot read logs from logs reader")
	}
	stdoutLogs := strings.TrimSpace(stdoutBuffer.String())
	stderrLogs := strings.TrimSpace(stderrBuffer.String())
	stdoutLogLines := strings.Split(stdoutLogs, "\n")
	stderrLogLines := strings.Split(stderrLogs, "\n")
	if len(stdoutLogLines) != 5 {
		t.Fatalf("wrong number of stdout logs: len=%d", len(stdoutLogLines))
	}
	if len(stderrLogLines) != 5 {
		t.Fatalf("wrong number of stderr logs: len=%d", len(stdoutLogLines))
	}
	for i, line := range stdoutLogLines {
		expectedSuffix := fmt.Sprintf("Z line %d", 41+2*i)
		if !strings.HasSuffix(line, expectedSuffix) {
			t.Fatalf("expected stdout log line \"%s\" to end with \"%s\"", line, expectedSuffix)
		}
	}
	for i, line := range stderrLogLines {
		expectedSuffix := fmt.Sprintf("Z line %d", 40+2*i)
		if !strings.HasSuffix(line, expectedSuffix) {
			t.Fatalf("expected stderr log line \"%s\" to end with \"%s\"", line, expectedSuffix)
		}
	}
}

func TestContainerStats(t *testing.T) {
	client := testDockerClient(t)
	var expectedContainerStats Stats
	if err := json.Unmarshal([]byte(statsResp), &expectedContainerStats); err != nil {
		t.Fatalf("cannot parse expected resp: %s", err.Error())
	}
	containerIds := []string{"foobar", "foo"}
	expectedResults := [][]StatsOrError{
		{{Stats: expectedContainerStats}, {Error: fmt.Errorf("invalid character 'i' looking for beginning of value")}},
		{{Stats: expectedContainerStats}, {Stats: expectedContainerStats}},
	}

	for i := range containerIds {
		t.Logf("on outer iter %d\n", i)
		stopChan := make(chan struct{})
		statsOrErrorChan, err := client.ContainerStats(containerIds[i], stopChan)
		if err != nil {
			t.Fatalf("cannot get stats from server: %s", err.Error())
		}

		for j, expectedResult := range expectedResults[i] {
			t.Logf("on iter %d\n", j)
			containerStatsOrError := <-statsOrErrorChan
			if containerStatsOrError.Error != nil {
				if expectedResult.Error == nil {
					t.Fatalf("index %d, got unexpected error %v", j, containerStatsOrError.Error)
				} else if containerStatsOrError.Error.Error() == expectedResult.Error.Error() {
					// continue so that we don't try to
					// compare error values directly
					continue
				} else {
					t.Fatalf("index %d, expected error %q but got %q", j, expectedResult.Error, containerStatsOrError.Error)
				}
			}
			if !reflect.DeepEqual(containerStatsOrError, expectedResult) {
				t.Fatalf("index %d, got:\n%#v\nexpected:\n%#v", j, containerStatsOrError, expectedResult)
			}
			t.Logf("done with iter %d\n", j)
		}
		close(stopChan)
		t.Logf("done with outer iter %d\n", i)
	}
}

func TestMonitorEvents(t *testing.T) {
	client := testDockerClient(t)
	decoder := json.NewDecoder(bytes.NewBufferString(eventsResp))
	var expectedEvents []Event
	for {
		var event Event
		if err := decoder.Decode(&event); err != nil {
			if err == io.EOF {
				break
			} else {
				t.Fatalf("cannot parse expected resp: %s", err.Error())
			}
		} else {
			expectedEvents = append(expectedEvents, event)
		}
	}

	// test passing stop chan
	stopChan := make(chan struct{})
	eventInfoChan, err := client.MonitorEvents(nil, stopChan)
	if err != nil {
		t.Fatalf("cannot get events from server: %s", err.Error())
	}

	eventInfo := <-eventInfoChan
	if eventInfo.Error != nil || eventInfo.Event != expectedEvents[0] {
		t.Fatalf("got:\n%#v\nexpected:\n%#v", eventInfo, expectedEvents[0])
	}
	close(stopChan)
	for i := 0; i < 3; i++ {
		_, ok := <-eventInfoChan
		if i == 2 && ok {
			t.Fatalf("read more than 2 events successfully after closing stopChan")
		}
	}

	// test when you don't pass stop chan
	eventInfoChan, err = client.MonitorEvents(nil, nil)
	if err != nil {
		t.Fatalf("cannot get events from server: %s", err.Error())
	}

	for i, expectedEvent := range expectedEvents {
		t.Logf("on iter %d\n", i)
		eventInfo := <-eventInfoChan
		if eventInfo.Error != nil || eventInfo.Event != expectedEvent {
			t.Fatalf("index %d, got:\n%#v\nexpected:\n%#v", i, eventInfo, expectedEvent)
		}
		t.Logf("done with iter %d\n", i)
	}
}

func TestDockerClientInterface(t *testing.T) {
	iface := reflect.TypeOf((*Client)(nil)).Elem()
	test := testDockerClient(t)

	if !reflect.TypeOf(test).Implements(iface) {
		t.Fatalf("DockerClient does not implement the Client interface")
	}
}
