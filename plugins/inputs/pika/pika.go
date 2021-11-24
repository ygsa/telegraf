package pika

import (
	"bufio"
	"fmt"
	"io"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/go-redis/redis"
	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/common/tls"
	"github.com/influxdata/telegraf/plugins/inputs"
)

type PikaCommand struct {
	Command []interface{}
	Field   string
	Type    string
}

type Pika struct {
	Commands []*PikaCommand
	Servers  []string
	Password string
	tls.ClientConfig

	Log telegraf.Logger

	clients     []Client
	initialized bool
}

type Client interface {
	Do(returnType string, args ...interface{}) (interface{}, error)
	Info() *redis.StringCmd
	Options() *redis.Options
	BaseTags() map[string]string
}

type PikaClient struct {
	client *redis.Client
	tags   map[string]string
}

// PikaFieldTypes defines the types expected for each of the fields redis reports on
type PikaFieldTypes struct {
	Clients                     int64   `json:"clients"`
	TcpPort                     int64   `json:tcp_port`
	ArchBits                    int64   `json:arch_bits`
	ProcessID                   int64   `json:process_id`
	ConnectedSlaves             int64   `json:"connected_slaves"`
	EvictedKeys                 int64   `json:"evicted_keys"`
	ExpireCycleCPUMilliseconds  int64   `json:"expire_cycle_cpu_milliseconds"`
	ExpiredKeys                 int64   `json:"expired_keys"`
	ExpiredStalePerc            float64 `json:"expired_stale_perc"`
	ExpiredTimeCapReachedCount  int64   `json:"expired_time_cap_reached_count"`
	InstantaneousOpsPerSec      int64   `json:"instantaneous_ops_per_sec"`
	MasterReplOffset            int64   `json:"master_repl_offset"`
	PikaVersion                 string  `json:"pika_version"`
	ThreadNum                   int64   `json:"thread_num"`
	SyncThreadNum               int64   `json:"sync_thread_num"`
	TotalCommandsProcessed      int64   `json:"total_commands_processed"`
	TotalConnectionsReceived    int64   `json:"total_connections_received"`
	Uptime                      int64   `json:"uptime"`
	UsedCPUSys                  float64 `json:"used_cpu_sys"`
	UsedCPUSysChildren          float64 `json:"used_cpu_sys_children"`
	UsedCPUUser                 float64 `json:"used_cpu_user"`
	UsedCPUUserChildren         float64 `json:"used_cpu_user_children"`
	UsedMemory                  int64   `json:"used_memory"`
	ServerId                    int64   `json:"server_id"`
	IsBgsaving                  string  `json:is_bgsaving`
	IsCompact                   string  `json:is_compact`
	IsScaningKeyspace           string  `json:is_scaning_keyspace`
	LogSize                     int64   `json:log_size`
	DBFatal                     int64   `json:db_fatal`
	DBMemtableUsage             int64   `json:db_memtable_usage`
	DBSize                      int64   `json:db_size`
	DBTablereaderUsage          int64   `json:db_tablereader_usage`
}

func (r *PikaClient) Do(returnType string, args ...interface{}) (interface{}, error) {
	rawVal := r.client.Do(args...)

	switch returnType {
	case "integer":
		return rawVal.Int64()
	case "string":
		return rawVal.String()
	case "float":
		return rawVal.Float64()
	default:
		return rawVal.String()
	}
}

func (r *PikaClient) Info() *redis.StringCmd {
	return r.client.Info("ALL")
}

func (r *PikaClient) Options() *redis.Options {
	return r.client.Options()
}

func (r *PikaClient) BaseTags() map[string]string {
	tags := make(map[string]string)
	for k, v := range r.tags {
		tags[k] = v
	}
	return tags
}

var replicationSlaveMetricPrefix = regexp.MustCompile(`^slave\d+`)

var sampleConfig = `
  ## specify servers via a url matching:
  ##  [protocol://][:password]@address[:port]
  ##  e.g.
  ##    tcp://localhost:9221
  ##    tcp://:password@192.168.99.100
  ##
  ## If no servers are specified, then localhost is used as the host.
  ## If no port is specified, 9221 is used
  servers = ["tcp://localhost:9221"]

  ## Optional. Specify pika commands to retrieve values
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
`

func (r *Pika) SampleConfig() string {
	return sampleConfig
}

func (r *Pika) Description() string {
	return "Read metrics from one or many pika servers"
}

var Tracking = map[string]string{
	"uptime_in_seconds": "uptime",
	"connected_clients": "clients",
	"role":              "replication_role",
}

