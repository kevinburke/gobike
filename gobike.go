// Package gobike contains tools for interacting with GoBike system data.
//
// GoBike provides CSV's containing trip data for download from
// https://www.fordgobike.com/system-data. The tools here parse those files into
// Go code.
//
// In addition, the binary in cmd/monitor-station-capacity produces CSV's with
// information about system capacity over time. These CSV's can be parsed using
// the LoadCapacity or LoadCapacityDir commands.
package gobike

import (
	"bufio"
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/geo/s2"
	"github.com/kevinburke/gobike/geo"
	"golang.org/x/sync/errgroup"
)

const Version = "0.8"

// This station is not present in the public station list, but trips reference
// it, so we have to match for it when iterating through trips.
const DepotStationID = "344"

var tz *time.Location
var tzOnce sync.Once

func populateTZ() {
	var err error
	tz, err = time.LoadLocation("America/Los_Angeles")
	if err != nil {
		panic(err)
	}
}

type Station struct {
	ID              int      `json:"id"`
	Name            string   `json:"name"`
	ShortName       string   `json:"short_name"`
	Longitude       float64  `json:"longitude"`
	Latitude        float64  `json:"latitude"`
	RegionID        int      `json:"region_id"`
	Capacity        int      `json:"capacity"`
	HasKiosk        bool     `json:"has_kiosk"`
	RentalMethods   []string `json:"rental_methods"`
	RentalURL       string   `json:"rental_url"`
	HasKeyDispenser bool     `json:"eightd_has_key_dispenser"`

	City *geo.City
}

type Trip struct {
	Duration  time.Duration
	StartTime time.Time
	EndTime   time.Time

	StartStationID        string
	StartStationName      string
	StartStationLongitude float64
	StartStationLatitude  float64

	EndStationID        string
	EndStationName      string
	EndStationLongitude float64
	EndStationLatitude  float64

	BikeID              int64
	UserType            string
	MemberBirthYear     int
	MemberGender        string
	BikeShareForAllTrip bool
}

// SingleRidePriceCents is the price of a single ride in cents ($2.19). Include
// the credit card processing fee since it seems like most trips are paid for
// using credit cards, and we can't guess.
const SingleRidePriceCents = 219

const EstimatedTripsPerSubscriberPerYear = 120

// EstimatedBikeShareForAllSingleRideRevenueCents estimates the per-trip revenue
// from a Bike Share For All ride. The program costs $5 per year and we estimate
// the average user makes 120 trips per year.
const EstimatedBikeShareForAllSingleRideRevenueCents int = 500 / EstimatedTripsPerSubscriberPerYear

// A subscription costs $15 per month ($180 per year) or $145 per year if
// prepaid. We estimate half of subscribers choose each subscription type, and
// the average subscriber makes 120 trips per year.
const EstimatedSubscriberSingleRideRevenueCents int = 16250 / EstimatedTripsPerSubscriberPerYear

// Revenue provides an estimate of the revenue from the trip. This is
// approximate since we don't know how many trips are taken by the average
// subscriber.
func (t Trip) RevenueCents() int {
	switch t.UserType {
	case "Customer":
		return SingleRidePriceCents
	case "Subscriber":
		if t.BikeShareForAllTrip {
			return EstimatedBikeShareForAllSingleRideRevenueCents
		}
		return EstimatedSubscriberSingleRideRevenueCents
	default:
		panic("unknown user type " + t.UserType)
	}
}

func (t Trip) Dockless() bool {
	return t.StartStationID == "" || t.StartStationName == "NULL" ||
		t.EndStationID == "" || t.EndStationName == "NULL"
}

const earthRadiusMiles = 3959.0

func (t Trip) Distance() float64 {
	start := s2.LatLngFromDegrees(t.StartStationLatitude, t.StartStationLongitude)
	end := s2.LatLngFromDegrees(t.EndStationLatitude, t.EndStationLongitude)
	dist := start.Distance(end)
	return earthRadiusMiles * dist.Radians()
}

