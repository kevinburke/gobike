package stats

import (
	"encoding/json"
	"time"

	"github.com/kevinburke/gobike"
)

type TimeSeries []*TimeStat

func (t TimeSeries) MarshalJSON() ([]byte, error) {
	a := make([][2]float64, len(t))
	for i := 0; i < len(t); i++ {
		a[i][0] = float64(t[i].Date.Unix() * 1000)
		a[i][1] = t[i].Data
	}
	return json.Marshal(a)
}

type TimeStat struct {
	Date time.Time
	Data float64
}

func TripsPerWeek(trips []*gobike.Trip) TimeSeries {
	mp := make(map[string]int)
	earliest := time.Date(3000, time.January, 1, 0, 0, 0, 0, time.Local)
	for i := 0; i < len(trips); i++ {
		start := trips[i].StartTime
		wday := start.Weekday()
		sunday := time.Date(start.Year(), start.Month(), start.Day()-int(wday), 0, 0, 0, 0, time.Local)
		_, ok := mp[sunday.Format("2006-01-02")]
		if ok {
			mp[sunday.Format("2006-01-02")] += 1
		} else {
			mp[sunday.Format("2006-01-02")] = 1
		}
		if sunday.Before(earliest) {
			earliest = sunday
		}
	}
	seen := 1
	result := make([]*TimeStat, 0)
	for i := earliest; ; i = time.Date(i.Year(), i.Month(), i.Day()+7, 0, 0, 0, 0, time.Local) {
		count, ok := mp[i.Format("2006-01-02")]
		if ok {
			seen++
		}
		result = append(result, &TimeStat{Date: i, Data: float64(count)})
		if seen >= len(mp) {
			break
		}
	}
	return result
}

func UniqueStationsPerWeek(trips []*gobike.Trip) TimeSeries {
	mp := make(map[string]map[int]bool)
	earliest := time.Date(3000, time.January, 1, 0, 0, 0, 0, time.Local)
	for i := 0; i < len(trips); i++ {
		start := trips[i].StartTime
		wday := start.Weekday()
		sunday := time.Date(start.Year(), start.Month(), start.Day()-int(wday), 0, 0, 0, 0, time.Local)
		sundayfmt := sunday.Format("2006-01-02")
		_, ok := mp[sundayfmt]
		if !ok {
			mp[sundayfmt] = make(map[int]bool)
		}
		mp[sundayfmt][trips[i].StartStationID] = true
		mp[sundayfmt][trips[i].EndStationID] = true
		if sunday.Before(earliest) {
			earliest = sunday
		}
	}
	seen := 1
	result := make([]*TimeStat, 0)
	for i := earliest; ; i = time.Date(i.Year(), i.Month(), i.Day()+7, 0, 0, 0, 0, time.Local) {
		weekMap, ok := mp[i.Format("2006-01-02")]
		if ok {
			seen++
		}
		result = append(result, &TimeStat{Date: i, Data: float64(len(weekMap))})
		if seen >= len(mp) {
			break
		}
	}
	return result
}