func (r *Pika) init() error {
	if r.initialized {
		return nil
	}

	if len(r.Servers) == 0 {
		r.Servers = []string{"tcp://localhost:9221"}
	}

	r.clients = make([]Client, len(r.Servers))

	for i, serv := range r.Servers {
		if !strings.HasPrefix(serv, "tcp://") {
			r.Log.Warn("Server URL found without scheme; please update your configuration file")
			serv = "tcp://" + serv
		}

		u, err := url.Parse(serv)
		if err != nil {
			return fmt.Errorf("unable to parse to address %q: %s", serv, err.Error())
		}

		password := ""
		if u.User != nil {
			pw, ok := u.User.Password()
			if ok {
				password = pw
			}
		}
		if len(r.Password) > 0 {
			password = r.Password
		}

		var address string
		address = u.Host

		tlsConfig, err := r.ClientConfig.TLSConfig()
		if err != nil {
			return err
		}

		client := redis.NewClient(
			&redis.Options{
				Addr:      address,
				Password:  password,
				Network:   u.Scheme,
				PoolSize:  1,
				TLSConfig: tlsConfig,
			},
		)

		tags := map[string]string{}
		tags["server"] = u.Hostname()
		tags["port"] = u.Port()

		r.clients[i] = &PikaClient{
			client: client,
			tags:   tags,
		}
	}

	r.initialized = true
	return nil
}

// Reads stats from all configured servers accumulates stats.
// Returns one of the errors encountered while gather stats (if any).
func (r *Pika) Gather(acc telegraf.Accumulator) error {
	if !r.initialized {
		err := r.init()
		if err != nil {
			return err
		}
	}

	var wg sync.WaitGroup

	for _, client := range r.clients {
		wg.Add(1)
		go func(client Client) {
			defer wg.Done()
			acc.AddError(r.gatherServer(client, acc))
			acc.AddError(r.gatherCommandValues(client, acc))
		}(client)
	}

	wg.Wait()
	return nil
}

func (r *Pika) gatherCommandValues(client Client, acc telegraf.Accumulator) error {
	fields := make(map[string]interface{})
	for _, command := range r.Commands {
		val, err := client.Do(command.Type, command.Command...)
		if err != nil {
			return err
		}

		fields[command.Field] = val
	}

	acc.AddFields("pika_commands", fields, client.BaseTags())

	return nil
}

func (r *Pika) gatherServer(client Client, acc telegraf.Accumulator) error {
	info, err := client.Info().Result()
	if err != nil {
		return fmt.Errorf("redis(%v) - %s", client.Options().Addr, err)
	}

	rdr := strings.NewReader(info)
	return gatherInfoOutput(rdr, acc, client.BaseTags())
}

// gatherInfoOutput gathers
func gatherInfoOutput(
	rdr io.Reader,
	acc telegraf.Accumulator,
	tags map[string]string,
) error {
	var section string

	scanner := bufio.NewScanner(rdr)
	fields := make(map[string]interface{})
	for scanner.Scan() {
		line := scanner.Text()

		if len(line) == 0 {
			continue
		}

		if strings.HasPrefix(line, "# Time:") {
			continue
		}

		if strings.HasPrefix(line, "# Duration:") {
			continue
		}

		if line[0] == '#' {
			if len(line) > 2 {
				section = line[2:]

				kt := getSubstr(`Replication(?P<name>\(\w+\))`, line)
				var suff string = kt["name"]
				if len(suff) > 0 {
					section = strings.TrimRight(section, suff)
				}
			}
			continue
		}

		re := regexp.MustCompile(`^db\d+\s`)
		var parts []string
		if !strings.EqualFold(section, "Replication") && re.MatchString(line) {
			parts = strings.SplitN(line, " ", 2)
		} else {
			parts = strings.SplitN(line, ":", 2)
		}

		if len(parts) < 2 {
			continue
		}
		name := parts[0]

		if section == "Server" {
			if (name != "uptime_in_seconds" && name != "pika_version" &&
				name != "arch_bits" && name != "tcp_port"  &&
				name != "server_id" && name != "process_id"  &&
				name != "thread_num" && name != "sync_thread_num") {
				continue
			}
		}

		if section == "Data" && name == "db_fatal_msg" {
			continue
		}

		if section == "Stats" && strings.HasPrefix(name, "compact_") {
			continue
		}

		if strings.HasSuffix(name, "_human") {
			continue
		}

		metric, ok := Tracking[name]
		if !ok {
			if section == "Keyspace" {
				kline := strings.TrimSpace(parts[1])
				gatherKeyspaceLine(name, kline, acc, tags)
				continue
			}
			if section == "Command_Exec_Count" {
				kline := strings.TrimSpace(parts[1])
				gatherCommandstateLine(strings.ToLower(name), kline, acc, tags)
				continue
			}
			//if section == "Replication" && replicationSlaveMetricPrefix.MatchString(name) {
			if section == "Replication" { 
				if replicationSlaveMetricPrefix.MatchString(name) {
					kline := strings.TrimSpace(parts[1])
					gatherReplicationLine(name, kline, acc, tags)
					continue
				}
			}

			metric = name
		}

		val := strings.TrimSpace(parts[1])

		// Some percentage values have a "%" suffix that we need to get rid of before int/float conversion
		val = strings.TrimSuffix(val, "%")

		if strings.EqualFold(val, "No") {
			val = string("0")
		}
		if strings.EqualFold(val, "Yes") {
			val = string("1")
		}

		// Try parsing as int
		if ival, err := strconv.ParseInt(val, 10, 64); err == nil {
			fields[metric] = ival
			continue
		}

		// Try parsing as a float
		if fval, err := strconv.ParseFloat(val, 64); err == nil {
			fields[metric] = fval
			continue
		}

		// Treat it as a string

		if name == "role" {
			tags["replication_role"] = val
			continue
		}

		fields[metric] = val
	}
	o := PikaFieldTypes{}

	setStructFieldsFromObject(fields, &o)
	setExistingFieldsFromStruct(fields, &o)


	acc.AddFields("pika", fields, tags)
	return nil
}

