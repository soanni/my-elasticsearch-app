package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"io"
	_ "io/ioutil"
	"strings"

	es7 "github.com/elastic/go-elasticsearch/v7"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/tidwall/gjson"
)

var (
	volume_stats string = `{
	  "aggs": {
	    "totalVolumeSize": {
	      "sum": {
	        "field": "mountSize"
	      }
	    }
	  },
	  "query": {
	    "bool": {
	      "must": {
	        "term": {
	          "name.keyword": "%s"
	        }    
	      },
	      "must_not": {
	        "term": {
	          "exclude": "true"
	        }
	      }, 
	      "filter": {
	        "match": {
	          "date": "%s"
	        }
	      }
	    }
	  },
	  "size": 0
	}`

	metric_stats string = `{
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
	                		"gte": "now-%dd/d",
	                    	"lte": "now/d"
						}
					}
				}
	        }
	    },
	    "size": 0
	}`
)

func getElasticClient() *es7.Client {
	var (
		elasticServer string = viper.GetString("elasticServer")
		elasticPort string = viper.GetString("elasticPort")
		elasticUser string = viper.GetString("elasticUser")
		elasticPass string = viper.GetString("elasticPass")
	)

	cfg := es7.Config{
		Addresses: []string{
			fmt.Sprintf("http://%s:%s", elasticServer, elasticPort),
		},
		Username:  elasticUser,
		Password:  elasticPass,
		Transport: &http.Transport{},
	}

	client, err := es7.NewClient(cfg)
	if err != nil {
		log.Fatalf("*** Error creating the client: %s ***\n", err)
	}

	return client
}


// func getContainerMetricStatsHttp(cname string, esIndexName string) {
// 	var (
// 		elasticServer string = viper.GetString("elasticServer")
// 		elasticPort string = viper.GetString("elasticPort")
// 		elasticUser string = viper.GetString("elasticUser")
// 		elasticPass string = viper.GetString("elasticPass")
// 		periodDays int = viper.GetInt("periodDays")
// 	)

// 	elasticUrl := fmt.Sprintf("http://%s:%s/", elasticServer, elasticPort)
// 	log.Println(elasticUrl, elasticUrl + esIndexName + "/_search")
// 	var jsonStr = []byte(fmt.Sprintf(metric_stats, cname, periodDays))
// 	log.Println(fmt.Sprintf(metric_stats, cname, periodDays))
// 	req, err := http.NewRequest("POST", elasticUrl + esIndexName + "/_search", bytes.NewBuffer(jsonStr))
// 	if err != nil {
// 		req.SetBasicAuth(elasticUser, elasticPass)
// 		req.Header.Set("Content-Type", "application/json")
// 		log.Println(req)
// 		client := &http.Client{}
// 		resp, err := client.Do(req)
// 		if err != nil {
// 			defer resp.Body.Close()
// 			body, _ := ioutil.ReadAll(resp.Body)
// 			log.Println(string(body))
// 		} else {
// 			log.Fatalf("*** ERROR 1: %v ***\n", err)
// 		}
// 	} else {
// 		log.Fatalf("*** ERROR 2: %v ***\n", err)
// 	}
// }

func getContainerVolumeStats(cname string, esIndexName string, date string, esClient *es7.Client) {
	var jsonStr = []byte(fmt.Sprintf(volume_stats, cname, date))

	// Perform the search request.
	res, err := esClient.Search(
		esClient.Search.WithContext(context.Background()),
		esClient.Search.WithIndex(esIndexName),
		esClient.Search.WithBody(bytes.NewBuffer(jsonStr)),
		esClient.Search.WithTrackTotalHits(true),
		esClient.Search.WithPretty(),
	)

	// Handle connection errors
	if err != nil {
		log.Fatalf("*** ERROR: %v ***\n", err)
	}
	defer res.Body.Close()

	// Handle error response (4xx, 5xx)
	if res.IsError() {
		log.Fatalf("*** ERROR: %s *** \n", res.Status())
	}

	// Handle successful response (2xx)
	responseJsonString := read(res.Body)
	totalVolumesSize := gjson.Get(responseJsonString, "aggregations.totalVolumeSize.value").Float()

	log.Infof("Cname: %s, Total volume size (GB): %g\n", cname, totalVolumesSize / 1024 / 1024 / 1024)
}


// cname - container name
// esIndexName - elasticsearch index name
// getting avg, max, min for net, block, cpu, memory metrics
func getContainerMetricStats(cname string, esIndexName string, esClient *es7.Client) {
	var periodDays int = viper.GetInt("periodDays")
	var jsonStr = []byte(fmt.Sprintf(metric_stats, strings.TrimPrefix(cname, "/"), periodDays))

	// Perform the search request.
	res, err := esClient.Search(
		esClient.Search.WithContext(context.Background()),
		esClient.Search.WithIndex(esIndexName),
		esClient.Search.WithBody(bytes.NewBuffer(jsonStr)),
		esClient.Search.WithTrackTotalHits(true),
		esClient.Search.WithPretty(),
	)

	// Handle connection errors
	if err != nil {
		log.Fatalf("*** ERROR: %v ***\n", err)
	}
	defer res.Body.Close()

	// Handle error response (4xx, 5xx)
	if res.IsError() {
		log.Fatalf("*** ERROR: %s *** \n", res.Status())
	}

	// Handle successful response (2xx)
	responseJsonString := read(res.Body)
	buckets := gjson.Get(responseJsonString, "aggregations.by_container.buckets")

	for _, bucket := range buckets.Array() {
		log.Infof("Cname: %s, Category: %s, Avg: %g, Min: %g, Max: %g\n", bucket.Get("key.container_name").String(), bucket.Get("key.category").String(), bucket.Get("average.value").Float(), bucket.Get("min.value").Float(), bucket.Get("max.value").Float())
	}
}

func read(r io.Reader) string {
	var b bytes.Buffer
	b.ReadFrom(r)
	return b.String()
}
