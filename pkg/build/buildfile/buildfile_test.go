package buildfile

import (
	"testing"
)

func TestWrite(t *testing.T) {

	var f = New()
	var got, want = f.String(), base
	if got != want {
		t.Errorf("Exepected New() returned %s, got %s", want, got)
	}

	f = &Buildfile{}
	f.WriteCmd("echo hi")
	got, want = f.String(), "echo '#DRONE:6563686f206869'\necho hi\n"
	if got != want {
		t.Errorf("Exepected WriteCmd returned %s, got %s", want, got)
	}

	f = &Buildfile{}
	f.WriteCmdSilent("echo hi")
	got, want = f.String(), "echo hi\n"
	if got != want {
		t.Errorf("Exepected WriteCmdSilent returned %s, got %s", want, got)
	}

	f = &Buildfile{}
	f.WriteComment("this is a comment")
	got, want = f.String(), "#this is a comment\n"
	if got != want {
		t.Errorf("Exepected WriteComment returned %s, got %s", want, got)
	}

	f = &Buildfile{}
	f.WriteEnv("FOO", "BAR")
	got, want = f.String(), "export FOO=BAR\n"
	if got != want {
		t.Errorf("Exepected WriteEnv returned %s, got %s", want, got)
	}

	f = &Buildfile{}
	f.WriteHost("127.0.0.1")
	got, want = f.String(), "[ -f /usr/bin/sudo ] || echo \"127.0.0.1\" | tee -a /etc/hosts\n[ -f /usr/bin/sudo ] && echo \"127.0.0.1\" | sudo tee -a /etc/hosts\n"
	if got != want {
		t.Errorf("Exepected WriteHost returned %s, got %s", want, got)
	}
}
