package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/kevinburke/gobike"
	"github.com/kevinburke/gobike/client"
	"github.com/kevinburke/gobike/geo"
	"github.com/kevinburke/gobike/stats"
	tss "github.com/kevinburke/tss/lib"
	"golang.org/x/sync/errgroup"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
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

	MovesPerWeek      template.JS
	MovesPerWeekCount int64

	EmptyStations, FullStations template.JS

	PopularStations     []*stats.StationCount
	PopularBS4AStations []*stats.StationCount
	TripsByDistrict     [11]int
	Population          int
	ShareOfTotalTrips   string
	AverageWeekdayTrips string
	EstimatedTotalTrips string

	RunRate       template.JS
	LatestRunRate string

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
		if station == nil {
			return true
			panic(fmt.Sprintln("nil station", ss.ID, ss.NumDocksAvailable, ss.IsRenting))
		}
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
		if station == nil {
			return false
			panic(fmt.Sprintln("nil station", ss.ID, ss.NumDocksAvailable, ss.IsRenting))
		}
		if city != nil && station.City != city {
			return false
		}
		if !ss.IsInstalled || !ss.IsRenting {
			return false
		}
		return ss.NumDocksAvailable == 0
	}
}

const stationCapacityInterval = 20 * time.Minute

var printer *message.Printer

func renderCity(w io.Writer, name string, city *geo.City, tpl, stationTpl *template.Template, stationMap map[string]*gobike.Station, trips []*gobike.Trip, statuses map[string][]*gobike.StationStatus) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	group, errctx := errgroup.WithContext(ctx)
	_ = errctx
	var stationsPerWeek, tripsPerWeek, bikeTripsPerWeek, tripsPerBikePerWeek, bs4aTripsPerWeek, emptyStations, fullStations, runRate, moves stats.TimeSeries
	var stationBytes, data, bikeData, tripPerBikeData, bs4aData, emptyStationData, fullStationData, runRateData, moveData []byte
	var mostPopularStations, popularBS4AStations []*stats.StationCount
	var shareOfTotalTrips, averageWeekdayTrips, estimatedTotalTrips string
	var tripsByDistrict [11]int
	if name == "sf" {
		group.Go(func() error {
			tripsByDistrict = stats.TripsLastWeekPerDistrict(trips)
			return nil
		})
	}
	tz, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		panic(err)
	}
	now := time.Now().In(tz)
	nowRounded := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute()-now.Minute()%20, 0, 0, tz)
	fmt.Fprintln(w, "collecting stats")
	group.Go(func() error {
		emptyStations = stats.StatusFilterOverTime(statuses, empty(city, stationMap), nowRounded.Add(-3*24*time.Hour), nowRounded, stationCapacityInterval)
		var err error
		emptyStationData, err = json.Marshal(emptyStations)
		return err
	})
	group.Go(func() error {
		moves = stats.MovesPerWeek(trips)
		var err error
		moveData, err = json.Marshal(moves)
		return err
	})
	group.Go(func() error {
		fullStations = stats.StatusFilterOverTime(statuses, full(city, stationMap), nowRounded.Add(-3*24*time.Hour), nowRounded, stationCapacityInterval)
		var err error
		fullStationData, err = json.Marshal(fullStations)
		return err
	})

	group.Go(func() error {
		stationsPerWeek = stats.UniqueStationsPerWeek(trips)
		var err error
		stationBytes, err = json.Marshal(stationsPerWeek)
		return err
	})
	group.Go(func() error {
		mostPopularStations = stats.PopularStationsLast7Days(stationMap, trips, statuses, 10)
		return nil
	})
	group.Go(func() error {
		runRate = stats.Revenue(trips)
		var err error
		runRateData, err = json.Marshal(runRate)
		return err
	})
	var allStations []*stats.StationCount
	group.Go(func() error {
		allStations = stats.PopularStationsLast7Days(stationMap, trips, statuses, 50000)
		for i := 0; i < len(allStations); i++ {
			if allStations[i].Station.ID == 372 {
				fmt.Println(allStations[i].Station.Name)
			}
		}
		return nil
	})
	group.Go(func() error {
		popularBS4AStations = stats.PopularBS4AStationsLast7Days(stationMap, trips, 10)
		return nil
	})
	group.Go(func() error {
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
	var distanceBuckets, durationBuckets *Histogram
	group.Go(func() error {
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
		Area:         name,
		FriendlyName: friendlyName,

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

		MovesPerWeek:      template.JS(moveData),
		MovesPerWeekCount: int64(moves[len(moves)-1].Data),

		TripsByDistrict:     tripsByDistrict,
		ShareOfTotalTrips:   shareOfTotalTrips,
		AverageWeekdayTrips: averageWeekdayTrips,
		DistanceBuckets:     distanceBuckets,
		DurationBuckets:     durationBuckets,
		Population:          geo.Populations[name],
		EstimatedTotalTrips: estimatedTotalTrips,

		EmptyStations: template.JS(string(emptyStationData)),
		FullStations:  template.JS(string(fullStationData)),

		RunRate:       template.JS(string(runRateData)),
		LatestRunRate: printer.Sprintf("%.0f", runRate[len(runRate)-1].Data/100),
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

var cities = map[string]*geo.City{
	"bayarea":    nil,
	"berkeley":   geo.Berkeley,
	"emeryville": geo.Emeryville,
	"sf":         geo.SF,
	"oakland":    geo.Oakland,
	"sj":         geo.SanJose,
}

func main() {
	// check out loadStationsFromDisk
	// localStationFile := flag.String("local-station-file", "", "Use local station file instead of retrieving stations over HTTP")
	flag.Parse()

	w := tss.NewWriter(os.Stdout, time.Time{})
	printer = message.NewPrinter(language.English)
	fmt.Fprintf(w, "get stations\n")
	var stations []*gobike.Station
	c := client.NewClient()
	c.Stations.CacheTTL = 24 * 14 * time.Hour
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := c.Stations.All(ctx)
	if err != nil {
		log.Fatal(err)
	}
	stations = resp.Stations
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
	fmt.Fprintf(w, "loaded data\n")
	if len(trips) == 0 {
		log.Fatalf("no trips")
	}
	byStation := stats.StatusMap(statuses)
	homepageTpl := template.Must(template.ParseFiles("templates/city.html"))
	stationTpl := template.Must(template.ParseFiles("templates/stations.html"))

	stationMap := gobike.StationMap(stations)
	tripsPerCity := make(map[string][]*gobike.Trip)
	tripsPerCity["bayarea"] = trips
	unknownStations := make(map[string]string)
	for i := range trips {
		station, ok := stationMap[trips[i].StartStationID]
		var slug string
		if ok {
			slug = station.City.Slug
		} else {
			slug, ok = unknownStations[trips[i].StartStationID]
			if !ok {
				// geocode and put in stations
				for citySlug, city := range cities {
					if city != nil && city.ContainsPoint(trips[i].StartStationLatitude, trips[i].StartStationLongitude) {
						unknownStations[trips[i].StartStationID] = citySlug
						slug = citySlug
					}
				}
			}
		}
		if tripsPerCity[slug] == nil {
			tripsPerCity[slug] = make([]*gobike.Trip, 0, 1000)
		}
		tripsPerCity[slug] = append(tripsPerCity[slug], trips[i])
	}
	for slug, city := range cities {
		fmt.Fprintf(w, "render %s\n", slug)
		if err := renderCity(w, slug, city, homepageTpl, stationTpl, stationMap, tripsPerCity[slug], byStation); err != nil {
			log.Fatalf("error building city %s: %s", slug, err)
		}
	}
}
