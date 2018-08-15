package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"

	"github.com/kevinburke/gobike"
	"github.com/kevinburke/gobike/geo"
	"github.com/kevinburke/gobike/stats"
)

type homepageData struct {
	TripsPerWeek             template.JS
	TripsPerWeekCount        int64
	StationsPerWeek          template.JS
	StationsPerWeekCount     int64
	BikesPerWeek             template.JS
	BikesPerWeekCount        int64
	TripsPerBikePerWeek      template.JS
	TripsPerBikePerWeekCount string
	BS4ATripsPerWeek         template.JS
	BS4ATripsPerWeekCount    int64
	MostPopularStations      []*stats.StationCount
	Area                     string
}

func renderCity(name string, city *geo.City, tpl *template.Template, allTrips []*gobike.Trip) error {
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

	stationsPerWeek := stats.UniqueStationsPerWeek(trips)
	stationData, err := json.Marshal(stationsPerWeek)
	if err != nil {
		return err
	}
	mostPopularStations := stats.PopularStationsLast7Days(trips, 10)
	tripsPerWeek := stats.TripsPerWeek(trips)
	data, err := json.Marshal(tripsPerWeek)
	if err != nil {
		return err
	}
	bikeTripsPerWeek := stats.UniqueBikesPerWeek(trips)
	bikeData, err := json.Marshal(bikeTripsPerWeek)
	if err != nil {
		return err
	}
	tripsPerBikePerWeek := stats.TripsPerBikePerWeek(trips)
	tripPerBikeData, err := json.Marshal(tripsPerBikePerWeek)
	if err != nil {
		return err
	}
	bs4aTripsPerWeek := stats.BikeShareForAllTripsPerWeek(trips)
	bs4aData, err := json.Marshal(bs4aTripsPerWeek)
	if err != nil {
		return err
	}

	dir := filepath.Join("docs", name)
	if city == nil {
		dir = filepath.Join("docs")
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	w, err := os.Create(filepath.Join(dir, "index.html"))
	if err != nil {
		return err
	}

	if err := tpl.ExecuteTemplate(w, "city.html", &homepageData{
		Area:                     name,
		TripsPerWeek:             template.JS(string(data)),
		TripsPerWeekCount:        int64(tripsPerWeek[len(tripsPerWeek)-1].Data),
		StationsPerWeek:          template.JS(string(stationData)),
		StationsPerWeekCount:     int64(stationsPerWeek[len(stationsPerWeek)-1].Data),
		BikesPerWeek:             template.JS(string(bikeData)),
		BikesPerWeekCount:        int64(bikeTripsPerWeek[len(bikeTripsPerWeek)-1].Data),
		TripsPerBikePerWeek:      template.JS(string(tripPerBikeData)),
		TripsPerBikePerWeekCount: fmt.Sprintf("%.1f", tripsPerBikePerWeek[len(tripsPerBikePerWeek)-1].Data),
		MostPopularStations:      mostPopularStations,
		BS4ATripsPerWeek:         template.JS(string(bs4aData)),
		BS4ATripsPerWeekCount:    int64(bs4aTripsPerWeek[len(bs4aTripsPerWeek)-1].Data),
	}); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
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

	for name, city := range cities {
		if err := renderCity(name, city, homepageTpl, trips); err != nil {
			log.Fatalf("error building city %s: %s", name, err)
		}
	}
}
