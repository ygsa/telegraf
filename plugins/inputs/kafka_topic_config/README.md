# Kafka Topic Config Input Plugin

Get kafka topic's config 

### Configuration:

```toml

[[inputs.kafka_topic_config]]
  ## An array of kafka brokers.
  brokers = ["localhost:9092"]

  ## An array of kafka topics conf. 
  ## When empty will gather all 
  ## example: include_conf = ["compression.type", "min.insync.replicas", "message.downconversion.enable", "segment.jitter.ms", 
                              "cleanup.policy", "flush.ms", "segment.bytes", "retention.ms", "flush.messages", 
                              "message.format.version", "max.compaction.lag.ms", "file.delete.delay.ms", "max.message.bytes",
                              "min.compaction.lag.ms", "message.timestamp.type", "preallocate", "index.interval.bytes", 
                              "min.cleanable.dirty.ratio", "unclean.leader.election.enable", "retention.bytes", 
                              "delete.retention.ms", "segment.ms", "message.timestamp.difference.max.ms", "segment.index.bytes"]
  include_conf = []

  ## Optional Client id
  # client_id = "Telegraf"

  ## Set the minimal supported Kafka version.  Setting this enables the use of new
  ## Kafka features and APIs.  Must be 0.10.2.0 or greater.
  ##   ex: version = "1.1.0"
  # version = ""

  ## Optional TLS Config
  # tls_ca = "/etc/telegraf/ca.pem"
  # tls_cert = "/etc/telegraf/cert.pem"
  # tls_key = "/etc/telegraf/key.pem"
  ## Use TLS but skip chain & host verification
  # insecure_skip_verify = false

  ## SASL authentication credentials.  These settings should typically be used
  ## with TLS encryption enabled
  # sasl_username = "kafka"
  # sasl_password = "secret"

  ## Optional SASL:
  ## one of: OAUTHBEARER, PLAIN, SCRAM-SHA-256, SCRAM-SHA-512, GSSAPI
  ## (defaults to PLAIN)
  # sasl_mechanism = ""

  ## used if sasl_mechanism is GSSAPI (experimental)
  # sasl_gssapi_service_name = ""
  # ## One of: KRB5_USER_AUTH and KRB5_KEYTAB_AUTH
  # sasl_gssapi_auth_type = "KRB5_USER_AUTH"
  # sasl_gssapi_kerberos_config_path = "/"
  # sasl_gssapi_realm = "realm"
  # sasl_gssapi_key_tab_path = ""
  # sasl_gssapi_disable_pafxfast = false

  ## used if sasl_mechanism is OAUTHBEARER (experimental)
  # sasl_access_token = ""

  ## SASL protocol version.  When connecting to Azure EventHub set to 0.
  # sasl_version = 1

```
### Measurements & Fields:

All boolean config will turn to integer true = 1, false = 0.
- kafka_topic_config
  - compression_type (integer, producer = 1, gzip = 2, snappy = 3, lz4 = 4, uncompressed = 5, unknown = 0)
  - min_insync_replicas
  - message_downconversion_enable
  - segment_jitter_ms
  - cleanup_policy (integer, delete = 1, compact = 2, unknown = 0)
  - flush_ms
  - segment_bytes
  - retention_ms
  - flush_messages
  - message_format_version
  - max_compaction_lag_ms
  - file_delete_delay_ms
  - max_message_bytes
  - min_compaction_lag_ms
  - message_timestamp_type (integer, LogAppendTime = 1, CreateTime = 2, unknown = 0)
  - preallocate
  - index_interval_bytes
  - min_cleanable_dirty_ratio
  - unclean_leader_election_enable
  - retention_bytes
  - delete_retention_ms
  - segment_ms
  - message_timestamp_difference_max_ms
  - segment_index_bytes
  

### tags:

- All measurements have the following tags:
  - topic
  - cluster (if cluster_name not nil)


### Example Output:
```shell
$ telegraf --config telegraf.conf --input-filter kafka_topic_config --test 
> kafka_topic_config,topic=test_topic1 cleanup_policy=1i,compression_type=1i,delete_retention_ms=86400000,file_delete_delay_ms=60000,flush_messages=9223372036854776000,flush_ms=9223372036854776000,follower_replication_throttled_replicas=0,index_interval_bytes=4096,leader_replication_throttled_replicas=0,max_compaction_lag_ms=9223372036854776000,max_message_bytes=1048588,message_downconversion_enable=1,message_format_version=0,message_timestamp_difference_max_ms=9223372036854776000,message_timestamp_type=0i,min_cleanable_dirty_ratio=0.5,min_compaction_lag_ms=0,min_insync_replicas=1,preallocate=0,retention_bytes=-1,retention_ms=604800000,segment_bytes=1073741824,segment_index_bytes=10485760,segment_jitter_ms=0,segment_ms=604800000,unclean_leader_election_enable=0 1679480965000000000
> kafka_topic_config,topic=test_topic2 cleanup_policy=1i,compression_type=1i,delete_retention_ms=86400000,file_delete_delay_ms=60000,flush_messages=9223372036854776000,flush_ms=9223372036854776000,follower_replication_throttled_replicas=0,index_interval_bytes=4096,leader_replication_throttled_replicas=0,max_compaction_lag_ms=9223372036854776000,max_message_bytes=1048588,message_downconversion_enable=1,message_format_version=0,message_timestamp_difference_max_ms=9223372036854776000,message_timestamp_type=0i,min_cleanable_dirty_ratio=0.5,min_compaction_lag_ms=0,min_insync_replicas=1,preallocate=0,retention_bytes=-1,retention_ms=604800000,segment_bytes=1073741824,segment_index_bytes=10485760,segment_jitter_ms=0,segment_ms=604800000,unclean_leader_election_enable=0 1679480965000000000
> kafka_topic_config,topic=test_topic3 cleanup_policy=1i,compression_type=1i,delete_retention_ms=86400000,file_delete_delay_ms=60000,flush_messages=9223372036854776000,flush_ms=9223372036854776000,follower_replication_throttled_replicas=0,index_interval_bytes=4096,leader_replication_throttled_replicas=0,max_compaction_lag_ms=9223372036854776000,max_message_bytes=1048588,message_downconversion_enable=1,message_format_version=0,message_timestamp_difference_max_ms=9223372036854776000,message_timestamp_type=0i,min_cleanable_dirty_ratio=0.5,min_compaction_lag_ms=0,min_insync_replicas=1,preallocate=0,retention_bytes=-1,retention_ms=604800000,segment_bytes=1073741824,segment_index_bytes=10485760,segment_jitter_ms=0,segment_ms=604800000,unclean_leader_election_enable=0 1679480965000000000
> kafka_topic_config,topic=test_topic4 cleanup_policy=2i,compression_type=1i,delete_retention_ms=86400000,file_delete_delay_ms=60000,flush_messages=9223372036854776000,flush_ms=9223372036854776000,follower_replication_throttled_replicas=0,index_interval_bytes=4096,leader_replication_throttled_replicas=0,max_compaction_lag_ms=9223372036854776000,max_message_bytes=1048588,message_downconversion_enable=1,message_format_version=0,message_timestamp_difference_max_ms=9223372036854776000,message_timestamp_type=0i,min_cleanable_dirty_ratio=0.5,min_compaction_lag_ms=0,min_insync_replicas=1,preallocate=0,retention_bytes=-1,retention_ms=604800000,segment_bytes=104857600,segment_index_bytes=10485760,segment_jitter_ms=0,segment_ms=604800000,unclean_leader_election_enable=0 1679480965000000000
```