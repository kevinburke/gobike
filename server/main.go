// server loads configuration from a file and starts a HTTP server
// that can render HTML templates and static assets.
//
// See config.yml for an explanation of the configuration options for the
// server, and the Makefile for various tasks you can run in coordination with
// the server (run tests, build assets, start the server).
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	log "github.com/inconshreveable/log15"
	"github.com/kevinburke/gobike/assets"
	"github.com/kevinburke/handlers"
	"github.com/kevinburke/nacl"
	"github.com/kevinburke/rest"
	yaml "gopkg.in/yaml.v2"
)

// DefaultPort is the listening port if no other port is specified.
var DefaultPort = 7065

// The server's Version.
const Version = "0.7"

var homepageTpl *template.Template
var logger log.Logger
var digests map[string][sha256.Size]byte

// hashurl returns a hash of the resource with the given key
func hashurl(key string) template.URL {
	d, ok := digests[strings.TrimPrefix(key, "/")]
	if !ok {
		return ""
	}
	// we don't actually need the whole hash.
	return template.URL("s=" + b64(d[:12]))
}

func b64(digest []byte) string {
	return strings.TrimRight(base64.URLEncoding.EncodeToString(digest), "=")
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
	logger = handlers.Logger

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

// NewServeMux returns a HTTP handler that covers all routes known to the
// server.
func NewServeMux() http.Handler {
	staticServer := &static{
		modTime: time.Now().UTC(),
	}

	r := new(handlers.Regexp)
	r.Handle(regexp.MustCompile(`(^/static|^/favicon.ico$)`), []string{"GET"}, handlers.GZip(staticServer))
	r.HandleFunc(regexp.MustCompile(`^/$`), []string{"GET"}, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		render(w, r, homepageTpl, "homepage", nil)
	})
	// Add more routes here. Routes not matched will get a 404 error page.
	// Call rest.RegisterHandler(404, http.HandlerFunc) to provide your own 404
	// page instead of the default.
	return r
}

// FileConfig represents the data in a config file.
type FileConfig struct {
	// SecretKey is used to encrypt sessions and other data before serving it to
	// the client. It should be a hex string that's exactly 64 bytes long. For
	// example:
	//
	//   d7211b215341871968869dontusethisc0ff1789fc88e0ac6e296ba36703edf8
	//
	// That key is invalid - you can generate a random key by running:
	//
	//   openssl rand -hex 32
	//
	// If no secret key is present, we'll generate one when the server starts.
	// However, this means that sessions may error when the server restarts.
	//
	// If a server key is present, but invalid, the server will not start.
	SecretKey string `yaml:"secret_key"`

	// Port to listen on. Set to 0 to choose a port at random. If unspecified,
	// defaults to 7065.
	Port *int `yaml:"port"`

	// Set to true to listen for HTTP traffic (instead of TLS traffic). Note
	// you need to terminate TLS to use HTTP server push.
	HTTPOnly bool `yaml:"http_only"`

	// For TLS configuration.
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`

	// Add other configuration settings here.
}

var cfg = flag.String("config", "config.yml", "Path to a config file")
var version = flag.Bool("version", false, "Print the version string and exit")

func main() {
	start := time.Now()
	flag.Parse()
	if *version {
		fmt.Fprintf(os.Stderr, "go-html-boilerplate version %s\n", Version)
		os.Exit(0)
	}
	data, err := ioutil.ReadFile(*cfg)
	c := new(FileConfig)
	if err == nil {
		if err := yaml.Unmarshal(data, c); err != nil {
			logger.Error("Couldn't parse config file", "err", err)
			os.Exit(2)
		}
	} else {
		logger.Error("Couldn't find config file", "err", err)
		os.Exit(2)
	}
	var key nacl.Key
	if c.SecretKey == "" {
		logger.Warn("No secret key specified, generating a random one")
		key = nacl.NewKey()
	} else {
		key, err = nacl.Load(c.SecretKey)
		if err != nil {
			logger.Error("Error getting secret key", "err", err)
			os.Exit(2)
		}
	}
	// You can use the secret key with secretbox
	// (godoc.org/github.com/kevinburke/nacl/secretbox/) to generate cookies and
	// secrets. See flash.go and crypto.go for examples.
	_ = key

	if c.Port == nil {
		port, ok := os.LookupEnv("PORT")
		if ok {
			iPort, err := strconv.Atoi(port)
			if err != nil {
				logger.Error("Invalid port", "err", err, "port", port)
				os.Exit(2)
			}
			c.Port = &iPort
		} else {
			c.Port = &DefaultPort
		}
	}
	mux := NewServeMux()
	mux = handlers.UUID(mux)                                   // add UUID header
	mux = handlers.Server(mux, "go-html-boilerplate/"+Version) // add Server header
	mux = handlers.Log(mux)                                    // log requests/responses
	mux = handlers.Duration(mux)                               // add Duration header
	addr := ":" + strconv.Itoa(*c.Port)
	if c.HTTPOnly {
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			logger.Error("Error listening", "addr", addr, "err", err)
			os.Exit(2)
		}
		logger.Info("Started server", "time", time.Since(start).Round(100*time.Microsecond),
			"protocol", "http", "port", *c.Port)
		http.Serve(ln, mux)
	} else {
		mux = handlers.STS(mux) // set Strict-Transport-Security header
		if c.CertFile == "" {
			c.CertFile = "certs/leaf.pem"
		}
		if _, err := os.Stat(c.CertFile); os.IsNotExist(err) {
			logger.Error("Could not find a cert file; generate using 'make generate_cert'", "file", c.CertFile)
			os.Exit(2)
		}
		if c.KeyFile == "" {
			c.KeyFile = "certs/leaf.key"
		}
		if _, err := os.Stat(c.KeyFile); os.IsNotExist(err) {
			logger.Error("Could not find a key file; generate using 'make generate_cert'", "file", c.KeyFile)
			os.Exit(2)
		}
		logger.Info("Starting server", "time", time.Since(start).Round(100*time.Microsecond), "protocol", "https", "port", *c.Port)
		listenErr := http.ListenAndServeTLS(addr, c.CertFile, c.KeyFile, mux)
		logger.Error("server shut down", "err", listenErr)
	}
}
