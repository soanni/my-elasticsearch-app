package main

import (
	"log"
	"net/http"
	es7 "github.com/elastic/go-elasticsearch/v7"
)

func main() {
	cfg := es7.Config{
	    Addresses: []string{
	        "http://10.69.12.196:9200",
	    },
	    Username: "***",
	    Password: "***",
	    Transport: &http.Transport{
	    },
	}

	client, err := es7.NewClient(cfg)
	if err != nil {
		log.Fatalf("Error creating the client: %s\n", err)
	}

	res, err := client.Info()
	if err != nil {
		log.Fatalf("Error getting response: %s\n", err)
	}

	log.Println(res)

	q := `{
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
	        "cpuPeriod": {
	          "terms": {
	            "field": "container.cpuPeriod"
	          }
	        }
	      },
	      {
	        "cpuQuota": {
	          "terms": {
	            "field": "container.cpuQuota"
	          }
	        }
	      },
	      {
	        "memLimit": {
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
	      "cname.keyword": "prod_xo-clients_xotimecard_2"
	    }
	  },
	  "filter": {
	    "range": {
	      "metricStartDate": {
	        "gte": "now-28d/d",
	        "lte": "now/d"
	      }
	    }
	  }
	}
	},
	"size": 0
}`

	res, err = client.Search(
		client.Search.WithContext(context.Background()),
		client.Search.WithIndex("d1_docker_metrics_test"),
		client.Search.WithBody(&buf),
		client.Search.WithTrackTotalHits(true),
		client.Search.WithPretty(),
	)

	if err != nil {
		log.Fatalf("ERROR: %s", err)
	}
	defer res.Body.Close()

	log.Println(res)

}