// parse line and return the group values
func getSubstr(regEx, line string) (subsMap map[string]string) {
	var compRegEx = regexp.MustCompile(regEx)
	match := compRegEx.FindStringSubmatch(line)

	subsMap = make(map[string]string)
	for i, name := range compRegEx.SubexpNames() {
		if i > 0 && i <= len(match) {
			subsMap[name] = match[i]
		}
	}
return subsMap
}

// Parse the special Keyspace line at end of redis stats
// This is a special line that looks something like:
//     db0 Strings_keys=400000080, expires=0, invalid_keys=0
//     db0 Hashes_keys=0, expires=0, invalid_keys=0
//     db0 Lists_keys=0, expires=0, invalid_keys=0
//     db0 Zsets_keys=0, expires=0, invalid_keys=0
//     db0 Sets_keys=0, expires=0, invalid_keys=0
// And there is one for each db on the redis instance
func gatherKeyspaceLine(
	name string,
	line string,
	acc telegraf.Accumulator,
	globalTags map[string]string,
) {
	if strings.Contains(line, "keys=") {
		fields := make(map[string]interface{})
		tags := make(map[string]string)
		for k, v := range globalTags {
			tags[k] = v
		}
		tags["database"] = name
		dbparts := strings.Split(line, ", ")

		// get key type 
		kp := getSubstr(`(?P<kname>\w+)_keys`, line)
		var kprefix string = kp["kname"]
		if len(kprefix) > 0 {
			tags["key_type"] = strings.ToLower(kprefix)
		}

		for _, dbp := range dbparts {
			kv := strings.Split(dbp, "=")
			ival, err := strconv.ParseInt(kv[1], 10, 64)
			if err == nil {
				var key string
				if strings.HasPrefix(kv[0], kprefix) {
					key = strings.Trim(kv[0], kprefix + "_")
				} else {
					key = kv[0]
				}

				fields[strings.ToLower(key)] = ival
			}
		}
		acc.AddFields("pika_keyspace", fields, tags)
	}
}

// Parse the special cmdstat lines.
// Example:
// --
// INFO:31
// SLAVEOF:1
// AUTH:78
// Tag: cmdstat=flush; 1
func gatherCommandstateLine(
	name string,
	line string,
	acc telegraf.Accumulator,
	globalTags map[string]string,
) {
	fields := make(map[string]interface{})
	tags := make(map[string]string)
	for k, v := range globalTags {
		tags[k] = v
	}
	tags["command"] = name
	ival, err := strconv.ParseInt(line, 10, 64)
	if err == nil {
		fields[name] = ival
	}
	acc.AddFields("pika_cmdstat", fields, tags)
}

