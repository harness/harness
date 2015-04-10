package buildfile

import (
	"testing"
)

func TestWrite(t *testing.T) {

	var f = New()
	var got, want = f.String(), base
	if got != want {
		t.Errorf("Expected New() returned %s, got %s", want, got)
	}

	f = &Buildfile{}
	f.WriteCmd("echo hi")
	got, want = f.String(), "echo '#DRONE:6563686f206869'\necho hi\n"
	if got != want {
		t.Errorf("Expected WriteCmd returned %s, got %s", want, got)
	}

	f = &Buildfile{}
	f.WriteCmdSilent("echo hi")
	got, want = f.String(), "echo hi\n"
	if got != want {
		t.Errorf("Expected WriteCmdSilent returned %s, got %s", want, got)
	}

	f = &Buildfile{}
	f.WriteComment("this is a comment")
	got, want = f.String(), "#this is a comment\n"
	if got != want {
		t.Errorf("Expected WriteComment returned %s, got %s", want, got)
	}

	f = &Buildfile{}
	f.WriteEnv("FOO", "BAR")
	got, want = f.String(), "export FOO=\"BAR\"\n"
	if got != want {
		t.Errorf("Expected WriteEnv returned %s, got %s", want, got)
	}

	f = &Buildfile{}
	f.WriteHost("127.0.0.1")
	got, want = f.String(), "[ -f /usr/bin/sudo ] || echo \"127.0.0.1\" | tee -a /etc/hosts\n[ -f /usr/bin/sudo ] && echo \"127.0.0.1\" | sudo tee -a /etc/hosts\n"
	if got != want {
		t.Errorf("Expected WriteHost returned %s, got %s", want, got)
	}

	f = &Buildfile{}
	f.WriteFile("$HOME/.ssh/id_rsa", []byte("ssh-rsa AAA..."), 600)
	got, want = f.String(), "echo 'ssh-rsa AAA...' | tee $HOME/.ssh/id_rsa > /dev/null\nchmod 600 $HOME/.ssh/id_rsa\n"
	if got != want {
		t.Errorf("Expected WriteFile returned \n%s, \ngot\n%s", want, got)
	}
}
