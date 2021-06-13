package pika

import (
	"bufio"
	"fmt"
	"strings"
	"testing"
	_ "time"

	"github.com/go-redis/redis"
	"github.com/influxdata/telegraf/testutil"
	_ "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testClient struct {
}

func (t *testClient) BaseTags() map[string]string {
	return map[string]string{"host": "pika.net"}
}

func (t *testClient) Info() *redis.StringCmd {
	return nil
}

func (t *testClient) Do(returnType string, args ...interface{}) (interface{}, error) {
	return 2, nil
}

func TestRedisConnectIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	addr := fmt.Sprintf("tcp://" + testutil.GetLocalHost() + ":9221")

	r := &Pika{
		Log:      testutil.Logger{},
		Servers:  []string{addr},
	}

	var acc testutil.Accumulator

	err := acc.GatherError(r.Gather)
	require.NoError(t, err)
}

func TestRedis_Commands(t *testing.T) {
	const redisListKey = "test-list-length"
	var acc testutil.Accumulator

	tc := &testClient{}

	rc := &PikaCommand{
		Command: []interface{}{"llen", "test-list"},
		Field:   redisListKey,
		Type:    "integer",
	}

	r := &Pika{
		Commands: []*PikaCommand{rc},
		clients:  []Client{tc},
	}

	err := r.gatherCommandValues(tc, &acc)
	require.NoError(t, err)

	fields := map[string]interface{}{
		redisListKey: 2,
	}

	acc.AssertContainsFields(t, "pika_commands", fields)
}

func TestRedis_ParseMetrics(t *testing.T) {
	var acc testutil.Accumulator
	tags := map[string]string{"host": "pika.net"}
	rdr := bufio.NewReader(strings.NewReader(testOutput))

	err := gatherInfoOutput(rdr, &acc, tags)
	require.NoError(t, err)

	tags = map[string]string{"host": "pika.net", "replication_role": "master"}
	fields := map[string]interface{}{
		"arch_bits":                       int64(64),
		"clients":                         int64(1),
		"compression":                     string("snappy"),
		"connected_slaves":                int64(1),
		"db_fatal":                        int64(0),
		"db_memtable_usage":               int64(32000),
		"db_size":                         int64(26109074480),
		"db_tablereader_usage":            int64(842546990),
		"instantaneous_ops_per_sec":       int64(0),
		"is_bgsaving":                     int64(0),
		"is_compact":                      int64(0),
		"is_scaning_keyspace":             int64(0),
		"log_size":                        int64(998386500),
		"pika_version":                    "3.4.0",
		"process_id":                      int64(70873),
		"server_id":                       int64(1),
		"sync_thread_num":                 int64(6),
		"tcp_port":                        int64(9221),
		"thread_num":                      int64(4),
		"total_commands_processed":        int64(513085913),
		"total_connections_received":      int64(100551),
		"uptime":                          int64(2219496),
		"used_cpu_sys":                    float64(114714.65),
		"used_cpu_sys_children":           float64(0.01),
		"used_cpu_user":                   float64(83025.21),
		"used_cpu_user_children":          float64(0.01),
		"used_memory":                     int64(842578990),
	}
	acc.AssertContainsTaggedFields(t, "pika", fields, tags)

	keyspaceTags := map[string]string{"host": "pika.net", "replication_role": "master", "database": "db0", "key_type": "strings"}
	keyspaceFields := map[string]interface{}{
		"key":          int64(400000080),
		"invalid_keys": int64(0),
		"expires":      int64(0),
	}
	acc.AssertContainsTaggedFields(t, "pika_keyspace", keyspaceFields, keyspaceTags)

	cmdstatSetTags := map[string]string{"host": "pika.net", "command": "info"}
	cmdstatSetFields := map[string]interface{}{
		"info":         int64(212545),
	}
	acc.AssertContainsTaggedFields(t, "pika_cmdstat", cmdstatSetFields, cmdstatSetTags)

	replicationTags := map[string]string{
		"host":             "pika.net",
		"pika_role":        "slave",
		"pika_ip":          "pika.slave.net",
		"pika_port":        "9221",
		"replication_role": "master",
	}
	replicationFields := map[string]interface{}{
		"conn_fd":int64(120),
		"db0":    int64(0),
		"db1":    int64(0),
		"db2":    int64(0),
		"db3":    int64(0),
	}

	acc.AssertContainsTaggedFields(t, "pika_replication", replicationFields, replicationTags)
}

