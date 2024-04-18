package main

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"

	utils "grs/scaler/utils"
	. "grs/scaler/types"
)

func main() {
	// TODO: wrap this client
	apiClient, err := client.NewClientWithOpts(client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	defer apiClient.Close()

	ctx := context.Background()

	containers, err1 := apiClient.ContainerList(ctx, container.ListOptions{All: true})
	if err1 != nil {
		panic(err1)
	}

	for _, ctr := range containers {
		if strings.Compare(ctr.State, "running") != 0 {
			continue
		}

		fmt.Printf("Container %s %s (status: %s) (state: %s)\n", ctr.Names[0], ctr.Image, ctr.Status, ctr.State)

		// TODO: contexts do not make sense here, but if we use several go-routines (we should) we should create a "context-tree" instead of re-using the Background context
		containerCTX, cancel := context.WithCancel(ctx)

		// TODO: wrap this call
		containerStats, err := apiClient.ContainerStats(containerCTX, ctr.ID, false)
		if err != nil {
			cancel()
			panic(err)
		}
		defer containerStats.Body.Close()

		buf := new(bytes.Buffer)

		buf.ReadFrom(containerStats.Body)

		var metrics Metrics
		if err, stats := utils.StatsParser(buf.Bytes(), metrics); err == nil {
			utils.PrettyPrint(stats)
		}
	}
}


