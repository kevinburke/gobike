package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/kevinburke/gobike"
	"github.com/kevinburke/gobike/geo"
	"github.com/kevinburke/gobike/stats"
)

func main() {
	cityFlag := flag.String("city", "sf", "City to use (sf, oak/oakland)")
	statsFlag := flag.String("stat", "trips-per-week", "Stats to show")

	flag.Parse()
	trips := make([]*gobike.Trip, 0)
	for i := 0; i < flag.NArg(); i++ {
		f, err := os.Open(flag.Arg(i))
		if err != nil {
			log.Fatal(err)
		}
		fileTrips, err := gobike.Load(bufio.NewReader(f))
		if err != nil {
			log.Fatal(err)
		}
		trips = append(trips, fileTrips...)
	}
	var city *geo.City
	switch *cityFlag {
	case "sf":
		city = geo.SF
	case "oak", "oakland":
		city = geo.Oakland
	case "sj", "sjc", "sanjose", "san jose":
		city = geo.SanJose
	default:
		log.Fatalf("unknown city %q", *cityFlag)
	}

	cityTrips := []*gobike.Trip{}

	for i := 0; i < len(trips); i++ {
		inCity := city.ContainsPoint(
			trips[i].StartStationLatitude,
			trips[i].StartStationLongitude,
		)
		if inCity {
			cityTrips = append(cityTrips, trips[i])
		}
	}

	var data stats.TimeSeries
	switch *statsFlag {
	case "trips-per-week":
		data = stats.TripsPerWeek(cityTrips)
	default:
		log.Fatalf("unknown stat %s", *statsFlag)
	}

	thisyear := time.Now().Year()
	for _, stat := range data {
		var timeFmt string
		if thisyear == stat.Date.Year() {
			timeFmt = "January 2 (Monday)"
		} else {
			timeFmt = "January 2, 2006 (Monday)"
		}
		fmt.Printf("%s: %f,\n", stat.Date.Format(timeFmt), stat.Data)
	}
}
