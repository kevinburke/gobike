package stats

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/kevinburke/gobike"
	"github.com/kevinburke/gobike/geo"
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
	weekBeforeEnd := sevenDaysBeforeDataEnd(trips)
	lastSunday := time.Date(weekBeforeEnd.Year(), weekBeforeEnd.Month(), weekBeforeEnd.Day()+(7-int(weekBeforeEnd.Weekday())), 0, 0, 0, 0, tz)
	mp := make(map[string]int)
	earliest := time.Date(3000, time.January, 1, 0, 0, 0, 0, tz)
	for i := 0; i < len(trips); i++ {
		start := trips[i].StartTime
		wday := start.Weekday()
		sunday := time.Date(start.Year(), start.Month(), start.Day()-int(wday), 0, 0, 0, 0, tz)
		if sunday.Equal(lastSunday) || sunday.After(lastSunday) {
			continue
		}
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
	seen := 0
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

func UniqueTripsPerWeek(trips []*gobike.Trip) TimeSeries {
	weekBeforeEnd := sevenDaysBeforeDataEnd(trips)
	lastSunday := time.Date(weekBeforeEnd.Year(), weekBeforeEnd.Month(), weekBeforeEnd.Day()+(7-int(weekBeforeEnd.Weekday())), 0, 0, 0, 0, tz)
	mp := make(map[string]map[[sha256.Size]byte]int)
	earliest := time.Date(3000, time.January, 1, 0, 0, 0, 0, tz)
	for i := 0; i < len(trips); i++ {
		start := trips[i].StartTime
		wday := start.Weekday()
		sunday := time.Date(start.Year(), start.Month(), start.Day()-int(wday), 0, 0, 0, 0, tz)
		if sunday.Equal(lastSunday) || sunday.After(lastSunday) {
			continue
		}
		hash := trips[i].Hash()
		sundayFmt := sunday.Format("2006-01-02")
		_, ok := mp[sundayFmt]
		if !ok {
			mp[sundayFmt] = make(map[[sha256.Size]byte]int)
		}
		if _, ok := mp[sundayFmt][hash]; ok {
			mp[sundayFmt][hash] += 1
		} else {
			mp[sundayFmt][hash] = 1
		}
		if sunday.Before(earliest) {
			earliest = sunday
		}
	}
	seen := 0
	result := make([]*TimeStat, 0)
	for i := earliest; ; i = time.Date(i.Year(), i.Month(), i.Day()+7, 0, 0, 0, 0, tz) {
		count, ok := mp[i.Format("2006-01-02")]
		if ok {
			seen++
		}
		result = append(result, &TimeStat{Date: i, Data: float64(len(count))})
		if seen >= len(mp) {
			break
		}
	}
	return result
}

func BikeShareForAllTripsPerWeek(trips []*gobike.Trip) TimeSeries {
	weekBeforeEnd := sevenDaysBeforeDataEnd(trips)
	lastSunday := time.Date(weekBeforeEnd.Year(), weekBeforeEnd.Month(), weekBeforeEnd.Day()+(7-int(weekBeforeEnd.Weekday())), 0, 0, 0, 0, tz)
	mp := make(map[string]int)
	earliest := time.Date(3000, time.January, 1, 0, 0, 0, 0, tz)
	for i := 0; i < len(trips); i++ {
		if !trips[i].BikeShareForAllTrip {
			continue
		}
		start := trips[i].StartTime
		wday := start.Weekday()
		sunday := time.Date(start.Year(), start.Month(), start.Day()-int(wday), 0, 0, 0, 0, tz)
		if sunday.Equal(lastSunday) || sunday.After(lastSunday) {
			continue
		}
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
	seen := 0
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
	weekBeforeEnd := sevenDaysBeforeDataEnd(trips)
	lastSunday := time.Date(weekBeforeEnd.Year(), weekBeforeEnd.Month(), weekBeforeEnd.Day()+(7-int(weekBeforeEnd.Weekday())), 0, 0, 0, 0, tz)
	mp := make(map[string]map[int]bool)
	earliest := time.Date(3000, time.January, 1, 0, 0, 0, 0, tz)
	for i := 0; i < len(trips); i++ {
		start := trips[i].StartTime
		wday := start.Weekday()
		sunday := time.Date(start.Year(), start.Month(), start.Day()-int(wday), 0, 0, 0, 0, tz)
		if sunday.Equal(lastSunday) || sunday.After(lastSunday) {
			continue
		}
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
	seen := 0
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
	weekBeforeEnd := sevenDaysBeforeDataEnd(trips)
	lastSunday := time.Date(weekBeforeEnd.Year(), weekBeforeEnd.Month(), weekBeforeEnd.Day()+(7-int(weekBeforeEnd.Weekday())), 0, 0, 0, 0, tz)
	mp := make(map[string]map[int64]bool)
	earliest := time.Date(3000, time.January, 1, 0, 0, 0, 0, tz)
	for i := 0; i < len(trips); i++ {
		start := trips[i].StartTime
		wday := start.Weekday()
		sunday := time.Date(start.Year(), start.Month(), start.Day()-int(wday), 0, 0, 0, 0, tz)
		if sunday.Equal(lastSunday) || sunday.After(lastSunday) {
			continue
		}
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
	seen := 0
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
	weekBeforeEnd := sevenDaysBeforeDataEnd(trips)
	lastSunday := time.Date(weekBeforeEnd.Year(), weekBeforeEnd.Month(), weekBeforeEnd.Day()+(7-int(weekBeforeEnd.Weekday())), 0, 0, 0, 0, tz)
	lastSundayFmt := lastSunday.Format("2006-01-02")
	mp := make(map[string]map[int64]int)
	earliest := time.Date(3000, time.January, 1, 0, 0, 0, 0, tz)
	for i := 0; i < len(trips); i++ {
		start := trips[i].StartTime
		wday := start.Weekday()
		sunday := time.Date(start.Year(), start.Month(), start.Day()-int(wday), 0, 0, 0, 0, tz)
		sundayfmt := sunday.Format("2006-01-02")
		if sundayfmt == lastSundayFmt {
			continue // partial last week
		}
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
	seen := 0
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
		var avg float64
		if numTrips+float64(len(weekMap)) > 0.0005 {
			avg = numTrips / float64(len(weekMap))
		}
		result = append(result, &TimeStat{Date: i, Data: avg})
		if seen >= len(mp) {
			break
		}
	}
	return result
}

type DestinationStation struct {
	Station *gobike.Station
	Count   int
}

type StationCount struct {
	Station          *gobike.Station `json:"station"`
	Count            int             `json:"count"`
	WeekdayRidership float64         `json:"weekday_ridership"`
	BS4ACount        int             `json:"bike_share_for_all_count"`
	ToStation        *DestinationStation
	FromStation      *DestinationStation
}

func (s StationCount) BS4APct() string {
	return strings.TrimSuffix(fmt.Sprintf("%.1f", float64(s.BS4ACount)*100/float64(s.Count)), ".0")
}

func (s StationCount) RidershipString() string {
	return strings.TrimSuffix(fmt.Sprintf("%.1f", float64(s.WeekdayRidership)), ".0")
}

type stationAggregate struct {
	AllRides  [7]int
	BS4ARides [7]int
	Station   *gobike.Station

	// Maps of station ID's to counts.
	From map[int]int
	To   map[int]int
}

func stationCounter(trips []*gobike.Trip, f func(t *gobike.Trip) bool) []*StationCount {
	agg := make(map[int]*stationAggregate)
	for i := range trips {
		if trips[i].Dockless() {
			continue
		}
		if !f(trips[i]) {
			continue
		}
		stationID := trips[i].StartStationID
		if _, ok := agg[stationID]; !ok {
			agg[stationID] = &stationAggregate{
				Station: &gobike.Station{
					ID:        stationID,
					Name:      trips[i].StartStationName,
					Latitude:  trips[i].StartStationLatitude,
					Longitude: trips[i].StartStationLongitude,
				},
			}
		}
		toStationID := trips[i].EndStationID
		if _, ok := agg[toStationID]; !ok {
			agg[toStationID] = &stationAggregate{
				Station: &gobike.Station{
					ID:        toStationID,
					Name:      trips[i].EndStationName,
					Latitude:  trips[i].EndStationLatitude,
					Longitude: trips[i].EndStationLongitude,
				},
			}
		}
		if agg[stationID].To == nil {
			agg[stationID].To = make(map[int]int)
		}
		if _, ok := agg[stationID].To[toStationID]; ok {
			agg[stationID].To[toStationID]++
		} else {
			agg[stationID].To[toStationID] = 1
		}
		if agg[toStationID].From == nil {
			agg[toStationID].From = make(map[int]int)
		}
		if _, ok := agg[toStationID].From[stationID]; ok {
			agg[toStationID].From[stationID]++
		} else {
			agg[toStationID].From[stationID] = 1
		}

		agg[stationID].AllRides[trips[i].StartTime.Weekday()]++
		if !trips[i].BikeShareForAllTrip {
			continue
		}
		agg[stationID].BS4ARides[trips[i].StartTime.Weekday()]++
	}
	stationCounts := make([]*StationCount, 0)
	for id := range agg {
		mpbucket := agg[id].AllRides
		mpcount := 0
		for j := 0; j < len(mpbucket); j++ {
			mpcount += mpbucket[j]
		}
		if mpcount == 0 {
			continue
		}
		bmpbucket := agg[id].BS4ARides
		bmpcount := 0
		for j := 0; j < len(bmpbucket); j++ {
			bmpcount += bmpbucket[j]
		}
		toStation, fromStation := new(DestinationStation), new(DestinationStation)
		for i := range agg[id].To {
			if agg[id].To[i] > toStation.Count || agg[id].To[i] == toStation.Count && agg[i].Station.Name > toStation.Station.Name {
				toStation.Count = agg[id].To[i]
				toStation.Station = agg[i].Station
			}
		}
		for i := range agg[id].From {
			if agg[id].From[i] > fromStation.Count || agg[id].From[i] == fromStation.Count && agg[i].Station.Name > fromStation.Station.Name {
				fromStation.Count = agg[id].From[i]
				fromStation.Station = agg[i].Station
			}
		}
		sort.Ints(mpbucket[time.Monday : time.Friday+1])
		stationCounts = append(stationCounts, &StationCount{
			Station:          agg[id].Station,
			Count:            mpcount,
			WeekdayRidership: (float64(mpbucket[time.Tuesday]) + float64(mpbucket[time.Wednesday]) + float64(mpbucket[time.Thursday])) / 3,
			BS4ACount:        bmpcount,
			ToStation:        toStation,
			FromStation:      fromStation,
		})
	}
	return stationCounts
}

func PopularStationsLast7Days(trips []*gobike.Trip, numStations int) []*StationCount {
	weekAgo := sevenDaysBeforeDataEnd(trips)
	stationCounts := stationCounter(trips, func(trip *gobike.Trip) bool {
		return !trip.StartTime.Before(weekAgo)
	})
	sort.Slice(stationCounts, func(i, j int) bool {
		if stationCounts[i].Count > stationCounts[j].Count {
			return true
		}
		if stationCounts[i].Count < stationCounts[j].Count {
			return false
		}
		return stationCounts[i].Station.Name > stationCounts[j].Station.Name
	})
	if numStations > len(stationCounts) {
		return stationCounts
	}
	return stationCounts[:numStations]
}

func sevenDaysBeforeDataEnd(trips []*gobike.Trip) time.Time {
	tzOnce.Do(populateTZ)
	latestDay := time.Date(1000, time.January, 1, 0, 0, 0, 0, tz)
	for i := 0; i < len(trips); i++ {
		if trips[i].StartTime.After(latestDay) {
			latestDay = trips[i].StartTime
		}
	}
	// latestDay is at the end of, say, the 14th
	// a full week is midnight on the 8th, six days.
	return time.Date(latestDay.Year(), latestDay.Month(), latestDay.Day()-6, 0, 0, 0, 0, tz)
}

func PopularBS4AStationsLast7Days(trips []*gobike.Trip, numStations int) []*StationCount {
	weekAgo := sevenDaysBeforeDataEnd(trips)
	stationCounts := stationCounter(trips, func(trip *gobike.Trip) bool {
		return !trip.StartTime.Before(weekAgo)
	})
	sort.Slice(stationCounts, func(i, j int) bool {
		if stationCounts[i].BS4ACount > stationCounts[j].BS4ACount {
			return true
		}
		if stationCounts[i].BS4ACount < stationCounts[j].BS4ACount {
			return false
		}
		return stationCounts[i].Station.Name > stationCounts[j].Station.Name
	})
	if numStations > len(stationCounts) {
		return stationCounts
	}
	return stationCounts[:numStations]
}

func TripsLastWeekPerDistrict(trips []*gobike.Trip) [11]int {
	weekAgo := sevenDaysBeforeDataEnd(trips)
	var counts [11]int
	for i := range geo.SFDistricts {
		district := geo.SFDistricts[i]
		for j := range trips {
			if trips[j].StartTime.Before(weekAgo) {
				continue
			}
			if !district.ContainsPoint(trips[j].StartStationLatitude, trips[j].StartStationLongitude) {
				continue
			}
			counts[i]++
		}
	}
	return counts
}

func AverageWeekdayTrips(trips []*gobike.Trip) float64 {
	// bucket trips by weekday
	weekAgo := sevenDaysBeforeDataEnd(trips)
	var buckets [7]int
	for i := range trips {
		if trips[i].StartTime.Before(weekAgo) {
			continue
		}
		buckets[trips[i].StartTime.Weekday()]++
	}
	sort.Ints(buckets[time.Monday : time.Friday+1])
	// drop highest and lowest
	return (float64(buckets[time.Tuesday]) + float64(buckets[time.Wednesday]) + float64(buckets[time.Thursday])) / 3
}
