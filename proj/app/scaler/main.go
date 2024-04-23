// Implements the auto scaler for containerized applications
package scaler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"log"
	"strings"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"

	. "grs/common/types"
	utils "grs/common/utils"
)

const GRS_NETWORK string = "grs-net"
const GRS_IMAGE string = "grs"
const GRS_LOAD_BALANCER string = "load_balancer"

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

	
	startErr := startContainer(GRS_IMAGE, apiClient, &ctx)
	if startErr != nil {
		cancel()
		log.Fatalln(startErr.Error())
	}
	startContainer(GRS_IMAGE, apiClient, &ctx)

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

	networkID, err := getNetworkID(GRS_NETWORK, cl, ctx)

	if err != nil {
		return errors.New(fmt.Sprintf("In startContainer: Failed to get ID of network %s", GRS_NETWORK))
	}

	netconf := make(map[string]*network.EndpointSettings)
	netconf[GRS_NETWORK] = &network.EndpointSettings{
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

	grsContainers, err := getContainersOnNetwork(GRS_NETWORK, cl, ctx)

	if err != nil {
		return err
	}

	allStats := map[string]Stats{}

	
	for _, ctr := range *grsContainers {
		if strings.Compare(ctr.Name, GRS_LOAD_BALANCER) == 0 { // Don't add the Load Balancer to the map, since we will never want to stop it
			continue
		}

		stats, err := getContainerStats(ctr.Name, cl, ctx)

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

	sorted := sortContainersByUsage(allStats, false)

	for i, v := range(sorted) {
		fmt.Printf("%d: %s\n", i, v)
	}

	leastUsedContainer := sorted[0]

	containerID, err := getContainerID(leastUsedContainer, cl, ctx)

	if err != nil {
		return err
	}

	stopErr := cl.ContainerStop(*ctx, *containerID, container.StopOptions{})

	if stopErr != nil {
		return errors.New(fmt.Sprintf("In stopContainer: Failed to stop container -> %s", stopErr.Error()))
	}

	return nil
}

// Returns the ID of a container with name containerName
func getContainerID(containerName string, cl *client.Client, ctx *context.Context) (*string, error) {
	containers, _ := cl.ContainerList(*ctx, container.ListOptions{All: true})

	var containerID string

	for _, c := range containers {
		if strings.Compare(c.Names[0], fmt.Sprintf("/%s", containerName)) == 0 {
			containerID = c.ID
		} 
	}

	if strings.Compare(containerID, "") == 0 {
		return nil, errors.New(fmt.Sprintf("In getContainerID: No container was found with name %s", containerName))
	}

	return &containerID, nil
}

// Returns the ID of a network with name networkName
func getNetworkID(networkName string, cl *client.Client, ctx *context.Context) (*string, error) {

	networks, err := cl.NetworkList(*ctx, types.NetworkListOptions{})

	if err != nil {
		return nil, errors.New("In getNetworkID: Failed to get networks") 
	}

	for _, network := range networks {
		if strings.Compare(network.Name, networkName) == 0 {
			return &network.ID, nil
		}
	}

	return nil, errors.New(fmt.Sprintf("In getNetworkID: Failed to find network with name %s", networkName))
}

// Returns the containers on a network with name networkName
func getContainersOnNetwork(networkName string, cl *client.Client, ctx *context.Context) (*map[string]types.EndpointResource, error) {

	networkID, err := getNetworkID(networkName, cl, ctx)

	if err != nil {
		return nil, err
	}

	networkInfo, err := cl.NetworkInspect(*ctx, *networkID, types.NetworkInspectOptions{})

	if err != nil {
		return nil, errors.New(fmt.Sprintf("In getContainersOnNetwork: Failed to inspect network with name %s", networkName))
	}

	return &networkInfo.Containers, nil
}

// Returns the stats of container with name containerName 
func getContainerStats(containerName string, cl *client.Client, ctx *context.Context) (*Stats, error) {

	containerID, err := getContainerID(containerName, cl, ctx)

	if err != nil {
		return nil, err
	}

	metrics, err := cl.ContainerStats(*ctx, *containerID, false)

	defer metrics.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(metrics.Body)

	stats, err := utils.StatsParser(buf.Bytes())

	if err != nil {
		return nil, err
	}

	return stats, nil
}

// Returns a list with the containers names sorted by CPU or Memory Usage
func sortContainersByUsage(allStats map[string]Stats, byCPUUsage bool) []string {

	var pl []Pair[string, Stats]

	for name, s := range allStats {
		entry := new(Pair[string, Stats])
		entry.Key = name
		entry.Value = s
		pl = append(pl, *entry)
	}

	if byCPUUsage {

		cpuUsage := func(s1, s2 Stats) bool {
			return s1.CPUUsage < s2.CPUUsage
		}
	
		By(cpuUsage).Sort(pl)

	} else {
		memoryUsage := func(s1, s2 Stats) bool {
			return s1.MemoryUsage < s2.MemoryUsage
		}
	
		By(memoryUsage).Sort(pl)
	}

	var sortedContainersNames []string

	for _, p := range pl {
		sortedContainersNames = append(sortedContainersNames, p.Key)
	}

	return sortedContainersNames
}
