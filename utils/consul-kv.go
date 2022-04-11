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
	"github.com/hashicorp/consul/api"
)

func main() {
	// options
	var dc, server, token, key string
	var timeout time.Duration
	var tls bool
	var scheme string = "http"

	flag.StringVar(&dc, "dc", "", "the consul datacenter")
	flag.StringVar(&server, "server", "localhost:8500", "consul server address")
	flag.StringVar(&token, "token", "", "the token to access address, like env variable HTTP_CONSUL_TOKEN")
	flag.StringVar(&key, "key", "", "the key of the consul that you want get")
	flag.DurationVar(&timeout, "timeout", 3 * time.Second, "request timeout(seconds) before give up")
	flag.BoolVar(&tls, "tls", false, "whether use tls or not")

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
		})
	if err != nil {
		fmt.Printf("error: %s\n", err.Error())
		os.Exit(4)
	}
	if pair == nil {
		fmt.Printf("Error! No key exists at: %s\n", key)
		os.Exit(1)
	}

	fmt.Printf("%s\n", pair.Value)
}

