package geo

import (
	"sync"

	"github.com/golang/geo/s2"
)

// We generate city polygons using processed OpenStreetMaps GeoJSON data.
//
// The process for adding a new city is as follows:
// - Go to http://nominatim.openstreetmap.org/ and search for the city
// - Click the "details" button in list of search results on the left
// - Look for OSM ID and copy it (control+c), example: 2018776
// - Go to http://polygons.openstreetmap.fr/index.py
// - Paste the ID download the GeoJSON polygon
// - Use gobike-rewind to rewind the polygon (see below for instructions)
// - Validate your GeoJSON using http://geojsonlint.com/.
//   - You should see the following error: "GeometryCollection with a single geometry should be avoided in favor of single part or a single object of multi-part type". This is expected and okay
//   - That should be the only error. If there are more errors, please ask for help.
// - Add a new entry to the Makefile for your city. Use `geo/oakland.go` as a example
// - Regenerate the city Go files in the geo package with `make polygons`
// - Update cmd/gobike-dataset/main.go with the new city
// - Update cmd/gobike-site/main.go with the new city
// - Regenerate the site with `make site`
// - Optionally, regenerate the dataset with `make dataset`
//
// Your new city should now be ready to use.
//
// A note on rewinding GeoJSON files. The official GeoJSON standard requires
// that all polygons follow the right-hand rule with respect to the area it
// bounds, i.e., exterior rings are counterclockwise, and holes are clockwise.
// The GeoJSON files from OpenStreetMap are defined in clockwise fashion and
// must be reversed to work. The gobike-rewind tool does this automatically
//
//     make $GOPATH/bin/gobike-rewind
//     $GOPATH/bin/gobike-rewind ~/Downloads/osm.geojson geojson/city.geojson

func init() {
	Berkeley.Name = "Berkeley"
	Emeryville.Name = "Emeryville"
	Oakland.Name = "Oakland"
	SF.Name = "San Francisco"
	SanJose.Name = "San Jose"
	SFD1.Name = "SFD1"
	SFD2.Name = "SFD2"
	SFD3.Name = "SFD3"
	SFD4.Name = "SFD4"
	SFD5.Name = "SFD5"
	SFD6.Name = "SFD6"
	SFD7.Name = "SFD7"
	SFD8.Name = "SFD8"
	SFD9.Name = "SFD9"
	SFD10.Name = "SFD10"
	SFD11.Name = "SFD11"
}

type City struct {
	Name string

	once   sync.Once
	poly   *s2.Polygon
	points [][][][]float64
}

var SFDistricts = [...]*City{
	SFD1,
	SFD2,
	SFD3,
	SFD4,
	SFD5,
	SFD6,
	SFD7,
	SFD8,
	SFD9,
	SFD10,
	SFD11,
}

func (c *City) ContainsPoint(lat, long float64) bool {
	c.once.Do(func() {
		loops := []*s2.Loop{}
		for _, loop := range c.points {
			pts := []s2.Point{}
			for _, n1 := range loop {
				for i, p := range n1 {
					// golang/geo does not like having the polygon end in the same point
					if i == len(n1)-1 {
						continue
					}
					pts = append(pts, s2.PointFromLatLng(s2.LatLngFromDegrees(p[1], p[0])))
				}
			}
			loops = append(loops, s2.LoopFromPoints(pts))
		}
		poly := s2.PolygonFromOrientedLoops(loops)
		if err := poly.Validate(); err != nil {
			panic(err)
		}
		c.poly = poly
	})
	return c.poly.ContainsPoint(s2.PointFromLatLng(s2.LatLngFromDegrees(lat, long)))
}
