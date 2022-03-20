package net

import (
	"fmt"
	"syscall"
	"strconv"

	"github.com/shirou/gopsutil/net"
	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/inputs"
	"github.com/influxdata/telegraf/plugins/inputs/system"
)

type NetPorts struct {
	ps system.PS
	Ports []uint32 `toml:"ports"`
}

func (ns *NetPorts) Description() string {
	return "Read TCP port metrics such as established, time wait and sockets counts."
}

var portSampleConfig = `
  ## gather the port connection status if you set ports parameter
  # ports = [22, 30022]
`

func (ns *NetPorts) SampleConfig() string {
	return portSampleConfig
}

func (ns *NetPorts) Gather(acc telegraf.Accumulator) error {
	netconns, err := ns.ps.NetConnections()
	if err != nil {
		return fmt.Errorf("error getting net connections info: %s", err)
	}

	// port gather
	for _, port := range ns.Ports {
		acc.AddError(ns.gatherPort(port, netconns, acc))
	}

	return nil
}

func (ns *NetPorts) gatherPort(port uint32, netconns []net.ConnectionStat, acc telegraf.Accumulator) error {
	counts := make(map[string]int)
	counts["UDP"] = 0

	// TODO: add family to tags or else
	for _, netcon := range netconns {
		if port == netcon.Laddr.Port {
			if netcon.Type == syscall.SOCK_DGRAM {
				counts["UDP"]++
				continue // UDP has no status
			}
			c, ok := counts[netcon.Status]
			if !ok {
				counts[netcon.Status] = 0
			}
			counts[netcon.Status] = c + 1
		}
	}

	tags := map[string]string{"port": strconv.Itoa(int(port))}
	fields := map[string]interface{}{
		"tcp_established": counts["ESTABLISHED"],
		"tcp_syn_sent":    counts["SYN_SENT"],
		"tcp_syn_recv":    counts["SYN_RECV"],
		"tcp_fin_wait1":   counts["FIN_WAIT1"],
		"tcp_fin_wait2":   counts["FIN_WAIT2"],
		"tcp_time_wait":   counts["TIME_WAIT"],
		"tcp_close":       counts["CLOSE"],
		"tcp_close_wait":  counts["CLOSE_WAIT"],
		"tcp_last_ack":    counts["LAST_ACK"],
		"tcp_listen":      counts["LISTEN"],
		"tcp_closing":     counts["CLOSING"],
		"tcp_none":        counts["NONE"],
		"udp_socket":      counts["UDP"],
	}
	acc.AddFields("netport", fields, tags)

	return nil
}

func init() {
	inputs.Add("netport", func() telegraf.Input {
		return &NetPorts{ps: system.NewSystemPS()}
	})
}
