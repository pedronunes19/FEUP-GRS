package utils

import (
	"encoding/json"
	"errors"
	"fmt"

	. "grs/scaler/types"
)

func PrettyPrint(v any) {
	output, errParse := json.MarshalIndent(v, "", "  ")
	if errParse != nil {
		panic(errParse)
	}

	fmt.Println(string(output))
}

func StatsParser(data []byte, metrics Metrics) (error, *Stats) {
	parseErr := json.Unmarshal(data, &metrics)

	if parseErr != nil {
		return errors.New(""), nil
	}

	usedMemory := metrics.MemStats.Usage - metrics.MemStats.Stats.Cache
	availableMemory := metrics.MemStats.Limit
	cpuDelta := metrics.CPUStats.CPUUsage.TotalUsage - metrics.PreCPUStats.CPUUsage.TotalUsage
	systemCPUDelta := metrics.CPUStats.SystemCPUUsage - metrics.PreCPUStats.SystemCPUUsage
	numberOfCPUs := metrics.CPUStats.NumberOfCPUs
	cpuUsge := ((cpuDelta / systemCPUDelta) * float64(numberOfCPUs)) * 100.0

	stats := &Stats{
		UsedMemory:      usedMemory,
		AvailableMemory: availableMemory,
		MemoryUsage:     fmt.Sprintf("%.03f%%", (usedMemory/availableMemory)*100.0),
		NumberOfCPUs:    metrics.CPUStats.NumberOfCPUs,
		CPUUsage:        fmt.Sprintf("%.03f%%", cpuUsge),
	}

	return nil, stats
}