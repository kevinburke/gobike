package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"math"
	"os"

	"github.com/kevinburke/gobike"
)

func average(xs []float64) float64 {
	total := 0.0
	for _, v := range xs {
		total += v
	}
	return total / float64(len(xs))
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
	bikes := make(map[int64][]float64)
	for i := range trips {
		trip := trips[i]
		if trip.Distance() == 0 {
			continue
		}
		if trip.AvgSpeed() == 0 {
			continue
		}
		_, ok := bikes[trip.BikeID]
		if ok {
			bikes[trip.BikeID] = append(bikes[trip.BikeID], trip.AvgSpeed())
		} else {
			bikes[trip.BikeID] = make([]float64, 1)
			// TODO: weight longer trips more
			bikes[trip.BikeID][0] = trip.AvgSpeed()
		}
	}
	// 3.0, 3.5, 4.0
	var buckets [14]int
	for k := range bikes {
		if len(bikes[k]) < 5 {
			continue
		}
		avg := average(bikes[k])
		idx := int(math.Floor((avg - 3) * 2))
		if idx < 0 {
			fmt.Println(avg, bikes[k])
			continue
		}
		buckets[idx]++
	}
	for i := 0; i < len(buckets); i++ {
		fmt.Printf("%3g: %d\n", float64(i)*0.5+3, buckets[i])
	}
}
