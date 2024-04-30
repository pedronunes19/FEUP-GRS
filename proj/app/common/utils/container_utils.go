package utils

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"

	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"

	"github.com/tufanbarisyildirim/gonginx/config"
	"github.com/tufanbarisyildirim/gonginx/dumper"
	"github.com/tufanbarisyildirim/gonginx/parser"

	. "grs/common/types"
)

// Returns the stats of container with name containerName
func GetContainerStats(containerName string, cl *client.Client, ctx *context.Context) (*Stats, error) {

	containerID, err := GetContainerID(containerName, cl, ctx)

	if err != nil {
		return nil, err
	}

	metrics, err := cl.ContainerStats(*ctx, *containerID, false)

	defer metrics.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(metrics.Body)

	stats, err := StatsParser(buf.Bytes())

	if err != nil {
		return nil, err
	}

	return stats, nil
}

// Returns the ID of a container with name containerName
func GetContainerID(containerName string, cl *client.Client, ctx *context.Context) (*string, error) {
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
func GetNetworkID(networkName string, cl *client.Client, ctx *context.Context) (*string, error) {

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
func GetContainersOnNetwork(networkName string, cl *client.Client, ctx *context.Context) (*map[string]types.EndpointResource, error) {

	networkID, err := GetNetworkID(networkName, cl, ctx)

	if err != nil {
		return nil, err
	}

	networkInfo, err := cl.NetworkInspect(*ctx, *networkID, types.NetworkInspectOptions{})

	if err != nil {
		return nil, errors.New(fmt.Sprintf("In getContainersOnNetwork: Failed to inspect network with name %s", networkName))
	}

	return &networkInfo.Containers, nil
}

// Returns a list with the containers names sorted by CPU or Memory Usage
func SortContainersByUsage(allStats map[string]Stats, byCPUUsage bool) []string {

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

// Updates Nginx config file and send signal to update the service
func UpdateNginxConfig(newConf string, cl *client.Client, ctx *context.Context) error {
	
	f, openErr := os.Create(NGINX_CONFIG_PATH)

	if openErr != nil {
		return errors.New(fmt.Sprintf("In UpdateNginxConfig: Failed to create/open config file -> %s", openErr.Error()))
	}

	_, writeErr := f.WriteString(newConf)

	if writeErr != nil {
		return errors.New(fmt.Sprintf("In UpdateNginxConfig: Failed to write to configfile -> %s", writeErr.Error()))
	}

	_, execErr := cl.ContainerExecCreate(*ctx, GRS_LOAD_BALANCER, types.ExecConfig{
		Tty: false,
		Cmd: []string {"kill", "-1", "1"},
	})

	if execErr != nil {
		return errors.New(fmt.Sprintf("In UpdateNginxConfig: Failed to send signal to Nginx -> %s", execErr.Error()))
	}

	return nil
}

func AddNewServer(newServer string, cl *client.Client, ctx *context.Context) error {

	oldConf, openErr := openNginxConfigFile()
	if openErr != nil {
		return openErr
	}

	p := parser.NewStringParser(*oldConf)

	conf, err := p.Parse()
	if err != nil {
		return errors.New(fmt.Sprintf("In AddNewServer: Failed to parse Nginx old config -> %s", err.Error()))
	}

	upstreams := conf.FindUpstreams()

	upstreams[0].AddServer(&config.UpstreamServer{
		Address: fmt.Sprintf("%s:80", newServer),
	})

	newConf := dumper.DumpBlock(conf.Block, dumper.IndentedStyle)

	fmt.Println(newConf)

	UpdateNginxConfig(newConf, cl, ctx)
	
	return nil
}

func RemoveServer(serveToRemove string, cl *client.Client, ctx *context.Context) error {

	oldConf, openErr := openNginxConfigFile()
	if openErr != nil {
		return openErr
	}

	p := parser.NewStringParser(*oldConf)

	conf, err := p.Parse()
	if err != nil {
		return errors.New(fmt.Sprintf("In RemoveServer: Failed parse Nginx old config -> %s", err.Error()))
	}

	directives := conf.FindDirectives("upstream")

	for _, dir := range directives {
		fmt.Println("Found directive ", dir.GetName(), dir.GetParameters())

		servers := dir.GetBlock().GetDirectives()
		for _, server := range servers {
			fmt.Println(server.GetName(), server.GetParameters()[0])
			if server.GetParameters()[0] == fmt.Sprintf("%s:80", serveToRemove) {
				fmt.Println("apagar o peras")
				server.GetParameters()[0] = ""
				
			} 
		}
	}

	fmt.Printf(dumper.DumpBlock(conf.Block, dumper.IndentedStyle))
	
	return nil
}

func openNginxConfigFile() (*string, error) {
	f, openErr := os.Open(NGINX_CONFIG_PATH)

	if openErr != nil {
		return nil, errors.New(fmt.Sprintf("In AddNewServer: Failed to open Nginx old config -> %s", openErr.Error()))
	}

	fileInfo, statErr := f.Stat()
	if statErr != nil {
		return nil, errors.New(fmt.Sprintf("In AddNewServer: Failed to get Nginx old config file info -> %s", statErr.Error()))
	}

	buf := make([]byte, fileInfo.Size()) 
	f.Read(buf)

	res := string(buf)

	return &res, nil
}