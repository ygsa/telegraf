package ntpdate

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/influxdata/telegraf/testutil"
)

func TestGather(t *testing.T) {
	c := Ntpdate{
		Servers: []string{"122.117.24.100"},
		Timeout: 5,
		path: "ntpdate",
	}
	// overwriting exec commands with mock commands
	execCommand = fakeExecCommand
	defer func() { execCommand = exec.Command }()
	var acc testutil.Accumulator

	err := c.Gather(&acc)
	if err != nil {
		t.Fatal(err)
	}

	tags := map[string]string{
		"server":    "122.117.24.100",
		"ntpserver": "122.117.24.100",
		"stratum":   "2",
	}
	fields := map[string]interface{}{
		"offset":  0.001173,
		"delay":   0.03293,
	}

	acc.AssertContainsTaggedFields(t, "ntpdate", fields, tags)

	err = c.Gather(&acc)
	if err != nil {
		t.Fatal(err)
	}
	acc.AssertContainsTaggedFields(t, "ntpdate", fields, tags)

}

// fackeExecCommand is a helper function that mock
// the exec.Command call (and call the test binary)
func fakeExecCommand(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

// TestHelperProcess isn't a real test. It's used to mock exec.Command
// For example, if you run:
// GO_WANT_HELPER_PROCESS=1 go test -test.run=TestHelperProcess -- ntpdate
// it returns below mockData.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	mockData := `server 122.117.24.100, stratum 2, offset 0.001173, delay 0.03293`

	args := os.Args

	// Previous arguments are tests stuff, that looks like :
	// /tmp/go-build970079519/…/_test/integration.test -test.run=TestHelperProcess --
	cmd, args := args[3], args[4:]

	if cmd == "ntpdate" {
		fmt.Fprint(os.Stdout, mockData)
	} else {
		fmt.Fprint(os.Stdout, "command not found")
		os.Exit(1)

	}
	os.Exit(0)
}
