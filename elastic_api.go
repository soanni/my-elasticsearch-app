package main

import (
	es7 "github.com/elastic/go-elasticsearch/v7"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func getElasticClient() *es7.Client {
	cfg := es7.Config{
		Addresses: []string{
			fmt.Sprintf("http://%s", viper.GetStringMap("elasticServer")),
		},
		Username:  viper.GetStringMap("elasticUser"),
		Password:  viper.GetStringMap("elasticPass"),
		Transport: &http.Transport{},
	}

	client, err := es7.NewClient(cfg)
	if err != nil {
		log.Fatalf("Error creating the client: %s\n", err)
	}

	return client
}
