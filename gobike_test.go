package gobike

import (
	"bufio"
	"encoding/csv"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	f, err := os.Open(filepath.Join("testdata", "golden.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	trips, err := Load(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(trips) == 0 {
		t.Fatal("expected to see non-zero trips, got 0")
	}
}

func BenchmarkLoad(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		f, err := os.Open(filepath.Join("testdata", "golden.csv"))
		if err != nil {
			b.Fatal(err)
		}
		inf, err := f.Stat()
		if err != nil {
			b.Fatal(err)
		}
		b.SetBytes(inf.Size())
		trips, err := Load(bufio.NewReader(f))
		if err != nil {
			b.Fatal(err)
		}
		if len(trips) == 0 {
			b.Fatal("expected to see non-zero trips, got 0")
		}
		if err := f.Close(); err != nil {
			b.Fatal(err)
		}
	}
}

func officialDatasets(tb testing.TB) []string {
	tb.Helper()
	files, err := ioutil.ReadDir("testdata")
	if err != nil {
		tb.Fatal(err)
	}
	csvs := []string{}
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), "-fordgobike-tripdata.csv") {
			continue
		}
		csvs = append(csvs, filepath.Join("testdata", file.Name()))
	}
	return csvs
}

func TestLoadOfficial(t *testing.T) {
	paths := officialDatasets(t)
	if len(paths) == 0 {
		t.Skip("No FordGo CSVs in the testdata directory")
	}
	for _, tripdata := range paths {
		trippath := tripdata
		t.Run(filepath.Base(trippath), func(t *testing.T) {
			f, err := os.Open(trippath)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			trips, err := Load(f)
			if err != nil {
				t.Fatal(err)
			}
			if len(trips) == 0 {
				t.Fatal("expected to see non-zero trips, got 0")
			}
		})
	}
}

func BenchmarkLoadOfficial(b *testing.B) {
	paths := officialDatasets(b)
	if len(paths) == 0 {
		b.Skip("No FordGo CSVs in the testdata directory")
	}

	for _, tripdata := range paths {
		trippath := tripdata
		b.Run(filepath.Base(trippath), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				f, err := os.Open(trippath)
				if err != nil {
					b.Fatal(err)
				}
				inf, err := f.Stat()
				if err != nil {
					b.Fatal(err)
				}
				b.SetBytes(inf.Size())
				trips, err := Load(bufio.NewReader(f))
				if err != nil {
					b.Fatal(err)
				}
				if len(trips) == 0 {
					b.Fatal("expected to see non-zero trips, got 0")
				}
				if err := f.Close(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func TestTripDistance(t *testing.T) {
	s := `59989,"2018-07-31 18:20:32.7230","2018-08-01 11:00:22.1890",197,"El Embarcadero at Grand Ave",37.8088479,-122.2496799,181,"Grand Ave at Webster St",37.8113768,-122.2651925,1953,"Customer",1995,"Male","No"`
	record, err := csv.NewReader(strings.NewReader(s)).Read()
	if err != nil {
		panic(err)
	}
	trip, err := parseTrip(record)
	if err != nil {
		panic(err)
	}
	dist := trip.Distance()
	if dist < 0.8 || dist > 0.9 {
		t.Errorf("bad distance: %f", dist)
	}
}
