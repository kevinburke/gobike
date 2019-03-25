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
	"strconv"
	"time"

	"github.com/kevinburke/gobike"
	"github.com/kevinburke/gobike/client"
	"github.com/kevinburke/rest"
	"golang.org/x/sys/unix"
)

func lock(f *os.File) error {
	return unix.Flock(int(f.Fd()), unix.LOCK_EX)
}

func unlock(f *os.File) error {
	return unix.Flock(int(f.Fd()), unix.LOCK_UN)
}

func writeStation(buf *bytes.Buffer, station *gobike.StationStatus) {
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

func main() {
	version := flag.Bool("version", false, "Print the version string")
	flag.Parse()
	if *version {
		fmt.Fprintf(os.Stderr, "monitor-station-capacity version %s\n", gobike.Version)
		os.Exit(1)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dir := filepath.Join("data", "station-capacity")
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatal(err)
	}
	now := time.Now().UTC()
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
	gobike.ForeachStationStatus(bufio.NewReader(f), func(stationStatus *gobike.StationStatus) error {
		if stationStatus.LastReported.Equal(lastReported[stationStatus.ID]) || stationStatus.LastReported.Before(lastReported[stationStatus.ID]) {
			return nil
		}
		lastReported[stationStatus.ID] = stationStatus.LastReported
		return nil
	})
	ticker := time.NewTicker(10 * time.Second)
	buf := new(bytes.Buffer)
	client := client.NewClient()
	count := 0
	logMessage := false

	rest.Logger.Info("started", "version", gobike.Version, "filename", filename)
	for range ticker.C {
		response, err := client.Stations.Status(ctx)
		if err != nil {
			log.Print("error fetching status: %v\n", err)
			continue
		}
		var station *gobike.StationStatus
		var fullStations, emptyStations int
		for i := 0; i < len(response.Stations); i++ {
			station = response.Stations[i]
			if station.NumDocksAvailable == 0 {
				fullStations++
			}
			if station.NumBikesAvailable == 0 {
				emptyStations++
			}
			if station.LastReported.Equal(lastReported[station.ID]) || station.LastReported.Before(lastReported[station.ID]) {
				continue
			}
			if station.LastReported.After(now) && station.LastReported.Format("2006-01-02") > prefix {
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
				oldName := f.Name()
				filename = station.LastReported.Format("2006-01-02") + "-capacity.csv"
				f, err = os.Create(filepath.Join(dir, filename))
				if err != nil {
					log.Fatal(err)
				}
				prefix = station.LastReported.Format("2006-01-02")
				rest.Logger.Info("rotate file", "old", oldName, "new", filename)
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
