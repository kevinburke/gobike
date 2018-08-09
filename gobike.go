package gobike

import (
	"encoding/csv"
	"io"
	"strconv"
	"time"
)

type Trip struct {
	Duration  time.Duration
	StartTime time.Time
	EndTime   time.Time

	StartStationID        int
	StartStationName      string
	StartStationLongitude float64
	StartStationLatitude  float64

	EndStationID        int
	EndStationName      string
	EndStationLongitude float64
	EndStationLatitude  float64

	BikeID              int64
	UserType            string
	MemberBirthYear     int
	MemberGender        string
	BikeShareForAllTrip bool
}

func parseTrip(record []string) (*Trip, error) {
	// "duration_sec","start_time","end_time","start_station_id","start_station_name","start_station_latitude","start_station_longitude","end_station_id","end_station_name","end_station_latitude","end_station_longitude","bike_id","user_type","member_birth_year","member_gender","bike_share_for_all_trip"
	t := new(Trip)
	sec, err := strconv.Atoi(record[0])
	if err != nil {
		return nil, err
	}
	t.Duration = time.Duration(sec) * time.Second
	startTime, err := time.Parse("2006-01-02 15:04:05", record[1])
	if err != nil {
		return nil, err
	}
	t.StartTime = startTime
	endTime, err := time.Parse("2006-01-02 15:04:05", record[2])
	if err != nil {
		return nil, err
	}
	t.EndTime = endTime
	if record[3] != "NULL" {
		stationID, err := strconv.Atoi(record[3])
		if err != nil {
			return nil, err
		}
		t.StartStationID = stationID
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
		stationID, err := strconv.Atoi(record[7])
		if err != nil {
			return nil, err
		}
		t.EndStationID = stationID
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
