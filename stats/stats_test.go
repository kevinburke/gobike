package stats

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/kevinburke/gobike"
	"github.com/kevinburke/gobike/client"
	"github.com/kevinburke/gobike/geo"
)

func TestWeekAgoChoosesCorrectDay(t *testing.T) {
	tzOnce.Do(populateTZ)
	aDay := time.Date(2018, time.August, 16, 23, 59, 59, 0, tz)
	trip := &gobike.Trip{
		StartTime: aDay,
	}
	weekAgo := sevenDaysBeforeDataEnd([]*gobike.Trip{trip})
	diff := aDay.Sub(weekAgo)
	days := float64(diff) / float64(time.Hour*24)
	if days >= 7.3 || days <= 6.7 {
		t.Errorf("wrong number of days (should be ~7): %f", days)
	}
}

func empty(city *geo.City, stationMap map[string]*gobike.Station) func(ss *gobike.StationStatus) bool {
	return func(ss *gobike.StationStatus) bool {
		station := stationMap[ss.ID]
		// yuck pointer comparison
		if city != nil && station.City != city {
			return false
		}
		if !ss.IsInstalled || !ss.IsRenting {
			return false
		}
		return ss.NumBikesAvailable == 0
	}
}

var tsSink TimeSeries

func BenchmarkStatusFilterOverTime(b *testing.B) {
	tzOnce.Do(populateTZ)
	statuses, err := gobike.LoadCapacityDir("testdata")
	if err != nil {
		b.Fatal(err)
	}
	c := client.NewClient()
	c.Stations.CacheTTL = 24 * 14 * time.Hour
	resp, err := c.Stations.All(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	stationMap := gobike.StationMap(resp.Stations)
	byStation := StatusMap(statuses)
	start := time.Date(2018, time.August, 23, 0, 0, 0, 0, tz)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		tsSink = StatusFilterOverTime(byStation, empty(geo.SanJose, stationMap), start, start.Add(7*24*time.Hour), 20*time.Minute)
	}
}
