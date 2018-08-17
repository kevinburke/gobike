package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/kevinburke/gobike"
	"github.com/kevinburke/gobike/geo"
	"github.com/kevinburke/gobike/stats"
	"golang.org/x/sync/errgroup"
)

type homepageData struct {
	TripsPerWeek             template.JS
	TripsPerWeekCount        int64
	UniqueTripsPerWeek       template.JS
	UniqueTripsPerWeekCount  int64
	StationsPerWeek          template.JS
	StationsPerWeekCount     int64
	BikesPerWeek             template.JS
	BikesPerWeekCount        int64
	TripsPerBikePerWeek      template.JS
	TripsPerBikePerWeekCount string
	BS4ATripsPerWeek         template.JS
	BS4ATripsPerWeekCount    int64
	BS4ATripPct              string
	PopularStations          []*stats.StationCount
	PopularBS4AStations      []*stats.StationCount
	TripsByDistrict          [11]int
	Area                     string
	ShareOfTotalTrips        string
	AverageWeekdayTrips      string
}

type stationData struct {
	Area     string
	Stations []*stats.StationCount
}

func renderCity(name string, city *geo.City, tpl, stationTpl *template.Template, allTrips []*gobike.Trip) error {
	trips := make([]*gobike.Trip, 0)
	if city == nil {
		trips = allTrips
	} else {
		for i := range allTrips {
			if city.ContainsPoint(allTrips[i].StartStationLatitude, allTrips[i].StartStationLongitude) {
				trips = append(trips, allTrips[i])
			}
		}
	}

	var group errgroup.Group
	var stationsPerWeek, tripsPerWeek, uniqueTripsPerWeek, bikeTripsPerWeek, tripsPerBikePerWeek, bs4aTripsPerWeek stats.TimeSeries
	var stationBytes, data, uniqueTripsData, bikeData, tripPerBikeData, bs4aData []byte
	var mostPopularStations, popularBS4AStations []*stats.StationCount
	var shareOfTotalTrips, averageWeekdayTrips string
	var tripsByDistrict [11]int
	if name == "sf" {
		group.Go(func() error {
			tripsByDistrict = stats.TripsLastWeekPerDistrict(trips)
			return nil
		})
	}

	group.Go(func() error {
		stationsPerWeek = stats.UniqueStationsPerWeek(trips)
		var err error
		stationBytes, err = json.Marshal(stationsPerWeek)
		return err
	})
	group.Go(func() error {
		mostPopularStations = stats.PopularStationsLast7Days(trips, 10)
		return nil
	})
	var allStations []*stats.StationCount
	group.Go(func() error {
		allStations = stats.PopularStationsLast7Days(trips, 50000)
		return nil
	})
	group.Go(func() error {
		popularBS4AStations = stats.PopularBS4AStationsLast7Days(trips, 10)
		return nil
	})
	group.Go(func() error {
		uniqueTripsPerWeek = stats.UniqueTripsPerWeek(trips)
		var err error
		uniqueTripsData, err = json.Marshal(uniqueTripsPerWeek)
		return err
	})
	group.Go(func() error {
		tripsPerWeek = stats.TripsPerWeek(trips)
		if name == "sf" {
			averageWeekdayTripsf64 := stats.AverageWeekdayTrips(trips)
			averageWeekdayTrips = fmt.Sprintf("%.1f", averageWeekdayTripsf64)
			shareOfTotalTrips = fmt.Sprintf("%.2f", 100*averageWeekdayTripsf64/(4.2*1000*1000))
		}
		var err error
		data, err = json.Marshal(tripsPerWeek)
		return err
	})
	group.Go(func() error {
		bikeTripsPerWeek = stats.UniqueBikesPerWeek(trips)
		var err error
		bikeData, err = json.Marshal(bikeTripsPerWeek)
		return err
	})
	group.Go(func() error {
		tripsPerBikePerWeek = stats.TripsPerBikePerWeek(trips)
		var err error
		tripPerBikeData, err = json.Marshal(tripsPerBikePerWeek)
		return err
	})
	group.Go(func() error {
		bs4aTripsPerWeek = stats.BikeShareForAllTripsPerWeek(trips)
		var err error
		bs4aData, err = json.Marshal(bs4aTripsPerWeek)
		return err
	})
	if err := group.Wait(); err != nil {
		return err
	}

	tripsPerWeekCountf64 := tripsPerWeek[len(tripsPerWeek)-1].Data
	uniqueTripsPerWeekCountf64 := uniqueTripsPerWeek[len(uniqueTripsPerWeek)-1].Data
	bs4aTripsPerWeekCountf64 := bs4aTripsPerWeek[len(bs4aTripsPerWeek)-1].Data

	hdata := &homepageData{
		Area:                     name,
		TripsPerWeek:             template.JS(string(data)),
		TripsPerWeekCount:        int64(tripsPerWeekCountf64),
		UniqueTripsPerWeek:       template.JS(string(uniqueTripsData)),
		UniqueTripsPerWeekCount:  int64(uniqueTripsPerWeekCountf64),
		StationsPerWeek:          template.JS(string(stationBytes)),
		StationsPerWeekCount:     int64(stationsPerWeek[len(stationsPerWeek)-1].Data),
		BikesPerWeek:             template.JS(string(bikeData)),
		BikesPerWeekCount:        int64(bikeTripsPerWeek[len(bikeTripsPerWeek)-1].Data),
		TripsPerBikePerWeek:      template.JS(string(tripPerBikeData)),
		TripsPerBikePerWeekCount: fmt.Sprintf("%.1f", tripsPerBikePerWeek[len(tripsPerBikePerWeek)-1].Data),
		PopularStations:          mostPopularStations,
		PopularBS4AStations:      popularBS4AStations,
		BS4ATripsPerWeek:         template.JS(string(bs4aData)),
		BS4ATripsPerWeekCount:    int64(bs4aTripsPerWeekCountf64),
		BS4ATripPct:              fmt.Sprintf("%.1f", 100*bs4aTripsPerWeekCountf64/tripsPerWeekCountf64),
		TripsByDistrict:          tripsByDistrict,
		ShareOfTotalTrips:        shareOfTotalTrips,
		AverageWeekdayTrips:      averageWeekdayTrips,
	}
	dir := filepath.Join("docs", name)
	if city == nil {
		dir = "docs"
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	buf := new(bytes.Buffer)
	if err := tpl.ExecuteTemplate(buf, "city.html", hdata); err != nil {
		return err
	}
	if err := ioutil.WriteFile(filepath.Join(dir, "index.html"), buf.Bytes(), 0644); err != nil {
		return err
	}
	if city == nil {
		dir = filepath.Join(dir, "bayarea")
	}
	stationDir := filepath.Join(dir, "stations")
	if err := os.MkdirAll(stationDir, 0755); err != nil {
		return err
	}
	buf.Reset()
	sdata := &stationData{
		Area:     name,
		Stations: allStations,
	}
	if err := stationTpl.ExecuteTemplate(buf, "stations.html", sdata); err != nil {
		return err
	}
	if err := ioutil.WriteFile(filepath.Join(stationDir, "index.html"), buf.Bytes(), 0644); err != nil {
		return err
	}
	return nil
}

func main() {
	flag.Parse()

	trips, err := gobike.LoadDir(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}
	if len(trips) == 0 {
		log.Fatalf("no trips")
	}

	cities := map[string]*geo.City{
		"bayarea":    nil,
		"berkeley":   geo.Berkeley,
		"emeryville": geo.Emeryville,
		"sf":         geo.SF,
		"oakland":    geo.Oakland,
		"sj":         geo.SanJose,
	}

	homepageTpl := template.Must(template.ParseFiles("templates/city.html"))
	stationTpl := template.Must(template.ParseFiles("templates/stations.html"))

	for name, city := range cities {
		if err := renderCity(name, city, homepageTpl, stationTpl, trips); err != nil {
			log.Fatalf("error building city %s: %s", name, err)
		}
	}
}
