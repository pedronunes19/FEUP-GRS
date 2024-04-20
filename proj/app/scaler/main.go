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

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"

	. "grs/common/types"
	//utils "grs/common/utils"
)

const GRS_NETWORK string = "grs-net"
const GRS_IMAGE string = "grs"

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

	time.Sleep(5 * time.Second)

	stopErr := stopContainer("grs_service", apiClient, &ctx)
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

func stopContainer(containerName string, cl *client.Client, ctx *context.Context) error {

	containerID, err := getContainerID(containerName, cl, ctx)

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
