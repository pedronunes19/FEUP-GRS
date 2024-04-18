package main

import (
	"fmt"
	"log"
	"os"
	"sync"

	. "grs/common/utils"
	metric_collector "grs/metric-collector"
	scaler "grs/scaler"
)

func main() {
	fmt.Println("Starting program...")

	file, err := os.ReadFile("config.yaml")

	if err != nil {
		log.Fatalln("Failed to read config file")
	}

	err, config := ConfigParser(file)

	if err != nil {
		log.Fatalln("Failed to parse config file")
	}

	YAMLPrettyPrint(config)

	var s sync.WaitGroup
	s.Add(2)

	go metric_collector.Run(&s)
	go scaler.Run(&s)

	s.Wait()
}
