package geo

import (
	"testing"
)

func TestStationCityMapping(t *testing.T) {
	stations := []struct {
		city *City
		name string
		lat  float64
		long float64
	}{
		// {Berkeley, "Telegraph Ave at Ashby Ave", 37.8559558, -122.2597949},
		{Emeryville, "Emeryville Town Hall", 37.8312752, -122.2856333},
		// {Oakland, "19th Street BART Station", 37.8090126, -122.2682473},
		// {SanJose, "Pierce Ave at Market St", 37.327581, -121.884559},
		// {SF, "Market St at 10th St", 37.776619, -122.417385},
		// {SF, "Montgomery St BART Station (Market St at 2nd St)", 37.7896254, -122.400811},
	}

	// Station #21
	for _, station := range stations {
		test := station
		t.Run(test.name, func(t *testing.T) {
			if !test.city.ContainsPoint(test.lat, test.long) {
				t.Errorf("%s is in the wrong city", test.name)
			}
		})
	}
}
