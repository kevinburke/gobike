package gobike

import (
	"sync"

	"github.com/golang/geo/r3"
	"github.com/golang/geo/s2"
)

var sfPoints = []s2.Point{
	s2.Point{r3.Vector{-122.617, 37.811, 0}},
	s2.Point{r3.Vector{-122.618, 37.815, 0}},
	s2.Point{r3.Vector{-122.616, 37.818, 0}},
	s2.Point{r3.Vector{-122.533, 37.819, 0}},
	s2.Point{r3.Vector{-122.533, 37.822, 0}},
	s2.Point{r3.Vector{-122.53, 37.825, 0}},
	s2.Point{r3.Vector{-122.528, 37.825, 0}},
	s2.Point{r3.Vector{-122.525, 37.829, 0}},
	s2.Point{r3.Vector{-122.51, 37.829, 0}},
	s2.Point{r3.Vector{-122.504, 37.826, 0}},
	s2.Point{r3.Vector{-122.497, 37.826, 0}},
	s2.Point{r3.Vector{-122.492, 37.831, 0}},
	s2.Point{r3.Vector{-122.483, 37.831, 0}},
	s2.Point{r3.Vector{-122.482, 37.835, 0}},
	s2.Point{r3.Vector{-122.478, 37.838, 0}},
	s2.Point{r3.Vector{-122.472, 37.837, 0}},
	s2.Point{r3.Vector{-122.423, 37.855, 0}},
	s2.Point{r3.Vector{-122.436, 37.929, 0}},
	s2.Point{r3.Vector{-122.436, 37.932, 0}},
	s2.Point{r3.Vector{-122.431, 37.934, 0}},
	s2.Point{r3.Vector{-122.37, 37.886, 0}},
	s2.Point{r3.Vector{-122.343, 37.813, 0}},
	s2.Point{r3.Vector{-122.278, 37.71, 0}},
	s2.Point{r3.Vector{-122.278, 37.706, 0}},
	s2.Point{r3.Vector{-122.281, 37.704, 0}},
	s2.Point{r3.Vector{-122.577, 37.703, 0}},
	s2.Point{r3.Vector{-122.579, 37.705, 0}},
	s2.Point{r3.Vector{-122.585, 37.76, 0}},
	s2.Point{r3.Vector{-122.592, 37.787, 0}},
	s2.Point{r3.Vector{-122.603, 37.8, 0}},
	s2.Point{r3.Vector{-122.617, 37.811, 0}},
}

type City struct {
	once   sync.Once
	loop   *s2.Loop
	points []s2.Point
}

var SF = City{
	points: sfPoints,
}

var sfLoop *s2.Loop
var sfLoopOnce sync.Once

func makeSFLoop() {
	sfLoop = s2.LoopFromPoints(sfPoints)
}

func (c *City) ContainsPoint(lat, long float64) bool {
	c.once.Do(func() {
		c.loop = s2.LoopFromPoints(c.points)
	})
	return c.loop.ContainsPoint(s2.Point{r3.Vector{long, lat, 0}})
}
