package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"time"

	"github.com/kevinburke/gobike"
)

func average(xs []*tripData) float64 {
	totalDistance := 0.0
	totalTime := 0.0
	for _, v := range xs {
		totalDistance += v.Distance
		totalTime += v.Time
	}
	return totalDistance / totalTime
}

type tripData struct {
	Distance float64
	Time     float64
}

func (t *tripData) String() string {
	return fmt.Sprintf("%f: %s", t.Distance, time.Duration(t.Time*float64(time.Hour)))
}

func main() {
	flag.Parse()

	//w := tss.NewWriter(os.Stdout, time.Time{})
	f, err := os.Open(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	trips, err := gobike.Load(bufio.NewReader(f))
	if err != nil {
		log.Fatal(err)
	}
	bikes := make(map[int64][]*tripData)
	for i := range trips {
		trip := trips[i]
		if trip.Distance() == 0 {
			continue
		}
		if trip.AvgSpeed() == 0 {
			continue
		}
		data := &tripData{
			Time:     float64(trip.EndTime.Sub(trip.StartTime)) / float64(time.Hour),
			Distance: trip.Distance(),
		}
		_, ok := bikes[trip.BikeID]
		if ok {
			bikes[trip.BikeID] = append(bikes[trip.BikeID], data)
		} else {
			bikes[trip.BikeID] = make([]*tripData, 1)
			bikes[trip.BikeID][0] = data
		}
	}
	// 3.0, 3.5, 4.0
	var buckets [16]int
	start := 2.0
	for k := range bikes {
		if len(bikes[k]) < 5 {
			continue
		}
		avg := average(bikes[k])
		idx := int(math.Floor((avg - start) * 2))
		if idx < 0 {
			fmt.Println(avg, bikes[k])
			continue
		}
		buckets[idx]++
	}
	for i := 0; i < len(buckets); i++ {
		fmt.Printf("%3g: %d\n", float64(i)*0.5+start, buckets[i])
	}
}
