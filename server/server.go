package server

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"html/template"
	"net/http"
	"regexp"
	"strings"
	"time"

	log "github.com/inconshreveable/log15"
	"github.com/kevinburke/gobike"
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
	TripsPerWeek         template.JS
	TripsPerWeekCount    int64
	StationsPerWeek      template.JS
	StationsPerWeekCount int64
}

var homepageTpl *template.Template

// NewServeMux returns a HTTP handler that covers all routes known to the
// server.
func NewServeMux(trips []*gobike.Trip) http.Handler {
	staticServer := &static{
		modTime: time.Now().UTC(),
	}

	r := new(handlers.Regexp)
	r.Handle(regexp.MustCompile(`(^/static|^/favicon.ico$)`), []string{"GET"}, handlers.GZip(staticServer))
	r.HandleFunc(regexp.MustCompile(`^/$`), []string{"GET"}, func(w http.ResponseWriter, r *http.Request) {
		stationsPerWeek := stats.UniqueStationsPerWeek(trips)
		stationData, err := json.Marshal(stationsPerWeek)
		if err != nil {
			rest.ServerError(w, r, err)
			return
		}
		tripsPerWeek := stats.TripsPerWeek(trips)
		data, err := json.Marshal(tripsPerWeek)
		if err != nil {
			rest.ServerError(w, r, err)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		render(w, r, homepageTpl, "homepage", &homepageData{
			TripsPerWeek:         template.JS(string(data)),
			TripsPerWeekCount:    int64(tripsPerWeek[len(tripsPerWeek)-1].Data),
			StationsPerWeek:      template.JS(string(stationData)),
			StationsPerWeekCount: int64(stationsPerWeek[len(stationsPerWeek)-1].Data),
		})
	})
	// Add more routes here. Routes not matched will get a 404 error page.
	// Call rest.RegisterHandler(404, http.HandlerFunc) to provide your own 404
	// page instead of the default.
	return r
}
