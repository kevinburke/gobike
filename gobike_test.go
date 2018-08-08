package gobike

import (
	"bufio"
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	f, err := os.Open("data/201806-fordgobike-tripdata.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	trips, err := Load(bufio.NewReader(f))
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
		f, err := os.Open("data/201806-fordgobike-tripdata.csv")
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
