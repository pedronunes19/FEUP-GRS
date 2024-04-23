package utils

import (
	"encoding/json"
	"errors"
	"fmt"

	. "grs/common/types"

	"gopkg.in/yaml.v3"
)

// Pretty prints JSON
func PrettyPrint(v any) error {
	output, errParse := json.MarshalIndent(v, "", "  ")

	if errParse != nil {
		return errors.New(fmt.Sprintf("In PrettyPrint: Failed to marshal data -> %s", errParse))
	}

	fmt.Println(string(output))

	return nil
}

// Parses the Docker stats command returned data into a Stats struct
func StatsParser(data []byte) (*Stats, error) {

	var metrics Metrics
	parseErr := json.Unmarshal(data, &metrics)

	if parseErr != nil {
		return nil, errors.New(fmt.Sprintf("In StatsParser: Failed to parse JSON data -> %s", parseErr))
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

	return stats, nil
}

// Parses the app's config to a Config struct
func ConfigParser(data []byte) (error, *Config) {
	var config Config

	if err := yaml.Unmarshal(data, &config); err != nil {
		return errors.New(fmt.Sprintf("In ConfigParser: Failed to parse config file -> %s", err)), nil
	}

	return nil, &config
}

// Pretty prints YAML
func YAMLPrettyPrint(v any) error {
	output, errParse := yaml.Marshal(v)

	if errParse != nil {
		return errors.New(fmt.Sprintf("In YAMLPrettyPrint: Failed to parse YAML -> %s", errParse))
	}

	fmt.Println(string(output))

	return nil
}
