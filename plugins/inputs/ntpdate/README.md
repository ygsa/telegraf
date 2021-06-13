# ntpdate Input Plugin

Get standard ntpdate metrics, requires ntpdate executable.

### Configuration:

```toml
[[inputs.ntpdate]]
  # An array of address to gather stats about. Specify an ip address or domain name.
  servers = ["0.centos.pool.ntp.org", "162.159.200.1"]

  # Specify the number of samples to be acquired from each server as the integer 
  # samples, with values from 1 to 8 inclusive, default is 2. 
  # Equal to ntpdate '-p' option
  samples = 2

  # Specify the maximum time waiting for a server response as the value timeout.
  timeout = 5
```

### Measurements & Fields:

- ntpdate
    - offset (float, seconds)
    - delay (float, seconds)

### Tags:

- All measurements have the following tags:
    - server
    - ntpserver
    - stratum

### Example Output:

```
$ telegraf --config telegraf.conf --input-filter ntpdate --test
> ntpdate,host=cz2,ntpserver=122.117.253.246,server=0.centos.pool.ntp.org,stratum=2 delay=0.03848,offset=0.001743 1623224584000000000
> ntpdate,host=cz2,ntpserver=60.248.114.17,server=0.centos.pool.ntp.org,stratum=2 delay=0.03387,offset=0.000042 1623224584000000000
> ntpdate,host=cz2,ntpserver=118.163.74.161,server=0.centos.pool.ntp.org,stratum=3 delay=0.08369,offset=0.001024 1623224584000000000
> ntpdate,host=cz2,ntpserver=103.159.118.4,server=0.centos.pool.ntp.org,stratum=2 delay=0.03545,offset=-0.000605 1623224584000000000
> ntpdate,host=cz2,ntpserver=162.159.200.1,server=162.159.200.1,stratum=3 delay=0.12311,offset=0.006341 1623224589000000000
```