// Parse the special Replication line
// Example:
//     slave0:ip=pika.slave.net,port=9221,conn_fd=120,lag=(db0:0)(db1:0)(db2:0)(db3:0)
// This line will only be visible when a node has a replica attached.
func gatherReplicationLine(
	name string,
	line string,
	acc telegraf.Accumulator,
	globalTags map[string]string,
) {
	fields := make(map[string]interface{})
	tags := make(map[string]string)
	for k, v := range globalTags {
		tags[k] = v
	}

	tags["pika_role"] = "slave"

	parts := strings.Split(line, ",")
	for _, part := range parts {
		kv := strings.Split(part, "=")
		if len(kv) != 2 {
			continue
		}

		switch kv[0] {
		case "ip":
			tags["pika_ip"] = kv[1]
		case "port":
			tags["pika_port"] = kv[1]
		case "lag":
			kv[1] = strings.ReplaceAll(kv[1], ")(", "|")
			mt := getSubstr(`\((?P<match>.+)\)`, kv[1])
			if len(mt["match"]) > 0 {
				nkvs := strings.Split(mt["match"], "|")
				for _, nkv := range nkvs {
					nl := strings.Split(nkv, ":")
					nval, err := strconv.ParseInt(nl[1], 10, 64)
					if err == nil {
						fields[nl[0]] = nval
					}
				}
			}
		default:
			ival, err := strconv.ParseInt(kv[1], 10, 64)
			if err == nil {
				fields[kv[0]] = ival
			}
		}
	}

	acc.AddFields("pika_replication", fields, tags)
}

func init() {
	inputs.Add("pika", func() telegraf.Input {
		return &Pika{}
	})
}

func setExistingFieldsFromStruct(fields map[string]interface{}, o *PikaFieldTypes) {
	val := reflect.ValueOf(o).Elem()
	typ := val.Type()

	for key := range fields {
		if _, exists := fields[key]; exists {
			for i := 0; i < typ.NumField(); i++ {
				f := typ.Field(i)
				jsonFieldName := f.Tag.Get("json")
				if jsonFieldName == key {
					fields[key] = val.Field(i).Interface()
					break
				}
			}
		}
	}
}

func setStructFieldsFromObject(fields map[string]interface{}, o *PikaFieldTypes) {
	val := reflect.ValueOf(o).Elem()
	typ := val.Type()

	for key, value := range fields {
		if _, exists := fields[key]; exists {
			for i := 0; i < typ.NumField(); i++ {
				f := typ.Field(i)
				jsonFieldName := f.Tag.Get("json")
				if jsonFieldName == key {
					structFieldValue := val.Field(i)
					structFieldValue.Set(coerceType(value, structFieldValue.Type()))
					break
				}
			}
		}
	}
}

func coerceType(value interface{}, typ reflect.Type) reflect.Value {
	switch sourceType := value.(type) {
	case bool:
		switch typ.Kind() {
		case reflect.String:
			if sourceType {
				value = "true"
			} else {
				value = "false"
			}
		case reflect.Int64:
			if sourceType {
				value = int64(1)
			} else {
				value = int64(0)
			}
		case reflect.Float64:
			if sourceType {
				value = float64(1)
			} else {
				value = float64(0)
			}
		default:
			panic(fmt.Sprintf("unhandled destination type %s", typ.Kind().String()))
		}
	case int, int8, int16, int32, int64:
		switch typ.Kind() {
		case reflect.String:
			value = fmt.Sprintf("%d", value)
		case reflect.Int64:
			// types match
		case reflect.Float64:
			value = float64(reflect.ValueOf(sourceType).Int())
		default:
			panic(fmt.Sprintf("unhandled destination type %s", typ.Kind().String()))
		}
	case uint, uint8, uint16, uint32, uint64:
		switch typ.Kind() {
		case reflect.String:
			value = fmt.Sprintf("%d", value)
		case reflect.Int64:
			// types match
		case reflect.Float64:
			value = float64(reflect.ValueOf(sourceType).Uint())
		default:
			panic(fmt.Sprintf("unhandled destination type %s", typ.Kind().String()))
		}
	case float32, float64:
		switch typ.Kind() {
		case reflect.String:
			value = fmt.Sprintf("%f", value)
		case reflect.Int64:
			value = int64(reflect.ValueOf(sourceType).Float())
		case reflect.Float64:
			// types match
		default:
			panic(fmt.Sprintf("unhandled destination type %s", typ.Kind().String()))
		}
	case string:
		switch typ.Kind() {
		case reflect.String:
			// types match
		case reflect.Int64:
			value, _ = strconv.ParseInt(value.(string), 10, 64)
		case reflect.Float64:
			value, _ = strconv.ParseFloat(value.(string), 64)
		default:
			panic(fmt.Sprintf("unhandled destination type %s", typ.Kind().String()))
		}
	default:
		panic(fmt.Sprintf("unhandled source type %T", sourceType))
	}
	return reflect.ValueOf(value)
}