func parseTrip(record []string) (*Trip, error) {
	// "duration_sec","start_time","end_time","start_station_id","start_station_name","start_station_latitude","start_station_longitude","end_station_id","end_station_name","end_station_latitude","end_station_longitude","bike_id","user_type","member_birth_year","member_gender","bike_share_for_all_trip"
	tzOnce.Do(populateTZ)
	t := new(Trip)
	sec, err := strconv.Atoi(record[0])
	if err != nil {
		return nil, err
	}
	t.Duration = time.Duration(sec) * time.Second
	startTime, err := time.ParseInLocation("2006-01-02 15:04:05", record[1], tz)
	if err != nil {
		return nil, err
	}
	t.StartTime = startTime
	endTime, err := time.ParseInLocation("2006-01-02 15:04:05", record[2], tz)
	if err != nil {
		return nil, err
	}
	t.EndTime = endTime
	if record[3] != "NULL" {
		if record[3] == "347" {
			record[3] = "136" // san bruno ave and 23rd st.
		}
		// for the moment we expect station ID's to be integers. error if we get
		// anything else back in case we have integer-dependent code elsewhere
		// that might be corrupted.
		_, err := strconv.Atoi(record[3])
		if err != nil {
			return nil, err
		}
		t.StartStationID = record[3]
	}
	t.StartStationName = record[4]
	slat, err := strconv.ParseFloat(record[5], 64)
	if err != nil {
		return nil, err
	}
	t.StartStationLatitude = slat
	slng, err := strconv.ParseFloat(record[6], 64)
	if err != nil {
		return nil, err
	}
	t.StartStationLongitude = slng
	if record[7] != "NULL" {
		if record[7] == "347" {
			record[7] = "136" // san bruno ave and 23rd st.
		}
		// for the moment we expect station ID's to be integers. error if we get
		// anything else back in case we have integer-dependent code elsewhere
		// that might be corrupted.
		_, err := strconv.Atoi(record[7])
		if err != nil {
			return nil, err
		}
		t.EndStationID = record[7]
	}
	t.EndStationName = record[8]
	elat, err := strconv.ParseFloat(record[9], 64)
	if err != nil {
		return nil, err
	}
	t.EndStationLatitude = elat
	elng, err := strconv.ParseFloat(record[10], 64)
	if err != nil {
		return nil, err
	}
	t.EndStationLongitude = elng
	id, err := strconv.ParseInt(record[11], 10, 64)
	if err != nil {
		return nil, err
	}
	t.BikeID = id
	t.UserType = record[12]
	if record[13] != "" {
		birthYear, err := strconv.Atoi(record[13])
		if err != nil {
			return nil, err
		}
		t.MemberBirthYear = birthYear
	}
	t.MemberGender = record[14]

	if len(record) == 16 {
		switch record[15] {
		case "No":
			t.BikeShareForAllTrip = false
		case "Yes":
			t.BikeShareForAllTrip = true
		default:
			panic("unknown record 15 " + record[15])
		}
	}

	return t, nil
}

