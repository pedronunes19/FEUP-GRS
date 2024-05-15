// Implements the auto scaler for containerized applications
package scaler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"log"
	"strings"
	"sync"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"

	. "grs/common/types"
	utils "grs/common/utils"
)

func Run(s *sync.WaitGroup, config *Config, c chan []*Stats, ct *context.Context) {
	apiClient, err := client.NewClientWithOpts(client.WithAPIVersionNegotiation())

	if err != nil {
		log.Fatalln("In scaler.Run: Failed to create Docker API Client")
	}
	defer apiClient.Close()

	ctx, cancel := context.WithCancel(*ct)

	stats := <-c

	for range stats {
		//fmt.Println("Printing stats received from Metric Collector")
		//utils.PrettyPrint(stat)
	}

	s.Done()
	
	startErr := startContainer(utils.GRS_IMAGE, apiClient, &ctx)
	if startErr != nil {
		cancel()
		log.Fatalln(startErr.Error())
	}
	startContainer(utils.GRS_IMAGE, apiClient, &ctx)

	time.Sleep(5 * time.Second)
	

	stopErr := stopContainer(apiClient, &ctx)
	if stopErr != nil {
		cancel()
		log.Fatalln(stopErr.Error())
	}

	s.Done()
	cancel()
}

// Starts a container provided an image name
func startContainer(imageName string, cl *client.Client, ctx *context.Context) error {

	networkID, err := utils.GetNetworkID(utils.GRS_NETWORK, cl, ctx)

	if err != nil {
		return errors.New(fmt.Sprintf("In startContainer: Failed to get ID of network %s", utils.GRS_NETWORK))
	}

	netconf := make(map[string]*network.EndpointSettings)
	netconf[utils.GRS_NETWORK] = &network.EndpointSettings{
		NetworkID: *networkID,
	}

	response, createErr := cl.ContainerCreate(*ctx,
		&container.Config{
			Tty: false,
			Image: imageName,

		}, nil, 
		
		&network.NetworkingConfig{
			EndpointsConfig: netconf,
		}, 
		
		nil, "",
	)

	if createErr != nil {
		return errors.New(fmt.Sprintf("In startContainer: Failed to create container -> %s", createErr.Error()))
	}
	
	startErr := cl.ContainerStart(*ctx, response.ID, container.StartOptions{})
	
	if startErr != nil {
		return errors.New(fmt.Sprintf("In startContainer: Failed to start container with ID %s -> %s", response.ID, startErr.Error()))
	}

	return nil
}

// Stops the container with less usage
func stopContainer(cl *client.Client, ctx *context.Context) error {

	grsContainers, err := utils.GetContainersOnNetwork(utils.GRS_NETWORK, cl, ctx)

	if err != nil {
		return err
	}

	allStats := map[string]Stats{}

	
	for _, ctr := range *grsContainers {
		if strings.Compare(ctr.Name, utils.GRS_LOAD_BALANCER) == 0 { // Don't add the Load Balancer to the map, since we will never want to stop it
			continue
		}

		stats, err := utils.GetContainerStats(ctr.Name, cl, ctx)

		if err != nil {
			return err
		}
		
		allStats[ctr.Name] = *stats
	}

	/*
	allStats["Less CPU Usage"] = Stats{
		UsedMemory: 1000,
		AvailableMemory: 2000,
		MemoryUsage: "50%",
		NumberOfCPUs: 16,
		CPUUsage: "10%",
	}

	allStats["Most CPU Usage"] = Stats{
		UsedMemory: 1000,
		AvailableMemory: 2000,
		MemoryUsage: "30%",
		NumberOfCPUs: 16,
		CPUUsage: "70%",
	}*/

	sorted := utils.SortContainersByUsage(allStats, false)

	for i, v := range(sorted) {
		fmt.Printf("%d: %s\n", i, v)
	}

	leastUsedContainer := sorted[0]

	containerID, err := utils.GetContainerID(leastUsedContainer, cl, ctx)

	if err != nil {
		return err
	}

	stopErr := cl.ContainerStop(*ctx, *containerID, container.StopOptions{})

	if stopErr != nil {
		return errors.New(fmt.Sprintf("In stopContainer: Failed to stop container -> %s", stopErr.Error()))
	}

	return nil
}
