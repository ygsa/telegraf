package kafka_topic_config

import (
	"github.com/Shopify/sarama"
	"github.com/influxdata/telegraf/plugins/common/kafka"
	"github.com/influxdata/telegraf/testutil"
	"testing"
)

func TestGather(t *testing.T) {

	seedBroker := sarama.NewMockBroker(t, 1)
	defer seedBroker.Close()
	seedBroker.SetHandlerByMap(map[string]sarama.MockResponse{
		"MetadataRequest": sarama.NewMockMetadataResponse(t).
			SetController(seedBroker.BrokerID()).
			SetLeader("test_topic", 1, seedBroker.BrokerID()).
			SetBroker(seedBroker.Addr(), seedBroker.BrokerID()),
		"DescribeConfigsRequest": sarama.NewMockDescribeConfigsResponse(t),
	})
	var acc testutil.Accumulator

	KTC := KafkaTopicConfig{
		Brokers:     []string{seedBroker.Addr()},
		IncludeConf: []string{},
		ReadConfig:  kafka.ReadConfig{},
	}
	e := KTC.Gather(&acc)
	if e != nil {
		t.Fatal(e)
	}
	ktcTags := map[string]string{"topic": "test_topic"}
	ktcFields := map[string]interface{}{
		"max_message_bytes":  float64(1000000),
		"retention_ms":       float64(5000),
		"password":           float64(12345),
		"num_partitions":     float64(1),
		"replication_factor": float64(1),
	}

	acc.AssertContainsTaggedFields(t, "kafka_topic_config", ktcFields, ktcTags)

}
