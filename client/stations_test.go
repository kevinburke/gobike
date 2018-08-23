package client

import (
	"context"
	"fmt"
	"testing"
)

func TestStationsAll(t *testing.T) {
	c := NewClient()
	stations, err := c.Stations.All(context.TODO())
	if err != nil {
		t.Fatal(err)
	}
	for i := range stations.Stations {
		fmt.Println(stations.Stations[i].ID, stations.Stations[i].Name)
	}
}