// LoadDir loads all trip CSV's in a given directory.
func LoadDir(directory string) ([]*Trip, error) {
	files, err := ioutil.ReadDir(directory)
	if err != nil {
		return nil, err
	}
	group, errctx := errgroup.WithContext(context.Background())

	trips := make([]*Trip, 0)
	var mu sync.Mutex
	for _, file := range files {
		file := file
		if !strings.HasSuffix(file.Name(), "-fordgobike-tripdata.csv") {
			continue
		}
		group.Go(func() error {
			f, err := os.Open(filepath.Join(directory, file.Name()))
			if err != nil {
				return err
			}
			deadline, ok := errctx.Deadline()
			if ok {
				f.SetDeadline(deadline)
			}
			fileTrips, err := Load(bufio.NewReader(f))
			if err != nil {
				return err
			}
			if err := f.Close(); err != nil {
				return err
			}
			mu.Lock()
			defer mu.Unlock()
			trips = append(trips, fileTrips...)
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err
	}
	sort.Slice(trips, func(i, j int) bool {
		return trips[i].StartTime.Before(trips[j].StartTime)
	})
	return trips, nil
}

func Load(rdr io.Reader) ([]*Trip, error) {
	r := csv.NewReader(rdr)
	r.ReuseRecord = true
	trips := make([]*Trip, 0)
	for i := 0; ; i++ {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if i == 0 {
			// header row
			continue
		}
		t, err := parseTrip(record)
		if err != nil {
			return nil, err
		}
		trips = append(trips, t)
	}
	return trips, nil
}

type StationStatus struct {
	ID                 string    `json:"station_id"`
	NumBikesAvailable  int16     `json:"num_bikes_available"`
	NumEBikesAvailable int16     `json:"num_ebikes_available"`
	NumBikesDisabled   int16     `json:"num_bikes_disabled"`
	NumDocksAvailable  int16     `json:"num_docks_available"`
	NumDocksDisabled   int16     `json:"num_docks_disabled"`
	LastReported       time.Time `json:"last_reported"`
	IsInstalled        bool      `json:"is_installed"`
	IsRenting          bool      `json:"is_renting"`
	IsReturning        bool      `json:"is_returning"`
}

func parseInt16(line []byte) ([]byte, int16, error) {
	idx := bytes.IndexByte(line, ',')
	if idx == -1 {
		return nil, 0, fmt.Errorf("not enough commas: %s", string(line))
	}
	val, err := strconv.ParseInt(string(line[:idx]), 10, 16)
	if err != nil {
		return nil, 0, err
	}
	line = line[idx+1:]
	return line, int16(val), nil
}

var rentingReturningInstalled = []byte{'t', ',', 't', ',', 't', ','}

func parseLine(line []byte) (*StationStatus, error) {
	idx := bytes.IndexByte(line, ',')
	if idx == -1 {
		return nil, fmt.Errorf("not enough commas: %s", string(line))
	}
	t, err := time.Parse(time.RFC3339, string(line[:idx]))
	if err != nil {
		return nil, err
	}
	ss := new(StationStatus)
	ss.LastReported = t.In(tz)
	line = line[idx+1:]
	idx = bytes.IndexByte(line, ',')
	if idx == -1 {
		return nil, fmt.Errorf("not enough commas: %s", string(line))
	}
	ss.ID = string(line[:idx])
	line = line[idx+1:]

	var intVal int16
	line, intVal, err = parseInt16(line)
	if err != nil {
		return nil, err
	}
	ss.NumBikesAvailable = intVal

	line, intVal, err = parseInt16(line)
	if err != nil {
		return nil, err
	}
	ss.NumEBikesAvailable = intVal

	line, intVal, err = parseInt16(line)
	if err != nil {
		return nil, err
	}
	ss.NumBikesDisabled = intVal

	line, intVal, err = parseInt16(line)
	if err != nil {
		return nil, err
	}
	ss.NumDocksAvailable = intVal

	line, intVal, err = parseInt16(line)
	if err != nil {
		return nil, err
	}
	ss.NumDocksDisabled = intVal
	if bytes.Equal(line[:6], rentingReturningInstalled) {
		ss.IsInstalled = true
		ss.IsRenting = true
		ss.IsReturning = true
	} else {
		ss.IsInstalled = line[0] == 't'
		line = line[2:]
		ss.IsRenting = line[0] == 't'
		line = line[2:]
		ss.IsReturning = line[0] == 't'
	}
	return ss, nil
}

func ForeachStationStatus(r io.Reader, f func(*StationStatus) error) error {
	tzOnce.Do(populateTZ)
	bs := bufio.NewScanner(r)
	for bs.Scan() {
		stationStatus, err := parseLine(bs.Bytes())
		if err != nil {
			return err
		}
		if err := f(stationStatus); err != nil {
			return err
		}
	}
	return nil
}

func LoadCapacity(r io.Reader) ([]*StationStatus, error) {
	tzOnce.Do(populateTZ)
	bs := bufio.NewScanner(r)
	statuses := make([]*StationStatus, 0)
	for bs.Scan() {
		stationStatus, err := parseLine(bs.Bytes())
		if err != nil {
			return nil, err
		}
		statuses = append(statuses, stationStatus)
	}
	return statuses, nil
}

func StationMap(stations []*Station) map[string]*Station {
	stationMap := make(map[string]*Station, len(stations))
	for i := range stations {
		stationMap[strconv.Itoa(stations[i].ID)] = stations[i]
	}
	return stationMap
}

func LoadCapacityDir(directory string) ([]*StationStatus, error) {
	files, err := ioutil.ReadDir(directory)
	if err != nil {
		return nil, err
	}
	group, errctx := errgroup.WithContext(context.Background())
	statuses := make([]*StationStatus, 0)
	var mu sync.Mutex
	for _, file := range files {
		file := file
		if !strings.HasSuffix(file.Name(), "-capacity.csv") {
			continue
		}
		group.Go(func() error {
			f, err := os.Open(filepath.Join(directory, file.Name()))
			if err != nil {
				return err
			}
			deadline, ok := errctx.Deadline()
			if ok {
				f.SetDeadline(deadline)
			}
			fileStatuses, err := LoadCapacity(bufio.NewReader(f))
			if err != nil {
				return err
			}
			if err := f.Close(); err != nil {
				return err
			}
			mu.Lock()
			defer mu.Unlock()
			statuses = append(statuses, fileStatuses...)
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err
	}
	return statuses, nil
}
