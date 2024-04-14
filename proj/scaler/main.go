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
	apiClient, err := client.NewClientWithOpts(client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	defer apiClient.Close()

	containers, err := apiClient.ContainerList(context.Background(), container.ListOptions{All: true})
	if err != nil {
		panic(err)
	}

	for _, ctr := range containers {
		if strings.Compare(ctr.State, "running") != 0 {
			continue
		}

		fmt.Printf("Container %s %s (status: %s) (state: %s)\n", ctr.Names[0], ctr.Image, ctr.Status, ctr.State)

		containerStats, err := apiClient.ContainerStats(context.Background(), ctr.ID, false)
		if err != nil {
			panic(err)
		}
		defer containerStats.Body.Close()

		buf := new(bytes.Buffer)

		buf.ReadFrom(containerStats.Body)

		var metrics Metrics
		stats := utils.StatsParser(buf.Bytes(), metrics)

		utils.PrettyPrint(stats)
	}
}


