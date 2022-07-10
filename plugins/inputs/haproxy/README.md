# HAProxy Input Plugin

The [HAProxy](http://www.haproxy.org/) input plugin gathers
[statistics](https://cbonte.github.io/haproxy-dconv/1.9/intro.html#3.3.16)
using the [stats socket](https://cbonte.github.io/haproxy-dconv/1.9/management.html#9.3)
or [HTTP statistics page](https://cbonte.github.io/haproxy-dconv/1.9/management.html#9) of a HAProxy server.

### Configuration:

```toml
# Read metrics of HAProxy, via socket or HTTP stats page
[[inputs.haproxy]]
  ## An array of address to gather stats about. Specify an ip on hostname
  ## with optional port. ie localhost, 10.10.3.33:1936, etc.
  ## Make sure you specify the complete path to the stats endpoint
  ## including the protocol, ie http://10.10.3.33:1936/haproxy?stats

  ## Credentials for basic HTTP authentication
  # username = "admin"
  # password = "admin"

  ## If no servers are specified, then default to 127.0.0.1:1936/haproxy?stats
  servers = ["http://myhaproxy.com:1936/haproxy?stats"]

  ## You can also use local socket with standard wildcard globbing.
  ## Server address not starting with 'http' will be treated as a possible
  ## socket, so both examples below are valid.
  # servers = ["socket:/run/haproxy/admin.sock", "/run/haproxy/*.sock"]

  ## By default, some of the fields are renamed from what haproxy calls them.
  ## Setting this option to true results in the plugin keeping the original
  ## field names.
  # keep_field_names = false

  ## Optional TLS Config
  # tls_ca = "/etc/telegraf/ca.pem"
  # tls_cert = "/etc/telegraf/cert.pem"
  # tls_key = "/etc/telegraf/key.pem"
  ## Use TLS but skip chain & host verification
  # insecure_skip_verify = false
```

#### HAProxy Configuration

The following information may be useful when getting started, but please
consult the HAProxy documentation for complete and up to date instructions.

The [`stats enable`](https://cbonte.github.io/haproxy-dconv/1.8/configuration.html#4-stats%20enable)
option can be used to add unauthenticated access over HTTP using the default
settings.  To enable the unix socket begin by reading about the
[`stats socket`](https://cbonte.github.io/haproxy-dconv/1.8/configuration.html#3.1-stats%20socket)
option.


#### servers

Server addresses must explicitly start with 'http' if you wish to use HAProxy
status page.  Otherwise, addresses will be assumed to be an UNIX socket and
any protocol (if present) will be discarded.

When using socket names, wildcard expansion is supported so plugin can gather
stats from multiple sockets at once.

To use HTTP Basic Auth add the username and password in the userinfo section
of the URL: `http://user:password@1.2.3.4/haproxy?stats`.  The credentials are
sent via the `Authorization` header and not using the request URL.


#### keep_field_names

By default, some of the fields are renamed from what haproxy calls them.
Setting the `keep_field_names` parameter to `true` will result in the plugin
keeping the original field names.

The following renames are made:
- `pxname` -> `proxy`
- `svname` -> `sv`
- `act` -> `active_servers`
- `bck` -> `backup_servers`
- `cli_abrt` -> `cli_abort`
- `srv_abrt` -> `srv_abort`
- `hrsp_1xx` -> `http_response.1xx`
- `hrsp_2xx` -> `http_response.2xx`
- `hrsp_3xx` -> `http_response.3xx`
- `hrsp_4xx` -> `http_response.4xx`
- `hrsp_5xx` -> `http_response.5xx`
- `hrsp_other` -> `http_response.other`

### Metrics:

For more details about collected metrics reference the [HAProxy CSV format
documentation](https://cbonte.github.io/haproxy-dconv/1.8/management.html#9.1).

- haproxy
  - tags:
    - `server` - address of the server data was gathered from
    - `proxy` - proxy name
    - `sv` - service name
    - `type` - proxy session type
    - `mode` (string)
    - `addr`(string) # if type is server
  - fields:
    - `status` (int)
    - `check_status` (int)
    - `last_chk` (string)
    - `tracked` (string)
    - `agent_status` (int)
    - `last_agt` (string)
    - `cookie` (string)
    - `lastsess` (int)
    - **all other stats** (int)

> status, check_status and agent_status use the following value:
```
status:
        "DOWN":     0,
        "OPEN":     1,
        "UP":       2,
        "NOLB":     3,
        "MAINT":    4,
        "no check": 5,

check_status, agent_status:
        "UNK":        0,
        "INI":        1,
        "CHECKED":    2,
        "HANA":       3,
        "SOCKERR":    4,
        "L4OK":       5,
        "L4TOUT":     6,
        "L4CON":      7,
        "L6OK":       8,
        "L6TOUT":     9,
        "L6RSP":      10,
        "L7TOUT":     11,
        "L7RSP":      12,
        "L7OK":       13,
        "L7OKC":      14,
        "L7STS":      15,
        "PROCERR":    16,
        "PROCTOUT":   17,
        "PROCOK":     18,
```

> read more from [haproxy-configuration](https://www.haproxy.org/download/1.8/doc/configuration.txt), `section 9.1`.

### Example Output:
```
haproxy,mode="http",server=/run/haproxy/admin.sock,proxy=public,sv=FRONTEND,type=frontend http_response.other=0i,req_rate_max=1i,comp_byp=0i,status=1,rate_lim=0i,dses=0i,req_rate=0i,comp_rsp=0i,bout=9287i,comp_in=0i,smax=1i,slim=2000i,http_response.1xx=0i,conn_rate=0i,dreq=0i,ereq=0i,iid=2i,rate_max=1i,http_response.2xx=1i,comp_out=0i,intercepted=1i,stot=2i,pid=1i,http_response.5xx=1i,http_response.3xx=0i,http_response.4xx=0i,conn_rate_max=1i,conn_tot=2i,dcon=0i,bin=294i,rate=0i,sid=0i,req_tot=2i,scur=0i,dresp=0i 1513293519000000000
```
