package kafka_topic_config

import (
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/common/kafka"
	"github.com/influxdata/telegraf/plugins/inputs"
	"strconv"
	"strings"
)

type KafkaTopicConfig struct {
	Brokers     []string `toml:"brokers"`
	IncludeConf []string `toml:"include_conf"`
	ClusterName string   `toml:"cluster_name"`

	kafka.ReadConfig
	config *sarama.Config
	Log    telegraf.Logger `toml:"-"`
	adm    sarama.ClusterAdmin
}

func (k *KafkaTopicConfig) Description() string {
	return "Get kafka topic config info for metrics."
}

func (k *KafkaTopicConfig) SampleConfig() string {
	return `
  ## An array of kafka brokers.
  # brokers = ["localhost:9092"]

  ## cluster_name will add to tags
  # cluster_name = "my_cluster"

  ## An array of kafka topics conf. 
  ## When empty will gather all 
  ## example: include_conf = ["compression.type", "min.insync.replicas", "message.downconversion.enable", "segment.jitter.ms", 
                              "cleanup.policy", "flush.ms", "segment.bytes", "retention.ms", "flush.messages", 
                              "message.format.version", "max.compaction.lag.ms", "file.delete.delay.ms", "max.message.bytes",
                              "min.compaction.lag.ms", "message.timestamp.type", "preallocate", "index.interval.bytes", 
                              "min.cleanable.dirty.ratio", "unclean.leader.election.enable", "retention.bytes", 
                              "delete.retention.ms", "segment.ms", "message.timestamp.difference.max.ms", "segment.index.bytes"]
  # include_conf = []

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

  `
}

func (k *KafkaTopicConfig) setConfig() error {
	c := sarama.NewConfig()
	if e := k.SetConfig(c); e != nil {
		return e
	}
	k.config = c
	return nil
}

func (k *KafkaTopicConfig) makeKafkaAdmCli() error {
	adm, e := sarama.NewClusterAdmin(k.Brokers, k.config)
	if e != nil {
		return e
	}
	k.adm = adm
	return nil
}

func (k *KafkaTopicConfig) Gather(acc telegraf.Accumulator) error {
	e := k.makeKafkaAdmCli()
	if e != nil {
		e = fmt.Errorf("Unable to get cluster admin %s ", e)
		acc.AddError(e)
		return e
	}
	defer func() {
		e = k.adm.Close()
		if e != nil {
			e = fmt.Errorf("Close Kafka Cli Err %s ", e)
			acc.AddError(e)
		}
	}()
	topics, e := k.adm.ListTopics()
	if e != nil {
		e = fmt.Errorf("Unable to list topics: %v\n", e)
		acc.AddError(e)
		return e
	}

	for topicName, topicDetail := range topics {
		cfg, e := k.adm.DescribeConfig(sarama.ConfigResource{
			Type: sarama.TopicResource,
			Name: topicName,
		})
		if e != nil {
			e = fmt.Errorf("Unable to describe config: %v\n", e)
			acc.AddError(e)
			continue
		}
		tags := map[string]string{}
		tags["topic"] = topicName
		if k.ClusterName != "" {
			tags["cluster"] = k.ClusterName
		}
		fields := map[string]interface{}{}
		fields["replication_factor"] = float64(topicDetail.ReplicationFactor)
		fields["num_partitions"] = float64(topicDetail.NumPartitions)
		for _, entry := range cfg {
			if !k.needGatherConf(entry.Name) {
				continue
			}
			fieldsN, fieldsV := k.parseKafkaConfig(entry.Name, entry.Value)
			if fieldsN == "" || fieldsV == nil {
				continue
			}
			fields[fieldsN] = fieldsV
		}
		acc.AddFields("kafka_topic_config", fields, tags)
	}
	return nil
}

func (k *KafkaTopicConfig) parseKafkaConfig(configK, configV string) (fieldK string, fieldV interface{}) {
	fieldK = strings.ReplaceAll(configK, ".", "_")
	switch configK {
	case "compression.type":
		fieldV = k.parseCompressionType(configV)
	case "message.timestamp.type":
		fieldV = k.parseMessageTsType(configV)
	case "cleanup.policy":
		fieldV = k.parseCleanupPolicy(configV)
	default:
		v, e := k.parseValue(configV)
		if e != nil {
			fieldV = nil
		}
		fieldV = v
	}
	return
}

func (k *KafkaTopicConfig) needGatherConf(confName string) bool {
	Include := false
	if len(k.IncludeConf) == 0 {
		Include = true
	}
	for _, includeK := range k.IncludeConf {
		if includeK == confName {
			Include = true
			break
		}
	}
	return Include
}

func (k *KafkaTopicConfig) parseValue(configV string) (float64, error) {
	if v, e := strconv.ParseFloat(configV, 64); e == nil {
		return v, nil
	}

	v, e := strconv.ParseBool(configV)
	if e != nil {
		return 0, e
	}
	if v {
		return 1, nil
	} else {
		return 0, nil
	}

}

func (k *KafkaTopicConfig) parseCompressionType(compressionT string) int {
	var CT int
	switch compressionT {
	case "producer":
		CT = 1
	case "gzip":
		CT = 2
	case "snappy":
		CT = 3
	case "lz4":
		CT = 4
	case "uncompressed":
		CT = 5
	}
	return CT

}

func (k *KafkaTopicConfig) parseMessageTsType(MsgTsType string) int {
	var MTT int
	switch MsgTsType {
	case "LogAppendTime":
		MTT = 1
	case "CreateTime":
		MTT = 2
	}
	return MTT
}

func (k *KafkaTopicConfig) parseCleanupPolicy(cleanupPolicy string) int {
	var CP int
	switch cleanupPolicy {
	case "delete":
		CP = 1
	case "compact":
		CP = 2
	}
	return CP
}

func init() {
	inputs.Add("kafka_topic_config", func() telegraf.Input {
		return &KafkaTopicConfig{}
	})
}
