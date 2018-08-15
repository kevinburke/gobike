package geo

import (
	"sync"

	"github.com/golang/geo/r3"
	"github.com/golang/geo/s2"
)

// Adding more cities:
// You can get polygon coordenates in json for using with googlemaps using openstreetmap. Go to http://nominatim.openstreetmap.org/ search a place like "Partido de Ituzaingó"
// Click on "details"
// Look for OSM ID and copy it (control+c), example: 2018776
// paste the ID in http://polygons.openstreetmap.fr/index.py and download the geojson polygon
// https://www.openstreetmap.org/relation/{id}

type City struct {
	once   sync.Once
	poly   *s2.Polygon
	points [][][][]float64
}

func (c *City) ContainsPoint(lat, long float64) bool {
	c.once.Do(func() {
		loops := []*s2.Loop{}
		for _, loop := range c.points {
			points := []s2.Point{}
			for _, n1 := range loop {
				for _, p := range n1 {
					points = append(points, s2.Point{r3.Vector{p[0], p[1], 0}})
				}
			}
			loops = append(loops, s2.LoopFromPoints(points))
		}
		c.poly = s2.PolygonFromLoops(loops)
	})
	return c.poly.ContainsPoint(s2.Point{r3.Vector{long, lat, 0}})
}
