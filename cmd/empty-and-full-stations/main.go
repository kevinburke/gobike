package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/kevinburke/gobike"
	"github.com/kevinburke/gobike/client"
)

func main() {
	start := time.Now()
	flag.Parse()
	f, err := os.Open(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	byStation := make(map[string][]*gobike.StationStatus)
	err = gobike.ForeachStationStatus(bufio.NewReader(f), func(ss *gobike.StationStatus) error {
		if _, ok := byStation[ss.ID]; !ok {
			byStation[ss.ID] = make([]*gobike.StationStatus, 0)
		}
		byStation[ss.ID] = append(byStation[ss.ID], ss)
		return nil
	})
	empty := make(map[string]time.Duration, len(byStation))
	full := make(map[string]time.Duration, len(byStation))
	for id := range byStation {
		for j := 0; j < len(byStation[id]); j++ {
			measurements := byStation[id]
			status := measurements[j]
			if status.NumBikesAvailable == 0 {
				if j < len(measurements)-1 {
					dur := measurements[j+1].LastReported.Sub(status.LastReported)
					empty[id] = empty[id] + dur
				}
			}
			if status.NumDocksAvailable == 0 {
				if j < len(measurements)-1 {
					dur := measurements[j+1].LastReported.Sub(status.LastReported)
					full[id] = full[id] + dur
				}
			}
		}
	}
	if err != nil {
		log.Fatal(err)
	}
	c := client.NewClient()
	response, err := c.Stations.All(context.TODO())
	if err != nil {
		log.Fatal(err)
	}
	stationNames := make(map[string]*gobike.Station, len(response.Stations))
	for i := range response.Stations {
		stationNames[strconv.Itoa(response.Stations[i].ID)] = response.Stations[i]
	}

	for i := range empty {
		fmt.Printf("%q: %s empty\n", stationNames[i].Name, empty[i].Round(time.Minute))
	}
	for i := range full {
		fmt.Printf("%q: %s full\n", stationNames[i].Name, full[i].Round(time.Minute))
	}
	fmt.Println(time.Since(start))
}
