package main

import (
	"bufio"
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
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

func writeStation(buf *bytes.Buffer, station *StationStatus) {
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
}

func getStations(ctx context.Context, client *rest.Client) ([]*StationStatus, error) {
	req, err := client.NewRequest("GET", "/station_status.json", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "gobike/"+gobike.Version+" (github.com/kevinburke/gobike) "+req.Header.Get("User-Agent"))
	req = req.WithContext(ctx)
	body := new(StationStatusResponse)
	if err := client.Do(req, body); err != nil {
		return nil, err
	}
	stations := body.Data.Stations
	stationStatuses := make([]*StationStatus, len(stations))
	for i := 0; i < len(stations); i++ {
		stationStatuses[i] = NewStationStatus(stations[i])
	}
	sort.Slice(stationStatuses, func(i, j int) bool {
		return stationStatuses[i].LastReported.Before(stationStatuses[j].LastReported)
	})
	return stationStatuses, nil
}

func main() {
	flag.Parse()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
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
	prefix := now.Format("2006-01-02")
	filename := prefix + "-capacity.csv"
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
	count := 0
	logMessage := false

	for range ticker.C {
		stations, err := getStations(ctx, client)
		if err != nil {
			log.Fatal(err) // TODO: catch errors
		}
		var station *StationStatus
		var fullStations, emptyStations int
		for i := 0; i < len(stations); i++ {
			station = stations[i]
			if station.NumDocksAvailable == 0 {
				fullStations++
			}
			if station.NumBikesAvailable == 0 {
				emptyStations++
			}
			if station.LastReported.Equal(lastReported[station.ID]) || station.LastReported.Before(lastReported[station.ID]) {
				continue
			}
			if station.LastReported.After(now) && station.LastReported.Format("2006-01-02") != prefix {
				// write buf to file
				if _, err := f.Write(buf.Bytes()); err != nil {
					log.Fatal(err)
				}
				if err := f.Sync(); err != nil {
					log.Fatal(err)
				}
				if err := f.Close(); err != nil {
					log.Fatal(err)
				}
				buf.Reset()
				filename = station.LastReported.Format("2006-01-02") + "-capacity.csv"
				f, err = os.Create(filepath.Join(dir, filename))
				if err != nil {
					log.Fatal(err)
				}
			}
			writeStation(buf, station)
			lastReported[station.ID] = station.LastReported
			count++
			if count%5000 == 0 {
				logMessage = true
			}
		}
		if logMessage {
			rest.Logger.Info("Processing", "rows", count, "full_stations", fullStations, "empty_stations", emptyStations)
			logMessage = false
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
