package geo

import (
	"fmt"
	"sync"

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
	fmt.Println("count", len(c.poly.Loops()))
	fmt.Println("edges", c.poly.Loop(0).NumEdges())
	fmt.Println("verts", c.poly.Loop(0).NumVertices())
	return c.poly.ContainsPoint(s2.PointFromLatLng(s2.LatLngFromDegrees(lat, long)))
}
