package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/kevinburke/gobike"
	_ "github.com/lib/pq"
)

type Station struct {
	ID        int
	Name      string
	Longitude float64
	Latitude  float64
	City      string
}

const insertStation = `
INSERT INTO stations (
  id,
  name,
  longitude,
  latitude,
  city
) VALUES (
  $1,
  $2,
  $3,
  $4,
  $5
) ON CONFLICT DO NOTHING
`

const insertTrip = `
INSERT INTO trips (
  duration,
  during,
  start_station,
  end_station,
  bike,
  user_type,
  member_gender,
  member_birth_year,
  bike_share_for_all
) VALUES (
  $1,
  tstzrange($2,$3),
  $4,
  $5,
  $6,
  case when $7 = '' then null else $7 end,
  case when $8 = '' then null else $8 end,
  case when $9 = 0 then null else $9 end,
  $10
)
`

func run() error {
	// loc, err := time.LoadLocation("America/Los_Angeles")
	// if err != nil {
	// 	return err
	// }

	db, err := sql.Open("postgres", "postgres://localhost/gobike?sslmode=disable")
	if err != nil {
		return err
	}

	stationStmt, err := db.Prepare(insertStation)
	if err != nil {
		return err
	}

	tripStmt, err := db.Prepare(insertTrip)
	if err != nil {
		return err
	}

	files, err := ioutil.ReadDir("../../testdata")
	if err != nil {
		return err
	}

	trips := make([]*gobike.Trip, 0)
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), "-fordgobike-tripdata.csv") {
			continue
		}
		f, err := os.Open(filepath.Join("../../testdata", file.Name()))
		if err != nil {
			return err
		}
		fileTrips, err := gobike.Load(f)
		if err != nil {
			return err
		}
		trips = append(trips, fileTrips...)
	}

	geolocate := func(lat, long float64) string {
		switch {
		case gobike.SF.ContainsPoint(lat, long):
			return "San Francisco"
		case gobike.Oakland.ContainsPoint(lat, long):
			return "Oakland"
		case gobike.SanJose.ContainsPoint(lat, long):
			return "San Jose"
		default:
			return "Other"
		}
	}

	stations := make(map[int]Station, 0)
	for _, trip := range trips {
		if _, ok := stations[trip.StartStationID]; !ok {
			stations[trip.StartStationID] = Station{
				ID:        trip.StartStationID,
				Name:      trip.StartStationName,
				Longitude: trip.StartStationLongitude,
				Latitude:  trip.StartStationLatitude,
				City:      geolocate(trip.StartStationLatitude, trip.StartStationLongitude),
			}
		}
		if _, ok := stations[trip.EndStationID]; !ok {
			stations[trip.EndStationID] = Station{
				ID:        trip.EndStationID,
				Name:      trip.EndStationName,
				Longitude: trip.EndStationLongitude,
				Latitude:  trip.EndStationLatitude,
				City:      geolocate(trip.EndStationLatitude, trip.EndStationLongitude),
			}
		}
	}

	for _, station := range stations {
		_, err := stationStmt.Exec(
			station.ID,
			station.Name,
			station.Longitude,
			station.Latitude,
			station.City,
		)
		if err != nil {
			return fmt.Errorf("insert failed %s: %s", station.Name, err)
		}
	}

	for i, trip := range trips {
		if i%10000 == 0 {
			fmt.Println("Added 10,000 more trips")
		}
		if trip.EndTime.Before(trip.StartTime) {
			// WTF?
			continue
		}
		_, err := tripStmt.Exec(
			int(trip.Duration.Seconds()),
			trip.StartTime,
			trip.EndTime,
			trip.StartStationID,
			trip.EndStationID,
			trip.BikeID,
			trip.UserType,
			trip.MemberGender,
			trip.MemberBirthYear,
			trip.BikeShareForAllTrip,
		)
		if err != nil {
			return fmt.Errorf("insert failed %d: %s", trip.Duration, err)
		}
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
