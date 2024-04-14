package types

type Metrics struct {
	MemStats struct {
		Stats struct {
			Cache float64 `json:"cache"`
		}
		Usage float64 `json:"usage"`
		Limit float64 `json:"limit"`
	} `json:"memory_stats"`

	CPUStats struct {
		CPUUsage struct {
			TotalUsage float64 `json:"total_usage"`
		} `json:"cpu_usage"`
		SystemCPUUsage float64 `json:"system_cpu_usage"`
		NumberOfCPUs int16 `json:"online_cpus"`
	} `json:"cpu_stats"`

	PreCPUStats struct {
		CPUUsage struct {
			TotalUsage float64 `json:"total_usage"`
		} `json:"cpu_usage"`
		SystemCPUUsage float64 `json:"system_cpu_usage"`
	}
}

type Stats struct {
	UsedMemory float64
	AvailableMemory float64
	MemoryUsage string
	NumberOfCPUs int16
	CPUUsage string
}