func TestRedis_ParseFloatOnInts(t *testing.T) {
        var acc testutil.Accumulator
        tags := map[string]string{"host": "redis.net"}
        rdr := bufio.NewReader(strings.NewReader(strings.Replace(testOutput, "used_cpu_sys:114714.65", "used_cpu_sys:114715", 1)))
        err := gatherInfoOutput(rdr, &acc, tags)
        require.NoError(t, err)
        var m *testutil.Metric
        for i := range acc.Metrics {
                if _, ok := acc.Metrics[i].Fields["used_cpu_sys"]; ok {
                        m = acc.Metrics[i]
                        break
                }
        }
        require.NotNil(t, m)
        usedCpuSys, ok := m.Fields["used_cpu_sys"]
        require.True(t, ok)
        require.IsType(t, float64(0.0), usedCpuSys)
}

func TestRedis_ParseStringOnInts(t *testing.T) {
        var acc testutil.Accumulator
        tags := map[string]string{"host": "redis.net"}
        rdr := bufio.NewReader(strings.NewReader(strings.Replace(testOutput, "is_bgsaving:No-", "is_bgsaving:0", 1)))
        err := gatherInfoOutput(rdr, &acc, tags)
        require.NoError(t, err)
        var m *testutil.Metric
        for i := range acc.Metrics {
                if _, ok := acc.Metrics[i].Fields["is_bgsaving"]; ok {
                        m = acc.Metrics[i]
                        break
                }
        }
        require.NotNil(t, m)
        bgSaving, ok := m.Fields["is_bgsaving"]
        require.True(t, ok)
        require.IsType(t, int64(0), bgSaving)
}

const testOutput = `# Server
pika_version:3.4.0
pika_git_sha:
pika_build_compile_date: May 13 2021
os:Linux 3.10.0-957.21.2.el7.x86_64 x86_64
arch_bits:64
process_id:70873
tcp_port:9221
thread_num:4
sync_thread_num:6
uptime_in_seconds:2219496
uptime_in_days:27
config_file:/opt/pika/conf/pika.conf
server_id:1

# Data
db_size:26109074480
db_size_human:24899M
log_size:998386500
log_size_human:952M
compression:snappy
used_memory:842578990
used_memory_human:803M
db_memtable_usage:32000
db_tablereader_usage:842546990
db_fatal:0
db_fatal_msg:NULL

# Clients
connected_clients:1

# Stats
total_connections_received:100551
instantaneous_ops_per_sec:0
total_commands_processed:513085913
is_bgsaving:No
is_scaning_keyspace:No
is_compact:No
compact_cron:02-04/20
compact_interval:

# Command_Exec_Count
CLIENT:1
INFO:212545
TCMALLOC:1
FLUSHDB:1
SELECT:5
AUTH:100402
GET:4
SET:412772720
CONFIG:169
COMPACT:1
BGSAVE:19
SLOWLOG:1
KEYS:7
SETEX:100000037

# CPU
used_cpu_sys:114714.65
used_cpu_user:83025.21
used_cpu_sys_children:0.01
used_cpu_user_children:0.01

# Replication(MASTER)
role:master
connected_slaves:1
slave0:ip=pika.slave.net,port=9221,conn_fd=120,lag=(db0:0)(db1:0)(db2:0)(db3:0)
db0 binlog_offset=813 29855587,safety_purge=write2file803
db1 binlog_offset=0 80,safety_purge=none
db2 binlog_offset=0 80,safety_purge=none
db3 binlog_offset=0 0,safety_purge=none

# Keyspace
# Time:2021-05-31 12:34:17
# Duration: 360s
db0 Strings_keys=400000080, expires=0, invalid_keys=0
db0 Hashes_keys=0, expires=0, invalid_keys=0
db0 Lists_keys=0, expires=0, invalid_keys=0
db0 Zsets_keys=0, expires=0, invalid_keys=0
db0 Sets_keys=0, expires=0, invalid_keys=0

# Time:2021-05-31 12:40:17
# Duration: 0s
db1 Strings_keys=1, expires=0, invalid_keys=0
db1 Hashes_keys=0, expires=0, invalid_keys=0
db1 Lists_keys=0, expires=0, invalid_keys=0
db1 Zsets_keys=0, expires=0, invalid_keys=0
db1 Sets_keys=0, expires=0, invalid_keys=0

# Time:2021-05-31 12:40:17
# Duration: 0s
db2 Strings_keys=1, expires=0, invalid_keys=0
db2 Hashes_keys=0, expires=0, invalid_keys=0
db2 Lists_keys=0, expires=0, invalid_keys=0
db2 Zsets_keys=0, expires=0, invalid_keys=0
db2 Sets_keys=0, expires=0, invalid_keys=0

# Time:2021-05-31 12:40:17
# Duration: 0s
db3 Strings_keys=0, expires=0, invalid_keys=0
db3 Hashes_keys=0, expires=0, invalid_keys=0
db3 Lists_keys=0, expires=0, invalid_keys=0
db3 Zsets_keys=0, expires=0, invalid_keys=0
db3 Sets_keys=0, expires=0, invalid_keys=0`
