package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/kevinburke/gobike"
	"github.com/kevinburke/gobike/geo"
)

func run() error {
	flag.Parse()

	trips, err := gobike.LoadDir(flag.Arg(0))
	if err != nil {
		return err
	}
	sort.Slice(trips, func(i, j int) bool {
		return trips[i].StartTime.Before(trips[j].StartTime)
	})

	fw, err := os.Create(flag.Arg(1))
	if err != nil {
		return err
	}
	defer fw.Close()

	w := csv.NewWriter(fw)

	geolocate := func(lat, long float64) string {
		if geo.SF.ContainsPoint(lat, long) {
			return "San Francisco"
		}
		if geo.Oakland.ContainsPoint(lat, long) {
			return "Oakland"
		}
		if geo.SanJose.ContainsPoint(lat, long) {
			return "San Jose"
		}
		return ""
	}

	w.Write([]string{
		"duration",
		"start_time",
		"end_time",
		"start_station_id",
		"start_station_name",
		"start_station_latitude",
		"start_station_longitude",
		"start_station_city",
		"end_station_id",
		"end_station_name",
		"end_station_latitude",
		"end_station_longitude",
		"end_station_city",
		"bike_id",
		"user_type",
		"member_birth_year",
		"member_gender",
		"bike_share_for_all",
	})

	for i, trip := range trips {
		if trip.EndTime.Before(trip.StartTime) {
			// WTF?
			continue
		}

		bikeshare := "false"
		if trip.BikeShareForAllTrip {
			bikeshare = "true"
		}

		record := []string{
			strconv.Itoa(int(trip.Duration.Seconds())),
			trip.StartTime.Format(time.RFC3339),
			trip.EndTime.Format(time.RFC3339),

			// Start station
			strconv.Itoa(trip.StartStationID),
			trip.StartStationName,
			fmt.Sprint(trip.StartStationLatitude),
			fmt.Sprint(trip.StartStationLongitude),
			geolocate(trip.StartStationLatitude, trip.StartStationLongitude),

			// End Station
			strconv.Itoa(trip.EndStationID),
			trip.EndStationName,
			fmt.Sprint(trip.EndStationLatitude),
			fmt.Sprint(trip.EndStationLongitude),
			geolocate(trip.EndStationLatitude, trip.EndStationLongitude),

			strconv.Itoa(int(trip.BikeID)),
			trip.UserType,
			strconv.Itoa(trip.MemberBirthYear),
			trip.MemberGender,
			bikeshare,
		}

		if err := w.Write(record); err != nil {
			return fmt.Errorf("error writing record %d: %s", i, err)
		}
	}

	// Write any buffered data to the underlying writer (standard output).
	w.Flush()
	return w.Error()
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
