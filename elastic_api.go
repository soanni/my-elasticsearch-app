package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	es7 "github.com/elastic/go-elasticsearch/v7"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func getElasticClient() *es7.Client {
	cfg := es7.Config{
		Addresses: []string{
			fmt.Sprintf("http://%s:%s", viper.GetString("elasticServer"), viper.GetString("elasticPort")),
		},
		Username:  viper.GetString("elasticUser"),
		Password:  viper.GetString("elasticPass"),
		Transport: &http.Transport{},
	}

	client, err := es7.NewClient(cfg)
	if err != nil {
		log.Fatalf("*** Error creating the client: %s ***\n", err)
	}

	return client
}

// cname - container name
// esIndexName - elasticsearch index name
func getContainerStats(cname string, esIndexName string, esClient *es7.Client) {
	var search_stats string = `{
    "aggs": {
        "by_container": {
            "aggs": {
                "average": {
                    "max": {
                        "field": "avg"
                    }
                },
				"max": {
                    "max": {
                        "field": "max"
                    }
                },
				"min": {
                    "max": {
                        "field": "min"
                    }
                }
            },
            "composite": {
                "size": 10000,
                "sources": [
                    {
                        "category": {
                            "terms": {
                                "field": "cat.keyword",
                                "missing_bucket": true
                            }
                        }
                    },
                    {
                        "container_name": {
                            "terms": {
                                "field": "cname.keyword",
                                "order": "asc"
                            }
                        }
                    },
					{
						"cpuPeriod":{
							"terms": {
								"field": "container.cpuPeriod"
							}
						}
					},
					{
						"cpuQuota":{
							"terms": {
								"field": "container.cpuQuota"
							}
						}
					},
					{
						"memLimit":{
							"terms": {
								"field": "container.memLimit"
							}
						}
					}
                ]
            }
        }
    },
    "query": {
        "bool": {
			"must": {
				"term": {
					"cname.keyword" : "%s"
				}
			},
			"filter": {
				"range": {
					"metricStartDate": {
                		"gte": "now-%s/d",
                    	"lte": "now/d"
					}
				}
			}
        }
    },
    "size": 0
}`

	var jsonStr = []byte(fmt.Sprintf(search_stats, cname, viper.GetInt("periodDays")))

	cli := getElasticClient()

	// Perform the search request.
	res, err := cli.Search(
		cli.Search.WithContext(context.Background()),
		cli.Search.WithIndex(esIndexName),
		cli.Search.WithBody(bytes.NewBuffer(jsonStr)),
		cli.Search.WithTrackTotalHits(true),
		cli.Search.WithPretty(),
	)

	// Handle connection errors
	//
	if err != nil {
		log.Fatalf("*** ERROR: %v ***\n", err)
	}
	defer res.Body.Close()

	// Handle error response (4xx, 5xx)
	//
	if res.IsError() {
		log.Fatalf("*** ERROR: %s *** \n", res.Status())
	}

	// Handle successful response (2xx)
	//
	log.Println(res)
}
