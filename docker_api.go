package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// getting containers that were started X days ago

func getContainersList(host string, startedAfter int64) []types.Container {
	// valid containers are those containers
	// whose startedAt field is less than period in config file
	validContainers := make([]types.Container, 0)

	log.Infof("*** Getting container list for %s \n ***", host)

	cli := getDockerClient(host)
	defer func() {
		if err := cli.Close(); err != nil {
			log.Panic(err)
		}
	}()

	runningOnlyFilter := filters.NewArgs(filters.KeyValuePair{Key: "status", Value: "running"})
	listOptions := types.ContainerListOptions{Filters: runningOnlyFilter}

	containers, err := cli.ContainerList(context.Background(), listOptions)

	if err != nil {
		log.Fatal(err)
	}

	for _, container := range containers {
		inspectedContainer := dockerContainerInspect(host, strings.TrimPrefix(container.Names[0], "/"))
		if inspectedContainer != nil {
			// Golang Time for startedAt
			startedAtTime, err := time.Parse(time.RFC3339Nano, inspectedContainer.State.StartedAt)
			log.Infof("*** Container: %s, startedAt: %v \n***", container.Names[0], startedAtTime)
			if err != nil {
				log.Infof("*** Time parsing error ***\n")
				log.Fatal(err)
			}
			// Unix timestamp for startedAt
			startedAtStamp := startedAtTime.Unix()
			if startedAtStamp < startedAfter {
				log.Infof("*** Added container %s ***\n", container.Names[0])
				validContainers = append(validContainers, container)
			} else {
				log.Infof("*** Skipped container %s as it was started recently ***\n", container.Names[0])
			}
		}
	}

	log.Infof("*** Number of containers - %d ***", len(containers))
	return validContainers
}

// for cid parameter both containerID and containerName will work https://docs.docker.com/engine/api/v1.39/#operation/ContainerInspect
func dockerContainerInspect(host string, cid string) *types.ContainerJSON {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	cli := getDockerClient(host)
	defer func() {
		if err := cli.Close(); err != nil {
			log.Panic(err)
		}
	}()

	container, err := cli.ContainerInspect(ctx, cid)
	if err != nil {
		log.Info(err)
		return nil
	}

	return &container
}

func getViperDockerTLS(hostName string) bool {
	dockerHosts := viper.GetStringMap("dockerHosts")
	var tlsEnabled bool
	if v, ok := dockerHosts[hostName]; ok {
		if rec, ok := v.(map[string]interface{}); ok {
			tlsEnabled = rec["tls"].(bool)
		}
	} else {
		err := fmt.Sprintf("*** No such docker host in viper config file: %s\n ***", hostName)
		log.Panic(err)
	}
	return tlsEnabled
}

func getDockerClient(hostName string) *client.Client {
	var (
		dockerHost   string
		dockerClient *client.Client
		err          error
		cacertPath   = viper.GetString("cacertPath")
		certPath     = viper.GetString("certPath")
		keyPath      = viper.GetString("keyPath")
	)

	if !getViperDockerTLS(hostName) {
		dockerHost = fmt.Sprintf("http://%s:2375", hostName)
		dockerClient, err = client.NewClientWithOpts(client.WithHost(dockerHost), client.WithVersion("1.35"))
	} else {
		dockerHost = fmt.Sprintf("tcp://%s:9998", hostName)
		dockerClient, err = client.NewClientWithOpts(client.WithHost(dockerHost), client.WithTLSClientConfig(cacertPath, certPath, keyPath), client.WithVersion("1.35"))
	}

	if err != nil {
		log.Panic(err)
	}

	return dockerClient
}
