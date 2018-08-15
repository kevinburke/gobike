package stats

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/kevinburke/gobike"
)

var tz *time.Location
var tzOnce sync.Once

func populateTZ() {
	var err error
	tz, err = time.LoadLocation("America/Los_Angeles")
	if err != nil {
		panic(err)
	}
}

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
	tzOnce.Do(populateTZ)
	earliest := time.Date(3000, time.January, 1, 0, 0, 0, 0, tz)
	for i := 0; i < len(trips); i++ {
		start := trips[i].StartTime
		wday := start.Weekday()
		sunday := time.Date(start.Year(), start.Month(), start.Day()-int(wday), 0, 0, 0, 0, tz)
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
	for i := earliest; ; i = time.Date(i.Year(), i.Month(), i.Day()+7, 0, 0, 0, 0, tz) {
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
	tzOnce.Do(populateTZ)
	earliest := time.Date(3000, time.January, 1, 0, 0, 0, 0, tz)
	for i := 0; i < len(trips); i++ {
		start := trips[i].StartTime
		wday := start.Weekday()
		sunday := time.Date(start.Year(), start.Month(), start.Day()-int(wday), 0, 0, 0, 0, tz)
		sundayfmt := sunday.Format("2006-01-02")
		_, ok := mp[sundayfmt]
		if !ok {
			mp[sundayfmt] = make(map[int]bool)
		}
		// only count start station since end station might be in a different
		// city
		mp[sundayfmt][trips[i].StartStationID] = true
		if sunday.Before(earliest) {
			earliest = sunday
		}
	}
	seen := 1
	result := make([]*TimeStat, 0)
	for i := earliest; ; i = time.Date(i.Year(), i.Month(), i.Day()+7, 0, 0, 0, 0, tz) {
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

func UniqueBikesPerWeek(trips []*gobike.Trip) TimeSeries {
	mp := make(map[string]map[int64]bool)
	tzOnce.Do(populateTZ)
	earliest := time.Date(3000, time.January, 1, 0, 0, 0, 0, tz)
	for i := 0; i < len(trips); i++ {
		start := trips[i].StartTime
		wday := start.Weekday()
		sunday := time.Date(start.Year(), start.Month(), start.Day()-int(wday), 0, 0, 0, 0, tz)
		sundayfmt := sunday.Format("2006-01-02")
		_, ok := mp[sundayfmt]
		if !ok {
			mp[sundayfmt] = make(map[int64]bool)
		}
		mp[sundayfmt][trips[i].BikeID] = true
		if sunday.Before(earliest) {
			earliest = sunday
		}
	}
	seen := 1
	result := make([]*TimeStat, 0)
	for i := earliest; ; i = time.Date(i.Year(), i.Month(), i.Day()+7, 0, 0, 0, 0, tz) {
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

func TripsPerBikePerWeek(trips []*gobike.Trip) TimeSeries {
	mp := make(map[string]map[int64]int)
	tzOnce.Do(populateTZ)
	earliest := time.Date(3000, time.January, 1, 0, 0, 0, 0, tz)
	for i := 0; i < len(trips); i++ {
		start := trips[i].StartTime
		wday := start.Weekday()
		sunday := time.Date(start.Year(), start.Month(), start.Day()-int(wday), 0, 0, 0, 0, tz)
		sundayfmt := sunday.Format("2006-01-02")
		_, ok := mp[sundayfmt]
		if !ok {
			mp[sundayfmt] = make(map[int64]int)
		}
		_, hasBike := mp[sundayfmt][trips[i].BikeID]
		if hasBike {
			mp[sundayfmt][trips[i].BikeID]++
		} else {
			mp[sundayfmt][trips[i].BikeID] = 1
		}
		if sunday.Before(earliest) {
			earliest = sunday
		}
	}
	seen := 1
	result := make([]*TimeStat, 0)
	for i := earliest; ; i = time.Date(i.Year(), i.Month(), i.Day()+7, 0, 0, 0, 0, tz) {
		weekMap, ok := mp[i.Format("2006-01-02")]
		if ok {
			seen++
		}
		numTrips := float64(0)
		for j := range weekMap {
			numTrips += float64(weekMap[j])
		}
		avg := numTrips / float64(len(weekMap))
		result = append(result, &TimeStat{Date: i, Data: avg})
		if seen >= len(mp) {
			break
		}
	}
	return result
}
