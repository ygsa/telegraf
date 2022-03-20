# NetPort Input Plugin

This plugin collects TCP port connections state and UDP socket counts by using `lsof`.

### Configuration:

``` toml
# Collect TCP port connections state and UDP socket counts
[[inputs.netport]]
  ports = [22, 2003]
```

# Measurements:

Supported TCP port Connection states are follows.

- established
- syn_sent
- syn_recv
- fin_wait1
- fin_wait2
- time_wait
- close
- close_wait
- last_ack
- listen
- closing
- none

### TCP port Connection State measurements:

Meta:
- units: counts

Measurement names:
- tcp_established
- tcp_syn_sent
- tcp_syn_recv
- tcp_fin_wait1
- tcp_fin_wait2
- tcp_time_wait
- tcp_close
- tcp_close_wait
- tcp_last_ack
- tcp_listen
- tcp_closing
- tcp_none

If there are no connection on the state, the metric is not counted.

### UDP socket counts measurements:

Meta:
- units: counts

Measurement names:
- udp_socket

### output sample

```
> netport,host=cz,port=22 tcp_close=0i,tcp_close_wait=0i,tcp_closing=0i,tcp_established=1i,tcp_fin_wait1=0i,tcp_fin_wait2=0i,tcp_last_ack=0i,tcp_listen=2i,tcp_none=0i,tcp_syn_recv=0i,tcp_syn_sent=0i,tcp_time_wait=0i,udp_socket=0i 1647741023000000000
> netport,host=cz2,port=2003 tcp_close=0i,tcp_close_wait=0i,tcp_closing=0i,tcp_established=0i,tcp_fin_wait1=0i,tcp_fin_wait2=0i,tcp_last_ack=0i,tcp_listen=1i,tcp_none=0i,tcp_syn_recv=0i,tcp_syn_sent=0i,tcp_time_wait=0i,udp_socket=1i 1647741023000000000
```
