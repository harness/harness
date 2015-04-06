package dockerfile

import (
	"testing"
)

func TestWrite(t *testing.T) {

	var f = New("ubuntu")
	var got, want = f.String(), "FROM ubuntu\n"
	if got != want {
		t.Errorf("Exepected New() returned %s, got %s", want, got)
	}

	f = &Dockerfile{}
	f.WriteAdd("src", "target")
	got, want = f.String(), "ADD src target\n"
	if got != want {
		t.Errorf("Exepected WriteAdd returned %s, got %s", want, got)
	}

	f = &Dockerfile{}
	f.WriteFrom("ubuntu")
	got, want = f.String(), "FROM ubuntu\n"
	if got != want {
		t.Errorf("Exepected WriteFrom returned %s, got %s", want, got)
	}

	f = &Dockerfile{}
	f.WriteRun("whoami")
	got, want = f.String(), "RUN whoami\n"
	if got != want {
		t.Errorf("Exepected WriteRun returned %s, got %s", want, got)
	}

	f = &Dockerfile{}
	f.WriteUser("root")
	got, want = f.String(), "USER root\n"
	if got != want {
		t.Errorf("Exepected WriteUser returned %s, got %s", want, got)
	}

	f = &Dockerfile{}
	f.WriteEnv("FOO", "BAR")
	got, want = f.String(), "ENV FOO BAR\n"
	if got != want {
		t.Errorf("Exepected WriteEnv returned %s, got %s", want, got)
	}

	f = &Dockerfile{}
	f.WriteWorkdir("/home/ubuntu")
	got, want = f.String(), "WORKDIR /home/ubuntu\n"
	if got != want {
		t.Errorf("Exepected WriteWorkdir returned %s, got %s", want, got)
	}

	f = &Dockerfile{}
	f.WriteEntrypoint("/root")
	got, want = f.String(), "ENTRYPOINT /root\n"
	if got != want {
		t.Errorf("Exepected WriteEntrypoint returned %s, got %s", want, got)
	}
}
