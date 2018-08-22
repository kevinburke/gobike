package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/kevinburke/gobike"
	"github.com/kevinburke/rest"
	"golang.org/x/sys/unix"
)

type StationStatusResponse struct {
	LastUpdated int64              `json:"last_updated"`
	TTL         int                `json:"ttl"`
	Data        *StationStatusData `json:"data"`
}

type StationStatusData struct {
	Stations []*StationStatusJSON `json:"stations"`
}

type StationStatusJSON struct {
	StationID          string `json:"station_id"`
	NumBikesAvailable  int    `json:"num_bikes_available"`
	NumEBikesAvailable int    `json:"num_ebikes_available"`
	NumBikesDisabled   int    `json:"num_bikes_disabled"`
	NumDocksAvailable  int    `json:"num_docks_available"`
	NumDocksDisabled   int    `json:"num_docks_disabled"`
	LastReported       int64  `json:"last_reported"`
	IsInstalled        int    `json:"is_installed"`
	IsRenting          int    `json:"is_renting"`
	IsReturning        int    `json:"is_returning"`
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

func NewStationStatus(ss *StationStatusJSON) *StationStatus {
	return &StationStatus{
		ID:                 ss.StationID,
		NumBikesAvailable:  int16(ss.NumBikesAvailable),
		NumEBikesAvailable: int16(ss.NumEBikesAvailable),
		NumBikesDisabled:   int16(ss.NumBikesDisabled),
		NumDocksAvailable:  int16(ss.NumDocksAvailable),
		NumDocksDisabled:   int16(ss.NumDocksDisabled),
		LastReported:       time.Unix(ss.LastReported, 0),
		IsInstalled:        ss.IsInstalled != 0,
		IsRenting:          ss.IsRenting != 0,
		IsReturning:        ss.IsReturning != 0,
	}
}

func lock(f *os.File) error {
	return unix.Flock(int(f.Fd()), unix.LOCK_EX)
}

func unlock(f *os.File) error {
	return unix.Flock(int(f.Fd()), unix.LOCK_UN)
}

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
	ss.LastReported = t
	line = line[idx+1:]
	idx = bytes.IndexByte(line, ',')
	if idx == -1 {
		return nil, fmt.Errorf("not enough commas: %s", string(line))
	}
	ss.ID = string(line[:idx])
	line = line[idx+1:]
	idx = bytes.IndexByte(line, ',')
	if idx == -1 {
		return nil, fmt.Errorf("not enough commas: %s", string(line))
	}
	bikesAvailable, err := strconv.ParseInt(string(line[:idx]), 10, 16)
	if err != nil {
		return nil, err
	}
	ss.NumBikesAvailable = int16(bikesAvailable)
	return ss, nil
}

func main() {
	flag.Parse()
	dir := filepath.Join("data", "station-capacity")
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatal(err)
	}
	now := time.Now()
	lockfile, err := os.Create(filepath.Join(dir, "capacity.lock"))
	if err != nil {
		log.Fatal(err)
	}
	if err := lock(lockfile); err != nil {
		log.Fatal(err)
	}
	filename := fmt.Sprintf("%d%02d-capacity.csv", now.Year(), int(now.Month()))
	f, err := os.OpenFile(filepath.Join(dir, filename), os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		unlock(lockfile)
		lockfile.Close()
		f.Sync()
		f.Close()
	}()
	lastReported := make(map[string]time.Time)
	f.Seek(0, io.SeekStart)
	bs := bufio.NewScanner(f)
	var stationStatus *StationStatus
	for bs.Scan() {
		stationStatus, err = parseLine(bs.Bytes())
		if err != nil {
			log.Fatal(err)
		}
		if stationStatus.LastReported.Equal(lastReported[stationStatus.ID]) || stationStatus.LastReported.Before(lastReported[stationStatus.ID]) {
			continue
		}
		lastReported[stationStatus.ID] = stationStatus.LastReported
	}
	ticker := time.NewTicker(10 * time.Second)
	buf := new(bytes.Buffer)
	client := rest.NewClient("", "", "https://gbfs.fordgobike.com/gbfs/en")

	for range ticker.C {
		req, err := client.NewRequest("GET", "/station_status.json", nil)
		if err != nil {
			log.Fatal(err)
		}
		req.Header.Set("User-Agent", "gobike/"+gobike.Version+" (github.com/kevinburke/gobike) "+req.Header.Get("User-Agent"))
		body := new(StationStatusResponse)
		if err := client.Do(req, body); err != nil {
			log.Fatal(err)
		}
		stations := body.Data.Stations
		var station *StationStatus
		for i := 0; i < len(stations); i++ {
			station = NewStationStatus(stations[i])
			if station.LastReported.Equal(lastReported[station.ID]) || station.LastReported.Before(lastReported[station.ID]) {
				continue
			}
			buf.WriteString(station.LastReported.Format(time.RFC3339))
			buf.WriteByte(',')
			buf.WriteString(station.ID)
			buf.WriteByte(',')
			buf.WriteString(strconv.FormatInt(int64(station.NumBikesAvailable), 10))
			buf.WriteByte(',')
			buf.WriteString(strconv.FormatInt(int64(station.NumEBikesAvailable), 10))
			buf.WriteByte(',')
			buf.WriteString(strconv.FormatInt(int64(station.NumBikesDisabled), 10))
			buf.WriteByte(',')
			buf.WriteString(strconv.FormatInt(int64(station.NumDocksAvailable), 10))
			buf.WriteByte(',')
			buf.WriteString(strconv.FormatInt(int64(station.NumDocksDisabled), 10))
			buf.WriteByte(',')
			if station.IsInstalled {
				buf.WriteString("t,")
			} else {
				buf.WriteString("f,")
			}
			if station.IsRenting {
				buf.WriteString("t,")
			} else {
				buf.WriteString("f,")
			}
			if station.IsReturning {
				buf.WriteString("t")
			} else {
				buf.WriteString("f")
			}
			buf.WriteByte('\n')
			lastReported[station.ID] = station.LastReported
		}
		if buf.Len() == 0 {
			continue
		}
		if _, err := f.Write(buf.Bytes()); err != nil {
			log.Fatal(err)
		}
		buf.Reset()
	}
}
