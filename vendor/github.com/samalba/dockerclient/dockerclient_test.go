package dockerclient

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

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

func TestDockerClientInterface(t *testing.T) {
	iface := reflect.TypeOf((*Client)(nil)).Elem()
	test := testDockerClient(t)

	if !reflect.TypeOf(test).Implements(iface) {
		t.Fatalf("DockerClient does not implement the Client interface")
	}
}
