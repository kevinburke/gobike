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
)

type Day struct {
	Year  int
	Month time.Month
	Day   int
}

func (d Day) Before(d2 Day) bool {
	if d.Year < d2.Year {
		return true
	}
	if d.Year > d2.Year {
		return false
	}
	if d.Month < d2.Month {
		return true
	}
	if d.Month > d2.Month {
		return false
	}
	return d.Day < d2.Day
}

type dayData struct {
	Bikes map[int64]bool
	Trips int
}

func main() {
	cityFlag := flag.String("city", "sf", "City to use (sf, oak/oakland)")
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		log.Fatal(err)
	}
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
	m := make(map[Day]*dayData)
	var d Day
	earliest := Day{Year: 3000, Month: time.January, Day: 0}
	bikeIDs := make(map[int64]bool)
	for i := 0; i < len(trips); i++ {
		inCity := city.ContainsPoint(trips[i].StartStationLatitude, trips[i].StartStationLongitude)
		if !inCity {
			continue
		}
		bikeIDs[trips[i].BikeID] = true
		startTime := trips[i].StartTime
		d = Day{Day: startTime.Day(), Month: startTime.Month(), Year: startTime.Year()}
		_, ok := m[d]
		if ok {
			m[d].Trips++
		} else {
			m[d] = &dayData{Bikes: make(map[int64]bool), Trips: 1}
		}
		m[d].Bikes[trips[i].BikeID] = true
		if d.Before(earliest) {
			earliest = d
		}
	}
	start := time.Date(earliest.Year, earliest.Month, earliest.Day, 0, 0, 0, 0, loc)
	thisyear := time.Now().Year()
	for {
		d := Day{start.Year(), start.Month(), start.Day()}
		data := m[d]
		if data == nil {
			break
		}
		var timeFmt string
		if thisyear == start.Year() {
			timeFmt = "January 2 (Monday)"
		} else {
			timeFmt = "January 2, 2006 (Monday)"
		}
		fmt.Printf("%s: %d trips, %d bikes\n", start.Format(timeFmt), data.Trips, len(data.Bikes))
		start = start.Add(24 * time.Hour)
	}
}
