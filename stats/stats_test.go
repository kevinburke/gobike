package stats

import (
	"testing"
	"time"

	"github.com/kevinburke/gobike"
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
