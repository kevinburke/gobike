package geo

import (
	"sync"

	"github.com/golang/geo/r3"
	"github.com/golang/geo/s2"
)

// Adding more cities:
// - Create a new `<city>.go` file
// - Find the points using https://gis.stackexchange.com/a/192298

type City struct {
	once   sync.Once
	loop   *s2.Loop
	points []s2.Point
}

func (c *City) ContainsPoint(lat, long float64) bool {
	c.once.Do(func() {
		c.loop = s2.LoopFromPoints(c.points)
	})
	return c.loop.ContainsPoint(s2.Point{r3.Vector{long, lat, 0}})
}
