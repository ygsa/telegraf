package ntpdate 

import (
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/internal"
	"github.com/influxdata/telegraf/plugins/inputs"
)

var (
	execCommand = exec.Command // execCommand is used to mock commands in tests.
)

type Ntpdate struct {
	Servers	  []string
	Samples   int8
	Timeout   int8
	path      string
}

func (*Ntpdate) Description() string {
	return "Get standard ntpdate metrics, requires ntpdate executable."
}

func (*Ntpdate) SampleConfig() string {
	return `
  ## An array of address to gather stats about. Specify an ip address or domain name.
  # servers = ["0.centos.pool.ntp.org"]

  ## Specify the number of samples to be acquired from each server as the integer 
  ## samples, with values from 1 to 8 inclusive, default is 2. 
  ## Equal to ntpdate '-p' option
  samples = 2

  ## Specify the maximum time waiting for a server response as the value timeout.
  ## default is 5 seconds
  timeout = 5
  `
}

func (c *Ntpdate) Init() error {
	var err error
	c.path, err = exec.LookPath("ntpdate")
	if err != nil {
		return errors.New("ntpdate not found: verify that ntpdate is installed and that ntpdate is in your PATH")
	}
	return nil
}

func (c *Ntpdate) Gather(acc telegraf.Accumulator) error {
	if len(c.Servers) == 0 {
		c.Servers = []string{"0.centos.pool.ntp.org"}
	}

	for _, serverAddress := range c.Servers {
		acc.AddError(c.gatherServer(serverAddress, acc))
	}
	return nil
}

func (c *Ntpdate) gatherServer(server string, acc telegraf.Accumulator) error {
	flags := []string{}

	if c.Samples == 0 {
		c.Samples = 2
	}
	flags = append(flags, fmt.Sprintf("-p %d", c.Samples))

	if c.Timeout == 0 {
		c.Timeout = 5
	}
	flags = append(flags, fmt.Sprintf("-t %d", c.Timeout))

	flags = append(flags, "-s")
	flags = append(flags, "-q")
	flags = append(flags, server)

	cmd := execCommand(c.path, flags...)
	out, err := internal.CombinedOutputTimeout(cmd, time.Second*(time.Duration(c.Timeout * 2)))
	if err != nil {
		return fmt.Errorf("failed to run command %s: %s - %s", strings.Join(cmd.Args, " "), err, string(out))
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		fields, tags, err := processNtpdatecOutput(line, server)
		if err != nil {
			return err
		}
		acc.AddFields("ntpdate", fields, tags)
	}
	return nil

}

/*
# one domain server can replay by multiple ntp server address
server 162.159.200.123, stratum 3, offset 0.003571, delay 0.12885
server 1.34.13.89, stratum 2, offset -0.000674, delay 0.03146
server 112.104.189.124, stratum 2, offset -0.000450, delay 0.03650
server 183.177.72.201, stratum 2, offset 0.000230, delay 0.03067
Error resolving 1.34.13.89ss: Name or service not known (-2)  # error server response
server 1.34.13.33, stratum 0, offset 0.000000, delay 0.00000  # cannot connect stratum 0
*/

func processNtpdatecOutput(line string, server string) (map[string]interface{}, map[string]string, error) {
	tags := map[string]string{}
	fields := map[string]interface{}{}
	tags["server"] = server

	stats := strings.Split(line, ",")
	if len(stats) < 4 {
		return nil, nil, fmt.Errorf("unexpected output from ntpdate, expected ',' in %s", line)
	}

	for _, item := range stats {
		var name string
		kv := strings.Split(strings.TrimSpace(item), " ")
		if strings.EqualFold(kv[0], "server") {
			name = "ntpserver"
			tags["ntpserver"] = kv[1]
		} else {
			name = strings.ToLower(kv[0])
			if strings.EqualFold(name, "stratum") {
				tags[name] = kv[1]
			}
			if (strings.EqualFold(name, "offset") || strings.EqualFold(name, "delay")) {
				fval, err := strconv.ParseFloat(kv[1], 64)
				if err != nil {
					fval = 0.000000
				}
				fields[name] = fval
			}
		}
	}

	return fields, tags, nil
}

func init() {
	inputs.Add("ntpdate", func() telegraf.Input {
		return &Ntpdate{}
	})
}
