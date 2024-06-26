// Implements the auto scaler for containerized applications
package scaler

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"

	"log"
	"strings"
	"sync"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"

	. "grs/common/types"
	utils "grs/common/utils"
)

func Run(s *sync.WaitGroup, config *Config, stats []*Stats, ct *context.Context) {
	apiClient, err := client.NewClientWithOpts(client.WithAPIVersionNegotiation())

	if err != nil {
		log.Fatalln("In scaler.Run: Failed to create Docker API Client")
	}
	defer apiClient.Close()

	ctx, cancel := context.WithCancel(*ct)
	defer cancel()

	runningContainers, err := utils.GetContainersOnNetwork(utils.GRS_NETWORK, apiClient, &ctx)
	if err != nil {
		log.Fatalln("In scaler.Run: Failed to get containers on GRS network")
	}

	runningReplicas := len(*runningContainers) - 1 // remove load balancer

	for _, stat := range stats {
		fmt.Println("Printing stats received from Metric Collector")
		utils.PrettyPrint(stat)
		memUsage, convErr := strconv.ParseFloat(strings.Split(stat.MemoryUsage, "%")[0], 32)
		
		if convErr != nil {
			fmt.Println("Error converting string to float, skipping...")
			continue
		}

		memThreshold, convErr := strconv.ParseFloat(config.Metrics.Memory.Threshold, 32)
		if convErr != nil {
			fmt.Println("Error converting memory threshold to float, skipping...")
			continue
		}

		cpuUsage, convErr := strconv.ParseFloat(strings.Split(stat.CPUUsage, "%")[0], 32)

		cpuThreshold, convErr := strconv.ParseFloat(config.Metrics.CPU.Threshold, 32)

		desiredReplicas := max(math.Ceil(float64(runningReplicas) * (memUsage / memThreshold)), math.Ceil(float64(runningReplicas) * (cpuUsage / cpuThreshold)))

		fmt.Println(desiredReplicas, runningReplicas, memUsage, memThreshold)

		if desiredReplicas > float64(runningReplicas) {
			startContainer(utils.GRS_IMAGE, apiClient, &ctx)
			break
		}

		if desiredReplicas < float64(runningReplicas) {
			stopContainer(apiClient, &ctx)
			break
		}
	}
	
	s.Done()
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

	containerName, err := utils.GetContainerName(response.ID, cl, ctx)
	if err != nil {
		return errors.New(fmt.Sprintf("In startContainer: Failed to get container name -> %s", err.Error()))
	}

	utils.AddNewServer(*containerName, cl, ctx)

	return nil
}

// Stops the container with less usage
func stopContainer(cl *client.Client, ctx *context.Context) error {

	grsContainers, err := utils.GetContainersOnNetwork(utils.GRS_NETWORK, cl, ctx)

	if len(*grsContainers) <= 1 { // we need to have at least one container running
		return nil
	}

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

	removeErr := utils.RemoveServer(leastUsedContainer, cl, ctx)
	if removeErr != nil {
		return errors.New(fmt.Sprintf("In stopContainer: Failed to remove server -> %s", removeErr.Error()))
	}

	stopErr := cl.ContainerStop(*ctx, *containerID, container.StopOptions{})

	if stopErr != nil {
		return errors.New(fmt.Sprintf("In stopContainer: Failed to stop container -> %s", stopErr.Error()))
	}

	return nil
}
