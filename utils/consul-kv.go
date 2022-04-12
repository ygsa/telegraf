/*
The MIT License (MIT)

consul-kv: just get consul's key-value
arstercz<arstercz@gmail.com>

*/

package main

import (
	"flag"
	"fmt"
	"time"
	"os"
	b64 "encoding/base64"
	"encoding/json"
	"regexp"
	"strconv"
	"github.com/hashicorp/consul/api"
)

type MetaStatus struct {
	Status	int
	Flags	uint64
	Value	string
}

func main() {
	// options
	var dc, server, token, key string
	var timeout time.Duration
	var tls, meta bool
	var scheme string = "http"

	flag.StringVar(&dc, "dc", "", "the consul datacenter")
	flag.StringVar(&server, "server", "localhost:8500", "consul server address")
	flag.StringVar(&token, "token", "", "the token to access address, like env variable HTTP_CONSUL_TOKEN")
	flag.StringVar(&key, "key", "", "the key of the consul that you want get")
	flag.DurationVar(&timeout, "timeout", 5 * time.Second, "request timeout(seconds) before give up")
	flag.BoolVar(&tls, "tls", false, "whether use tls or not")
	flag.BoolVar(&meta, "meta", false, "whether get more response info or not")

	flag.Parse()

	if key == "" {
		fmt.Println("must set key option!")
		os.Exit(1)
	}

	tlsConfig := &api.TLSConfig{}
	if tls {
		scheme = "https"

		tlsConfig = &api.TLSConfig{
			CertFile: "/etc/telegraf/tls/client-cert.pem",
			KeyFile:  "/etc/telegraf/tls/client-key.pem",
			CAFile:   "/etc/telegraf/tls/ca.pem",
			InsecureSkipVerify: true,
		}
	}

	consulConfig := &api.Config{
		Datacenter:	dc,
		Address:	server,
		Token:		token,
		WaitTime:	timeout,
		Scheme:         scheme,
		TLSConfig:      *tlsConfig,
	}

	client, err := api.NewClient(consulConfig)
	if err != nil {
		fmt.Printf("error: %s\n", err.Error())
		os.Exit(2)
	}

	kv := client.KV()

	// Lookup the key
	pair, _, err := kv.Get(key, 
		&api.QueryOptions{
			AllowStale: true, 
			Token: token,
			WaitTime: timeout,
		})

	if meta {
		var status int = 599 // network connect timeout error
		var flags uint64
		var value string

		if err != nil {
			status = getResponseCode(err.Error())
		} else {
			if pair == nil {
				status = 404
			} else {
				status = 200
				flags  = pair.Flags
				value  = b64.StdEncoding.EncodeToString([]byte(pair.Value))
			}
		}

		result := MetaStatus{
			Status:	status,
			Flags:	flags,
			Value:	value,
		}
		res, err := json.Marshal(result)
		if err != nil {
			fmt.Printf("error for json data: %v", err)
			os.Exit(1)
		}

		fmt.Println(string(res))
	} else {
		if err != nil {
			fmt.Printf("error: %v\n", err.Error())
			os.Exit(4)
		}
		if pair == nil {
			fmt.Printf("Error! No key exists at: %s\n", key)
			os.Exit(1)
		}

		fmt.Printf("%s\n", pair.Value)
	}
}

func getResponseCode(line string) int {
	re := regexp.MustCompile(`response code:\s+(?P<code>\d+)$`)
	matches := re.FindStringSubmatch(line)

	code := 599
	if len(matches) > 1 {
		var err error
		code, err = strconv.Atoi(matches[1])
		if err != nil {
			code = 599
		}
	}

	return code
}
