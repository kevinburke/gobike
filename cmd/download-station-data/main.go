package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/kevinburke/gobike/client"
)

func main() {
	flag.Parse()
	c := client.NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := c.Stations.All(ctx)
	if err != nil {
		log.Fatal(err)
	}
	data, err := json.MarshalIndent(resp, "", "    ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
