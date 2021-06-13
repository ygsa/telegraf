# Pika Input Plugin

### Configuration:

```toml
# Read pika's basic status information
[[inputs.pika]]
  ## specify servers via a url matching:
  ##  [protocol://][:password]@address[:port]
  ##  e.g.
  ##    tcp://localhost:9221
  ##    tcp://:password@192.168.99.100
  ##
  ## If no servers are specified, then localhost is used as the host.
  ## If no port is specified, 6379 is used
  servers = ["tcp://localhost:9221"]
  ## Optional. Specify redis commands to retrieve values
  # [[inputs.pika.commands]]
  # command = ["get", "sample-key"]
  # field = "sample-key-value"
  # type = "string"

  ## specify server password
  # password = "s#cr@t%"

  ## Optional TLS Config
  # tls_ca = "/etc/telegraf/ca.pem"
  # tls_cert = "/etc/telegraf/cert.pem"
  # tls_key = "/etc/telegraf/key.pem"
  ## Use TLS but skip chain & host verification
  # insecure_skip_verify = true
```

> note: pika and redis are compatible, read more from [pika](https://github.com/Qihoo360/pika).  

### Measurements & Fields:

The plugin gathers the results of the [INFO](https://redis.io/commands/info) redis command.
There are two separate measurements: _redis_ and _redis\_keyspace_, the latter is used for gathering database related statistics.

Additionally the plugin also calculates the hit/miss ratio (keyspace\_hitrate) and the elapsed time since the last rdb save (rdb\_last\_save\_time\_elapsed).

- pika

    **Server**
    - uptime(int, seconds)
    - redis_version(string)

    **Clients**
    - clients(int, number)

    **Memory**
    - used_memory(int, bytes)

    **Stats**
    - total_connections_received(int, number)
    - total_commands_processed(int, number)
    - instantaneous_ops_per_sec(int, number)

    **Replication**
    - connected_slaves(int, number)
    - master_link_status(string)

    **CPU**
    - used_cpu_sys(float, number)
    - used_cpu_user(float, number)
    - used_cpu_sys_children(float, number)
    - used_cpu_user_children(float, number)

- pika_keyspace
    - keys(int, number)

- pika_cmdstat
    Every Redis used command will have 3 new fields:
    - calls(int, number)

- pika_replication
  - tags:
    - replication_role
    - replica_ip
    - replica_port

### Tags:

- All measurements have the following tags:
    - port
    - server
    - replication_role

- The redis_keyspace measurement has an additional database tag:
    - database

- The redis_cmdstat measurement has an additional tag:
    - command

### Example Output:

Using this configuration:
```toml
[[inputs.pika]]
  ## specify servers via a url matching:
  ##  [protocol://][:password]@address[:port]
  ##  e.g.
  ##    tcp://localhost:6379
  ##    tcp://:password@192.168.99.100
  ##
  ## If no servers are specified, then localhost is used as the host.
  ## If no port is specified, 9221 is used
  servers = ["tcp://localhost:9221"]
```

When run with:
```
./telegraf --config telegraf.conf --input-filter pika --test
```

It produces:
```
> pika_cmdstat,command=info,host=cz2,port=9221,server=10.1.1.25 info=289i 1623227456000000000
> pika_cmdstat,command=slaveof,host=cz2,port=9221,server=10.1.1.25 slaveof=1i 1623227456000000000
> pika_cmdstat,command=auth,host=cz2,port=9221,server=10.1.1.25 auth=36i 1623227456000000000
> pika_keyspace,database=db0,host=cz2,key_type=strings,port=9221,replication_role=slave,server=10.1.1.25 expires=0i,invalid_keys=0i,key=0i 1623227456000000000
> pika_keyspace,database=db0,host=cz2,key_type=hashes,port=9221,replication_role=slave,server=10.1.1.25 expires=0i,invalid_keys=0i,key=0i 1623227456000000000
> pika_keyspace,database=db0,host=cz2,key_type=lists,port=9221,replication_role=slave,server=10.1.1.25 expires=0i,invalid_keys=0i,key=0i 1623227456000000000
> pika_keyspace,database=db0,host=cz2,key_type=zsets,port=9221,replication_role=slave,server=10.1.1.25 expires=0i,invalid_keys=0i,key=0i 1623227456000000000
> pika_keyspace,database=db0,host=cz2,key_type=sets,port=9221,replication_role=slave,server=10.1.1.25 expires=0i,invalid_keys=0i,key=0i 1623227456000000000
> pika_keyspace,database=db1,host=cz2,key_type=strings,port=9221,replication_role=slave,server=10.1.1.25 expires=0i,invalid_keys=0i,key=0i 1623227456000000000
> pika_keyspace,database=db1,host=cz2,key_type=hashes,port=9221,replication_role=slave,server=10.1.1.25 expires=0i,invalid_keys=0i,key=0i 1623227456000000000
> pika_keyspace,database=db1,host=cz2,key_type=lists,port=9221,replication_role=slave,server=10.1.1.25 expires=0i,invalid_keys=0i,key=0i 1623227456000000000
> pika_keyspace,database=db1,host=cz2,key_type=zsets,port=9221,replication_role=slave,server=10.1.1.25 expires=0i,invalid_keys=0i,key=0i 1623227456000000000
> pika_keyspace,database=db1,host=cz2,key_type=sets,port=9221,replication_role=slave,server=10.1.1.25 expires=0i,invalid_keys=0i,key=0i 1623227456000000000
> pika_keyspace,database=db2,host=cz2,key_type=strings,port=9221,replication_role=slave,server=10.1.1.25 expires=0i,invalid_keys=0i,key=0i 1623227456000000000
> pika_keyspace,database=db2,host=cz2,key_type=hashes,port=9221,replication_role=slave,server=10.1.1.25 expires=0i,invalid_keys=0i,key=0i 1623227456000000000
> pika_keyspace,database=db2,host=cz2,key_type=lists,port=9221,replication_role=slave,server=10.1.1.25 expires=0i,invalid_keys=0i,key=0i 1623227456000000000
> pika_keyspace,database=db2,host=cz2,key_type=zsets,port=9221,replication_role=slave,server=10.1.1.25 expires=0i,invalid_keys=0i,key=0i 1623227456000000000
> pika_keyspace,database=db2,host=cz2,key_type=sets,port=9221,replication_role=slave,server=10.1.1.25 expires=0i,invalid_keys=0i,key=0i 1623227456000000000
> pika_keyspace,database=db3,host=cz2,key_type=strings,port=9221,replication_role=slave,server=10.1.1.25 expires=0i,invalid_keys=0i,key=0i 1623227456000000000
> pika_keyspace,database=db3,host=cz2,key_type=hashes,port=9221,replication_role=slave,server=10.1.1.25 expires=0i,invalid_keys=0i,key=0i 1623227456000000000
> pika_keyspace,database=db3,host=cz2,key_type=lists,port=9221,replication_role=slave,server=10.1.1.25 expires=0i,invalid_keys=0i,key=0i 1623227456000000000
> pika_keyspace,database=db3,host=cz2,key_type=zsets,port=9221,replication_role=slave,server=10.1.1.25 expires=0i,invalid_keys=0i,key=0i 1623227456000000000
> pika_keyspace,database=db3,host=cz2,key_type=sets,port=9221,replication_role=slave,server=10.1.1.25 expires=0i,invalid_keys=0i,key=0i 1623227456000000000
> pika,host=cz2,port=9221,replication_role=slave,server=10.1.1.25 arch_bits=64i,clients=1i,compression="snappy",connected_slaves=1i,db_fatal=0i,db_memtable_usage=32000i,db_size=3182658i,db_tablereader_usage=0i,instantaneous_ops_per_sec=0i,is_bgsaving="No",is_compact="No",is_scaning_keyspace="No",log_size=297582i,pika_version="3.4.0",process_id=20982i,server_id=1i,sync_thread_num=6i,tcp_port=9221i,thread_num=4i,total_commands_processed=22i,total_connections_received=16i,uptime=17077i,used_cpu_sys=110.24,used_cpu_sys_children=0,used_cpu_user=66.8,used_cpu_user_children=0,used_memory=32000i 1623243793000000000
```
