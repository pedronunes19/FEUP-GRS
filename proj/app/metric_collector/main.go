// Implements a container application metric collector
package metric_collector

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/docker/docker/api/types/container"
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

	containers, err := apiClient.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		cancel()
		return errors.New(fmt.Sprintf("In metric_collector.Run: Failed to get containers -> %s", err))
	}

	var allMetrics []*Stats

	for _, ctr := range containers {
		if strings.Compare(ctr.State, "running") != 0 {
			continue
		}

		fmt.Printf("Container %s %s (status: %s) (state: %s)\n", ctr.Names[0], ctr.Image, ctr.Status, ctr.State)

		// TODO: wrap this call
		containerStats, err := apiClient.ContainerStats(ctx, ctr.ID, false)
		if err != nil {
			cancel()
			return errors.New(fmt.Sprintf("In metric_collector.Run: Failed to get container stats -> %s", err))
		}
		defer containerStats.Body.Close()

		buf := new(bytes.Buffer)

		buf.ReadFrom(containerStats.Body)

		var metrics Metrics
		if err, stats := utils.StatsParser(buf.Bytes(), metrics); err == nil {
			utils.PrettyPrint(stats)
			allMetrics = append(allMetrics, stats)
		}
	}

	c <- allMetrics

	cancel()
	s.Done()

	return nil
}
