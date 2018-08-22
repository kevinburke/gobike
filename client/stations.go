package client

import (
	"context"
	"sort"
	"time"

	"github.com/kevinburke/gobike"
)

type StationService struct {
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
