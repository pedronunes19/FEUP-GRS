package main

import (
	"context"
	"log"
	"os"
	"sync"

	. "grs/common/types"
	. "grs/common/utils"
	metric_collector "grs/metric-collector"
	scaler "grs/scaler"
)

const CONFIG_FILE string = "config.yaml"

// Runs the application. One Go routine runs the metric collector and other runs the auto scaler 
func main() {
	file, err := os.ReadFile(CONFIG_FILE)

	if err != nil {
		log.Fatalln("Main: Failed to read config file")
	}

	err, config := ConfigParser(file)

	if err != nil {
		log.Fatalln("Main: Failed to parse config file")
	}

	YAMLPrettyPrint(config)

	var s sync.WaitGroup
	s.Add(2)

	c := make(chan []*Stats)

	ctx := context.Background()

	go metric_collector.Run(&s, c, &ctx)
	go scaler.Run(&s, config, c, &ctx)

	s.Wait()
}
