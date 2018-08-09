package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/kevinburke/gobike"
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
	m := make(map[Day]*dayData, 0)
	var d Day
	earliest := Day{Year: 3000, Month: time.January, Day: 0}
	bikeIDs := make(map[int64]bool, 0)
	for i := 0; i < len(trips); i++ {
		inSF := gobike.InSF(trips[i].StartStationLatitude, trips[i].StartStationLongitude)
		if !inSF {
			continue
		}
		bikeIDs[trips[i].BikeID] = true
		startTime := trips[i].StartTime
		d = Day{Day: startTime.Day(), Month: startTime.Month(), Year: startTime.Year()}
		_, ok := m[d]
		if ok {
			m[d].Trips++
		} else {
			m[d] = &dayData{Bikes: make(map[int64]bool, 0), Trips: 1}
		}
		m[d].Bikes[trips[i].BikeID] = true
		if d.Before(earliest) {
			earliest = d
		}
	}
	start := time.Date(earliest.Year, earliest.Month, earliest.Day, 0, 0, 0, 0, loc)
	for {
		d := Day{start.Year(), start.Month(), start.Day()}
		data := m[d]
		if data == nil {
			break
		}
		fmt.Printf("%s: %d trips, %d bikes\n", start.Format("January 2, 2006 (Monday)"), data.Trips, len(data.Bikes))
		start = start.Add(24 * time.Hour)
	}
}
