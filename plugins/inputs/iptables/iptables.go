// +build linux

package iptables

import (
	"errors"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"hash/crc32"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/inputs"
)

// Iptables is a telegraf plugin to gather packets and bytes throughput from Linux's iptables packet filter.
type Iptables struct {
	UseSudo bool
	UseLock bool
	UseRule bool
	Binary  string
	Table   string
	Chains  []string
	lister  chainLister
}

// Description returns a short description of the plugin.
func (ipt *Iptables) Description() string {
	return "Gather packets and bytes throughput from iptables"
}

// SampleConfig returns sample configuration options.
func (ipt *Iptables) SampleConfig() string {
	return `
  ## iptables require root access on most systems.
  ## Setting 'use_sudo' to true will make use of sudo to run iptables.
  ## Users must configure sudo to allow telegraf user to run iptables with no password.
  ## iptables can be restricted to list command "iptables -nvL" or "iptables -S".
  use_sudo = false
  ## Setting 'use_lock' to true runs iptables with the "-w" option.
  ## Adjust your sudo settings appropriately if using this option ("iptables -w 5 -nvl")
  use_lock = false
  ## Setting 'use_rule' to true to get ruleid result. default is true
  use_rule = true
  ## Define an alternate executable, such as "ip6tables". Default is "iptables".
  # binary = "ip6tables"
  ## defines the table to monitor:
  table = "filter"
  ## defines the chains to monitor.
  ## NOTE: iptables rules without a comment will not be monitored when use_rule is true.
  ## Read the plugin documentation for more information.
  chains = [ "INPUT" ]
`
}

// Gather gathers iptables packets and bytes throughput from the configured tables and chains.
func (ipt *Iptables) Gather(acc telegraf.Accumulator) error {
	if ipt.Table == "" || len(ipt.Chains) == 0 {
		return nil
	}
	// best effort : we continue through the chains even if an error is encountered,
	// but we keep track of the last error.
	for _, chain := range ipt.Chains {
		data, e := ipt.lister(ipt.Table, chain)
		if e != nil {
			acc.AddError(e)
			continue
		}
		e = ipt.parseAndGather(chain,data, acc)
		if e != nil {
			acc.AddError(e)
			continue
		}
	}
	return nil
}

func (ipt *Iptables) chainList(table, chain string) (string, error) {
	var binary string
	if ipt.Binary != "" {
		binary = ipt.Binary
	} else {
		binary = "iptables"
	}
	iptablePath, err := exec.LookPath(binary)
	if err != nil {
		return "", err
	}
	var args []string
	name := iptablePath
	if ipt.UseSudo {
		name = "sudo"
		args = append(args, iptablePath)
	}
	if ipt.UseLock {
		args = append(args, "-w", "5")
	}
	if ipt.UseRule {
		args = append(args, "-nvL", chain, "-t", table, "-x")
	} else {
		args = append(args, "-S", chain, "-t", table)
	}
	c := exec.Command(name, args...)
	out, err := c.Output()
	return string(out), err
}

const measurement = "iptables"

var errParse = errors.New("Cannot parse iptables list information")
var fieldsHeaderRe = regexp.MustCompile(`^\s*pkts\s+bytes\s+target`)
var valuesRe = regexp.MustCompile(`^\s*(\d+)\s+(\d+)\s+(\w+).*?/\*\s*(.+?)\s*\*/\s*`)
var inputStateRe = regexp.MustCompile(`^\-A\s+INPUT\s+\-m\s+state\s+`)
var inputDropRe = regexp.MustCompile(`^\-A\s+INPUT\s+.*\-j\s+DROP`)
var inputRejectRe = regexp.MustCompile(`^\-A\s+INPUT\s+.*\-j\s+REJECT`)
var forwardDropRe = regexp.MustCompile(`^\-A\s+FORWARD\s+.*\-j\s+DROP`)
var forwardRejectRe = regexp.MustCompile(`^\-A\s+FORWARD\s+.*\-j\s+REJECT`)

func (ipt *Iptables) parseAndGather(chain, data string, acc telegraf.Accumulator) error {
	lines := strings.Split(data, "\n")
	if ipt.UseRule {
		if len(lines) < 3 {
			return nil
		}
		if !fieldsHeaderRe.MatchString(lines[1]) {
			return errParse
		}
		for _, line := range lines[2:] {
			matches := valuesRe.FindStringSubmatch(line)
			if len(matches) != 5 {
				continue
			}

			pkts := matches[1]
			bytes := matches[2]
			target := matches[3]
			comment := matches[4]

			tags := map[string]string{"table": ipt.Table, "chain": chain, "target": target, "ruleid": comment}
			fields := make(map[string]interface{})

			var err error
			fields["pkts"], err = strconv.ParseUint(pkts, 10, 64)
			if err != nil {
				continue
			}
			fields["bytes"], err = strconv.ParseUint(bytes, 10, 64)
			if err != nil {
				continue
			}
			acc.AddFields(measurement, fields, tags)
		}
	} else {
		lenLines := len(lines) - 1 // ignore last empty line
		maxN := lenLines - 1
		tags := map[string]string{"table": ipt.Table, "chain": strings.ToLower(chain)}
		fields := map[string]interface{}{"total_rules": lenLines}

		// hash crc32 for data
		datasum := stringCrc(data)
		fields["checksum"] = datasum

		// only for iptables filter table
		if strings.Compare(ipt.Table, "filter") == 0 {
			if strings.Compare(chain, "INPUT") == 0 {
				fields["is_state"] = 0
				fields["is_drop"] = 0
				fields["is_reject"] = 0
				for _, line := range lines {
					if inputStateRe.MatchString(line) {
						fields["is_state"] = 1
						break
					}
				}
				// match the last line
				if inputDropRe.MatchString(lines[maxN]) {
					fields["is_drop"] = 1
				}
				if inputRejectRe.MatchString(lines[maxN]) {
					fields["is_reject"] = 1
				}

			}
			if strings.Compare(chain, "FORWARD") == 0 {
				fields["is_drop"] = 0
				fields["is_reject"] = 0

				// match the last line
				if forwardDropRe.MatchString(lines[maxN]) {
					fields["is_drop"] = 1
				}
				if forwardRejectRe.MatchString(lines[maxN]) {
					fields["is_reject"] = 1
				}
			}
			if strings.Compare(chain, "OUTPUT") == 0 {
				// ignore OUTPUT chain check
			}
		}
		// get result
		acc.AddFields(measurement, fields, tags)
	}
	return nil
}

func stringCrc(data string) uint32 {
	if len(data) == 0 {
		return 0
	}

	crc32q := crc32.MakeTable(0xD5828281)
	return crc32.Checksum([]byte(data), crc32q)
}

type chainLister func(table, chain string) (string, error)

func init() {
	inputs.Add("iptables", func() telegraf.Input {
		ipt := new(Iptables)
		ipt.UseRule = true
		ipt.lister = ipt.chainList
		return ipt
	})
}
