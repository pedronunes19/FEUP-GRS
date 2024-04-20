package scaler

import (
	"context"
	"fmt"
	"time"

	//"fmt"
	"log"
	"strings"
	"sync"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"

	. "grs/common/types"
	//utils "grs/common/utils"
)

func Run(s *sync.WaitGroup, config *Config, c chan []*Stats) {
	apiClient, err := client.NewClientWithOpts(client.WithAPIVersionNegotiation())

	if err != nil {
		log.Fatalln("Failed to create Docker API Client")
	}
	defer apiClient.Close()

	ctx := context.Background()

	stats := <-c

	for range stats {
		//fmt.Println("Printing stats received from Metric Collector")
		//utils.PrettyPrint(stat)
	}

	startContainer("grs_service", apiClient, &ctx)

	time.Sleep(5 * time.Second)

	stopContainer("grs_service", apiClient, &ctx)

	s.Done()
}

func startContainer(containerName string, cl *client.Client, ctx *context.Context) {

	containerID := getContainerID(containerName, cl, ctx)
	
	startErr := cl.ContainerStart(*ctx, containerID, container.StartOptions{})

	if startErr != nil {
		panic(startErr)
	}

	log.Printf("Started new container %s", containerName)
}

func stopContainer(containerName string, cl *client.Client, ctx *context.Context) {

	containerID := getContainerID(containerName, cl, ctx)

	stopErr := cl.ContainerStop(*ctx, containerID, container.StopOptions{})

	if stopErr != nil {
		panic(stopErr)
	}

	log.Printf("Stopped container %s", containerName)
}

func getContainerID(containerName string, cl *client.Client, ctx *context.Context) (string) {
	containers, _ := cl.ContainerList(*ctx, container.ListOptions{All: true})

	var containerID string

	for _, c := range containers {
		if strings.Compare(c.Names[0], fmt.Sprintf("/%s", containerName)) == 0 {
			containerID = c.ID
		} 
	}

	if containerID == "" {
		log.Printf("No container named %s was found\n", containerName)
	}

	return containerID
}