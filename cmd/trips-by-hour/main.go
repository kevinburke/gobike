package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/kevinburke/gobike"
)

func main() {
	flag.Parse()
	trips, err := gobike.LoadDir(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}
	buckets := make(map[int]int, 24)
	for i := range trips {
		buckets[trips[i].StartTime.Hour()]++
	}
	for i := 0; i < 24; i++ {
		fmt.Println("hour:", i, "trips:", buckets[i])
	}
}
