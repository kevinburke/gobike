package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"time"

	"github.com/kevinburke/gobike"
	"github.com/kevinburke/gobike/client"
	"github.com/kevinburke/gobike/geo"
	"github.com/kevinburke/gobike/stats"
	"github.com/kevinburke/semaphore"
	"golang.org/x/sync/errgroup"
)

type homepageData struct {
	Area         string
	FriendlyName string

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
	BS4ATripPct              string

	EmptyStations, FullStations template.JS

	PopularStations     []*stats.StationCount
	PopularBS4AStations []*stats.StationCount
	TripsByDistrict     [11]int
	Population          int
	ShareOfTotalTrips   string
	AverageWeekdayTrips string
	EstimatedTotalTrips string

	DistanceBuckets *Histogram
	DurationBuckets *Histogram
}

type stationData struct {
	Area     string
	Stations []*stats.StationCount
}

func empty(city *geo.City, stationMap map[string]*gobike.Station) func(ss *gobike.StationStatus) bool {
	return func(ss *gobike.StationStatus) bool {
		station := stationMap[ss.ID]
		// yuck pointer comparison
		if city != nil && station.City != city {
			return false
		}
		if !ss.IsInstalled || !ss.IsRenting {
			return false
		}
		return ss.NumBikesAvailable == 0
	}
}

func full(city *geo.City, stationMap map[string]*gobike.Station) func(ss *gobike.StationStatus) bool {
	return func(ss *gobike.StationStatus) bool {
		station := stationMap[ss.ID]
		// yuck pointer comparison
		if city != nil && station.City != city {
			return false
		}
		if !ss.IsInstalled || !ss.IsRenting {
			return false
		}
		return ss.NumDocksAvailable == 0
	}
}

