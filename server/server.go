package server

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"regexp"
	"strings"
	"time"

	log "github.com/inconshreveable/log15"
	"github.com/kevinburke/gobike"
	"github.com/kevinburke/gobike/geo"
	"github.com/kevinburke/gobike/server/assets"
	"github.com/kevinburke/gobike/stats"
	"github.com/kevinburke/handlers"
	"github.com/kevinburke/rest"
)

var Logger log.Logger
var digests map[string][sha256.Size]byte

func b64(digest []byte) string {
	return strings.TrimRight(base64.URLEncoding.EncodeToString(digest), "=")
}

// hashurl returns a hash of the resource with the given key
func hashurl(key string) template.URL {
	d, ok := digests[strings.TrimPrefix(key, "/")]
	if !ok {
		return ""
	}
	// we don't actually need the whole hash.
	return template.URL("s=" + b64(d[:12]))
}

func init() {
	var err error
	digests, err = assets.Digests()
	if err != nil {
		panic(err)
	}
	homepageHTML := assets.MustAssetString("templates/index.html")
	homepageTpl = template.Must(
		template.New("homepage").Option("missingkey=error").Funcs(template.FuncMap{
			"hashurl": hashurl,
		}).Parse(homepageHTML),
	)
	Logger = handlers.Logger

	// Add more templates here.
}

// A HTTP server for static files. All assets are packaged up in the assets
// directory with the go-bindata binary. Run "make assets" to rerun the
// go-bindata binary.
type static struct {
	modTime time.Time
}

var expires = time.Date(2050, time.January, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC1123)

func (s *static) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/favicon.ico" {
		r.URL.Path = "/static/favicon.ico"
	}
	bits, err := assets.Asset(strings.TrimPrefix(r.URL.Path, "/"))
	if err != nil {
		rest.NotFound(w, r)
		return
	}
	// with the hashurl implementation below, we can set a super-long content
	// expiry and ensure content is never stale.
	if query := r.URL.Query(); query.Get("s") != "" {
		w.Header().Set("Expires", expires)
	}
	http.ServeContent(w, r, r.URL.Path, s.modTime, bytes.NewReader(bits))
}

// Render a template, or a server error.
func render(w http.ResponseWriter, r *http.Request, tpl *template.Template, name string, data interface{}) {
	buf := new(bytes.Buffer)
	if err := tpl.ExecuteTemplate(buf, name, data); err != nil {
		rest.ServerError(w, r, err)
		return
	}
	w.Write(buf.Bytes())
}

type homepageData struct {
	TripsPerWeek             template.JS
	TripsPerWeekCount        int64
	StationsPerWeek          template.JS
	StationsPerWeekCount     int64
	BikesPerWeek             template.JS
	BikesPerWeekCount        int64
	TripsPerBikePerWeek      template.JS
	TripsPerBikePerWeekCount string
}

var homepageTpl *template.Template

func renderCityPage(city *geo.City, allTrips []*gobike.Trip) (http.Handler, error) {
	trips := make([]*gobike.Trip, 0)
	for i := range allTrips {
		if city.ContainsPoint(allTrips[i].StartStationLatitude, allTrips[i].StartStationLongitude) {
			trips = append(trips, allTrips[i])
		}
	}
	if len(trips) == 0 {
		panic("no trips")
	}
	stationsPerWeek := stats.UniqueStationsPerWeek(trips)
	stationData, err := json.Marshal(stationsPerWeek)
	if err != nil {
		return nil, err
	}
	tripsPerWeek := stats.TripsPerWeek(trips)
	data, err := json.Marshal(tripsPerWeek)
	if err != nil {
		return nil, err
	}
	bikeTripsPerWeek := stats.UniqueBikesPerWeek(trips)
	bikeData, err := json.Marshal(bikeTripsPerWeek)
	if err != nil {
		return nil, err
	}
	tripsPerBikePerWeek := stats.TripsPerBikePerWeek(trips)
	tripPerBikeData, err := json.Marshal(tripsPerBikePerWeek)
	if err != nil {
		return nil, err
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		render(w, r, homepageTpl, "homepage", &homepageData{
			TripsPerWeek:             template.JS(string(data)),
			TripsPerWeekCount:        int64(tripsPerWeek[len(tripsPerWeek)-1].Data),
			StationsPerWeek:          template.JS(string(stationData)),
			StationsPerWeekCount:     int64(stationsPerWeek[len(stationsPerWeek)-1].Data),
			BikesPerWeek:             template.JS(string(bikeData)),
			BikesPerWeekCount:        int64(bikeTripsPerWeek[len(bikeTripsPerWeek)-1].Data),
			TripsPerBikePerWeek:      template.JS(string(tripPerBikeData)),
			TripsPerBikePerWeekCount: fmt.Sprintf("%.1f", tripsPerBikePerWeek[len(tripsPerBikePerWeek)-1].Data),
		})
	}), nil
}

// NewServeMux returns a HTTP handler that covers all routes known to the
// server.
func NewServeMux(trips []*gobike.Trip) (http.Handler, error) {
	staticServer := &static{
		modTime: time.Now().UTC(),
	}

	r := new(handlers.Regexp)
	r.Handle(regexp.MustCompile(`(^/static|^/favicon.ico$)`), []string{"GET"}, handlers.GZip(staticServer))
	homepageHandler, err := renderCityPage(geo.US, trips)
	if err != nil {
		return nil, err
	}
	sfHandler, err := renderCityPage(geo.SF, trips)
	if err != nil {
		return nil, err
	}
	oakHandler, err := renderCityPage(geo.Oakland, trips)
	if err != nil {
		return nil, err
	}
	sjHandler, err := renderCityPage(geo.SanJose, trips)
	if err != nil {
		return nil, err
	}
	r.Handle(regexp.MustCompile(`^/$`), []string{"GET"}, homepageHandler)
	r.Handle(regexp.MustCompile(`^/sf$`), []string{"GET"}, sfHandler)
	r.Handle(regexp.MustCompile(`^/oakland$`), []string{"GET"}, oakHandler)
	r.Handle(regexp.MustCompile(`^/sj$`), []string{"GET"}, sjHandler)
	// Add more routes here. Routes not matched will get a 404 error page.
	// Call rest.RegisterHandler(404, http.HandlerFunc) to provide your own 404
	// page instead of the default.
	return r, nil
}
