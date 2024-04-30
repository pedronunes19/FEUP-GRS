// Implements a container application metric collector
package metric_collector

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/docker/docker/client"

	. "grs/common/types"
	utils "grs/common/utils"
)

// Collects metrics from running containers and sends them to the Scaler through a channel
func Run(s *sync.WaitGroup, c chan []*Stats, ct *context.Context) error {
	// TODO: wrap this client
	apiClient, err := client.NewClientWithOpts(client.WithAPIVersionNegotiation())
	if err != nil {
		return errors.New(fmt.Sprintf("In metric_collector.Run: Failed to create client -> %s", err))
	}

	defer apiClient.Close()

	ctx, cancel := context.WithCancel(*ct)

	containers, err := utils.GetContainersOnNetwork(utils.GRS_NETWORK, apiClient, &ctx)
	if err != nil {
		cancel()
		return errors.New(fmt.Sprintf("In metric_collector.Run: Failed to get containers -> %s", err))
	}

	var allMetrics []*Stats

	for _, ctr := range *containers {
		cStats, _ := utils.GetContainerStats(ctr.Name, apiClient, &ctx)

		if strings.Compare(ctr.Name, utils.GRS_LOAD_BALANCER) == 0 {
			continue
		}
		
		fmt.Printf("Container %s\n", ctr.Name)
		utils.PrettyPrint(cStats)
		allMetrics = append(allMetrics, cStats)
	}

	c <- allMetrics

	cancel()
	s.Done()

	return nil
}
