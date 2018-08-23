package client

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/kevinburke/gobike"
)

type StationService struct {
	// Set CacheTTL to a nonzero value to load station data from a local cache.
	// If the cached data is older than the TTL we will ignore it. The number of
	// stations does not change often, so the cached value may be good enough.
	CacheTTL time.Duration
	// Directory holding data files on disk, if empty, "data" is assumed.
	DataDir string

	client *Client
}

type StationStatusResponse struct {
	Response
	Stations []*gobike.StationStatus
}

type stationStatusResponse struct {
	response
	Data *stationStatusData `json:"data"`
}

type stationStatusData struct {
	Stations []*stationStatusJSON `json:"stations"`
}

type stationStatusJSON struct {
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

func newStationStatus(ss *stationStatusJSON) *gobike.StationStatus {
	return &gobike.StationStatus{
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

func (s *StationService) loadStationsFromDisk() (*StationResponse, error) {
	if s.CacheTTL == 0 {
		return nil, errors.New("cache set to zero")
	}
	dataDir := s.DataDir
	if dataDir == "" {
		dataDir = "data"
	}
	data, err := ioutil.ReadFile(filepath.Join(dataDir, "station_information.json"))
	if err != nil {
		return nil, err
	}
	body := new(stationResponse)
	if err := json.Unmarshal(data, body); err != nil {
		return nil, err
	}
	resp, err := buildStations(body)
	if err != nil {
		return nil, err
	}
	if time.Since(resp.LastUpdated) > s.CacheTTL {
		return nil, errors.New("local data too old")
	}
	return resp, nil
}

func buildStations(body *stationResponse) (*StationResponse, error) {
	stationJSONs := body.Data.Stations
	stations := make([]*gobike.Station, len(stationJSONs))
	for i := 0; i < len(stationJSONs); i++ {
		id, err := strconv.Atoi(stationJSONs[i].ID)
		if err != nil {
			return nil, err
		}
		stations[i] = &gobike.Station{
			ID:        id,
			Name:      stationJSONs[i].Name,
			ShortName: stationJSONs[i].ShortName,
			Latitude:  stationJSONs[i].Latitude,
			Longitude: stationJSONs[i].Longitude,
			RegionID:  stationJSONs[i].RegionID,
			Capacity:  stationJSONs[i].Capacity,
			HasKiosk:  stationJSONs[i].HasKiosk,
		}
	}
	sort.Slice(stations, func(i, j int) bool {
		return stations[i].ID < stations[j].ID
	})
	return &StationResponse{
		Response: Response{
			LastUpdated: time.Unix(body.LastUpdated, 0),
			TTL:         body.TTL,
		},
		Stations: stations,
	}, nil
}

func (s *StationService) All(ctx context.Context) (*StationResponse, error) {
	if stations, err := s.loadStationsFromDisk(); err == nil {
		return stations, nil
	}
	req, err := s.client.NewRequest("GET", "/station_information.json", nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	body := new(stationResponse)
	if err := s.client.Client.Do(req, body); err != nil {
		return nil, err
	}
	return buildStations(body)
}

type stationResponse struct {
	response
	Data *stationData `json:"data"`
}

type stationData struct {
	Stations []*stationJSON `json:"stations"`
}

type stationJSON struct {
	ID        string  `json:"station_id"`
	Name      string  `json:"name"`
	ShortName string  `json:"short_name"`
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lon"`
	RegionID  int     `json:"region_id"`
	Capacity  int     `json:"capacity"`
	HasKiosk  bool    `json:"has_kiosk"`
}

type StationResponse struct {
	Response
	Stations []*gobike.Station
}

func (s *StationService) Status(ctx context.Context) (*StationStatusResponse, error) {
	req, err := s.client.NewRequest("GET", "/station_status.json", nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	body := new(stationStatusResponse)
	if err := s.client.Client.Do(req, body); err != nil {
		return nil, err
	}
	stations := body.Data.Stations
	stationStatuses := make([]*gobike.StationStatus, len(stations))
	for i := 0; i < len(stations); i++ {
		stationStatuses[i] = newStationStatus(stations[i])
	}
	sort.Slice(stationStatuses, func(i, j int) bool {
		return stationStatuses[i].LastReported.Before(stationStatuses[j].LastReported)
	})
	return &StationStatusResponse{
		Response: Response{
			LastUpdated: time.Unix(body.LastUpdated, 0),
			TTL:         body.TTL,
		},
		Stations: stationStatuses,
	}, nil
}
