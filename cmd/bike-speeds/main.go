package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/kevinburke/gobike"
	tss "github.com/kevinburke/tss/lib"
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

	w := tss.NewWriter(os.Stdout, time.Time{})
	trips, err := gobike.LoadDir(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}
	bikes := make(map[int64][]float64)
	for i := range trips {
		_, ok := bikes[trips[i].BikeID]
		if ok {
			bikes[trips[i].BikeID] = append(bikes[trips[i].BikeID], trips[i].AvgSpeed())
		} else {
			bikes[trips[i].BikeID] = make([]float64, 1)
			bikes[trips[i].BikeID][0] = trips[i].AvgSpeed()
		}
	}
	i := 0
	for k := range bikes {
		if len(bikes[k]) < 10 {
			continue
		}
		fmt.Fprintln(w, k, len(bikes[k]), average(bikes[k]))
		i++
		if i > 100 {
			break
		}
	}
}
