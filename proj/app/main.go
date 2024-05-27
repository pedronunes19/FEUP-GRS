package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	. "grs/common/types"
	. "grs/common/utils"
	metric_collector "grs/metric-collector"
	scaler "grs/scaler"

	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/elastic/go-elasticsearch/v8"
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

	periodStr := config.Period[:len(config.Period)-1]
	
	period, err := strconv.Atoi(periodStr)

	if err != nil {
		log.Fatalln("Error converting period string to integer")
		return
	}

	interval := time.Duration(period) * time.Second

	for {
		var s sync.WaitGroup
		s.Add(2)

		c := make(chan []*Stats)
		ctx := context.Background()

		go metric_collector.Run(&s, c, &ctx)

		stats := <-c
		close(c)

		go scaler.Run(&s, config, stats, &ctx)

		s.Wait()

		for _, stat := range stats {
			log.Println(stat)
			// Convert stat to JSON
			output, errParse := json.Marshal(stat)
			if errParse != nil {
				log.Fatalln("Failed to marshal data:", errParse)
			}

			// Unmarshal JSON to map
			var data map[string]interface{}
			if err := json.Unmarshal(output, &data); err != nil {
				log.Fatalln("Failed to unmarshal JSON:", err)
			}

			// Add timestamp
			data["timestamp"] = time.Now().Format(time.RFC3339)
			data["CPUUsage"], _ = strconv.ParseFloat(stat.CPUUsage[:len(stat.CPUUsage) - 1], 32)
			data["MemoryUsage"], _ = strconv.ParseFloat(stat.MemoryUsage[:len(stat.MemoryUsage) - 1], 32)

			// Marshal back to JSON
			updatedOutput, err := json.MarshalIndent(data, "", "  ")
			
			if err != nil {
				log.Fatalln("Failed to marshal updated data:", err)
			}

			es, err := elasticsearch.NewDefaultClient()
			if err != nil {
				log.Fatalf("Error creating the Elastic client: %s", err)
			}


			req := esapi.IndexRequest{
				Index:   "containers",
				Body:    strings.NewReader(string(updatedOutput)),
				Refresh: "true",
			}
		
			res, err := req.Do(context.Background(), es)
			if err != nil {
				log.Fatalf("Error getting response: %s", err)
			}
		
			if res.IsError() {
				log.Printf("[%s] Error indexing document", res.Status())
			} else {
				log.Printf("[%s] Document indexed.", res.Status())
			}

			res.Body.Close()

		}

		time.Sleep(interval)
	}
}