func renderCity(name string, city *geo.City, tpl, stationTpl *template.Template, stationMap map[string]*gobike.Station, allTrips []*gobike.Trip, statuses map[string][]*gobike.StationStatus) error {
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	group, errctx := errgroup.WithContext(ctx)
	var stationsPerWeek, tripsPerWeek, bikeTripsPerWeek, tripsPerBikePerWeek, bs4aTripsPerWeek, emptyStations, fullStations stats.TimeSeries
	var stationBytes, data, bikeData, tripPerBikeData, bs4aData, emptyStationData, fullStationData []byte
	var mostPopularStations, popularBS4AStations []*stats.StationCount
	var shareOfTotalTrips, averageWeekdayTrips, estimatedTotalTrips string
	var tripsByDistrict [11]int
	if name == "sf" {
		group.Go(func() error {
			tripsByDistrict = stats.TripsLastWeekPerDistrict(trips)
			return nil
		})
	}
	sem := semaphore.New(runtime.NumCPU() * 2)

	tz, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		panic(err)
	}
	now := time.Now().In(tz)
	group.Go(func() error {
		sem.AcquireContext(errctx)
		defer sem.Release()
		emptyStations = stats.StatusFilterOverTime(statuses, empty(city, stationMap), now.Add(-7*24*time.Hour), now, 15*time.Minute)
		var err error
		emptyStationData, err = json.Marshal(emptyStations)
		return err
	})
	group.Go(func() error {
		sem.AcquireContext(errctx)
		defer sem.Release()
		fullStations = stats.StatusFilterOverTime(statuses, full(city, stationMap), now.Add(-7*24*time.Hour), now, 15*time.Minute)
		var err error
		fullStationData, err = json.Marshal(fullStations)
		return err
	})

	group.Go(func() error {
		sem.AcquireContext(errctx)
		defer sem.Release()
		stationsPerWeek = stats.UniqueStationsPerWeek(trips)
		var err error
		stationBytes, err = json.Marshal(stationsPerWeek)
		return err
	})
	group.Go(func() error {
		sem.AcquireContext(errctx)
		defer sem.Release()
		mostPopularStations = stats.PopularStationsLast7Days(stationMap, trips, 10)
		return nil
	})
	var allStations []*stats.StationCount
	group.Go(func() error {
		sem.AcquireContext(errctx)
		defer sem.Release()
		allStations = stats.PopularStationsLast7Days(stationMap, trips, 50000)
		return nil
	})
	group.Go(func() error {
		sem.AcquireContext(errctx)
		defer sem.Release()
		popularBS4AStations = stats.PopularBS4AStationsLast7Days(stationMap, trips, 10)
		return nil
	})
	group.Go(func() error {
		sem.AcquireContext(errctx)
		defer sem.Release()
		tripsPerWeek = stats.TripsPerWeek(trips)
		averageWeekdayTripsf64 := stats.AverageWeekdayTrips(trips)
		averageWeekdayTrips = fmt.Sprintf("%.1f", averageWeekdayTripsf64)
		estimatedTotalTripsf64 := 4.82*float64(geo.Populations[name]) - math.Mod(4.82*float64(geo.Populations[name]), 1000)
		shareOfTotalTrips = fmt.Sprintf("%.2f", 100*averageWeekdayTripsf64/estimatedTotalTripsf64)
		estimatedTotalTrips = strconv.FormatFloat(estimatedTotalTripsf64, 'f', 0, 64)
		var err error
		data, err = json.Marshal(tripsPerWeek)
		return err
	})
	group.Go(func() error {
		sem.AcquireContext(errctx)
		defer sem.Release()
		bikeTripsPerWeek = stats.UniqueBikesPerWeek(trips)
		var err error
		bikeData, err = json.Marshal(bikeTripsPerWeek)
		return err
	})
	group.Go(func() error {
		sem.AcquireContext(errctx)
		defer sem.Release()
		tripsPerBikePerWeek = stats.TripsPerBikePerWeek(trips)
		var err error
		tripPerBikeData, err = json.Marshal(tripsPerBikePerWeek)
		return err
	})
	group.Go(func() error {
		sem.AcquireContext(errctx)
		defer sem.Release()
		bs4aTripsPerWeek = stats.BikeShareForAllTripsPerWeek(trips)
		var err error
		bs4aData, err = json.Marshal(bs4aTripsPerWeek)
		return err
	})
	var distanceBuckets, durationBuckets *Histogram
	group.Go(func() error {
		sem.AcquireContext(errctx)
		defer sem.Release()
		distanceBucketsArr, avg := stats.DistanceBucketsLastWeek(trips, 0.50, 6)
		distanceBuckets = &Histogram{
			interval: 0.50,
			unit:     "mi",
			average:  avg,
			Buckets:  distanceBucketsArr,
		}
		return nil
	})
	group.Go(func() error {
		sem.AcquireContext(errctx)
		defer sem.Release()
		durationBucketsArr, avg := stats.DurationBucketsLastWeek(trips, 5*time.Minute, 8)
		durationBuckets = &Histogram{
			interval:   float64(5 * time.Minute),
			isDuration: true,
			unit:       "min",
			average:    avg,
			Buckets:    durationBucketsArr,
		}
		return nil
	})
	if err := group.Wait(); err != nil {
		return err
	}

	tripsPerWeekCountf64 := tripsPerWeek[len(tripsPerWeek)-1].Data
	bs4aTripsPerWeekCountf64 := bs4aTripsPerWeek[len(bs4aTripsPerWeek)-1].Data

	var friendlyName string
	if city != nil {
		friendlyName = city.Name
	}
	hdata := &homepageData{
		Area:                     name,
		FriendlyName:             friendlyName,
		TripsPerWeek:             template.JS(string(data)),
		TripsPerWeekCount:        int64(tripsPerWeekCountf64),
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
		DistanceBuckets:          distanceBuckets,
		DurationBuckets:          durationBuckets,
		Population:               geo.Populations[name],
		EstimatedTotalTrips:      estimatedTotalTrips,

		EmptyStations: template.JS(string(emptyStationData)),
		FullStations:  template.JS(string(fullStationData)),
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

type Histogram struct {
	interval   float64
	isDuration bool
	unit       string
	average    float64
	Buckets    []int
}

func (h *Histogram) Average() string {
	return strconv.FormatFloat(h.average, 'f', 1, 64)
}

func (b *Histogram) Distrange(i int) string {
	interval := b.interval
	if b.isDuration {
		switch b.unit {
		case "min":
			interval = b.interval / float64(time.Minute)
		default:
			panic("unknown distrange unit " + b.unit)
		}
	}
	s1 := strconv.FormatFloat(float64(i)*interval, 'f', -1, 64)
	s2 := strconv.FormatFloat(float64(i+1)*interval, 'f', -1, 64)
	if i == 0 {
		s1 = "0"
	}
	if i+1 == len(b.Buckets) {
		s2 = b.unit + " and up"
	} else {
		s1 = s1 + "-"
		s2 = s2 + b.unit
	}
	return s1 + s2
}

func (b *Histogram) Percent(i int) string {
	sum := 0
	for i := 0; i < len(b.Buckets); i++ {
		sum += b.Buckets[i]
	}
	return fmt.Sprintf("%.1f%%", 100*float64(b.Buckets[i])/float64(sum))
}

func main() {
	flag.Parse()

	group := errgroup.Group{}
	var trips []*gobike.Trip
	var statuses []*gobike.StationStatus
	group.Go(func() error {
		var err error
		trips, err = gobike.LoadDir(flag.Arg(0))
		return err
	})
	group.Go(func() error {
		var err error
		statuses, err = gobike.LoadCapacityDir(flag.Arg(1))
		return err
	})
	if err := group.Wait(); err != nil {
		log.Fatal(err)
	}
	if len(trips) == 0 {
		log.Fatalf("no trips")
	}
	byStation := make(map[string][]*gobike.StationStatus)
	for i := range statuses {
		ss := statuses[i]
		if _, ok := byStation[ss.ID]; !ok {
			byStation[ss.ID] = make([]*gobike.StationStatus, 0)
		}
		byStation[ss.ID] = append(byStation[ss.ID], ss)
	}

	for id := range byStation {
		sort.Slice(byStation[id], func(i, j int) bool {
			return byStation[id][i].LastReported.Before(byStation[id][j].LastReported)
		})
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

	c := client.NewClient()
	c.Stations.CacheTTL = 24 * 14 * time.Hour
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := c.Stations.All(ctx)
	if err != nil {
		log.Fatal(err)
	}
	stationMap := make(map[string]*gobike.Station, len(resp.Stations))
	for i := range resp.Stations {
		for _, city := range cities {
			if city != nil && city.ContainsPoint(resp.Stations[i].Latitude, resp.Stations[i].Longitude) {
				resp.Stations[i].City = city
			}
		}
		stationMap[strconv.Itoa(resp.Stations[i].ID)] = resp.Stations[i]
	}
	for name, city := range cities {
		if err := renderCity(name, city, homepageTpl, stationTpl, stationMap, trips, byStation); err != nil {
			log.Fatalf("error building city %s: %s", name, err)
		}
	}
}
