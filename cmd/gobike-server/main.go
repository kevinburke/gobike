// server loads configuration from a file and starts a HTTP server
// that can render HTML templates and static assets.
//
// See config.yml for an explanation of the configuration options for the
// server, and the Makefile for various tasks you can run in coordination with
// the server (run tests, build assets, start the server).
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/kevinburke/gobike"
	"github.com/kevinburke/gobike/server"
	"github.com/kevinburke/handlers"
	"github.com/kevinburke/nacl"
	yaml "gopkg.in/yaml.v2"
)

// DefaultPort is the listening port if no other port is specified.
var DefaultPort = 8333

// The server's Version.
const Version = "0.7"

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

func main() {
	cfg := flag.String("config", "config.yml", "Path to a config file")
	version := flag.Bool("version", false, "Print the version string and exit")
	tripDirectory := flag.String("trip-directory", "data", "Directory holding trip data")
	start := time.Now()
	flag.Parse()
	if *version {
		fmt.Fprintf(os.Stderr, "gobike version %s\n", Version)
		os.Exit(0)
	}
	data, err := ioutil.ReadFile(*cfg)
	c := new(FileConfig)
	if err == nil {
		if err := yaml.Unmarshal(data, c); err != nil {
			server.Logger.Error("Couldn't parse config file", "err", err)
			os.Exit(2)
		}
	} else {
		server.Logger.Error("Couldn't find config file", "err", err)
		os.Exit(2)
	}
	var key nacl.Key
	if c.SecretKey == "" {
		server.Logger.Warn("No secret key specified, generating a random one")
		key = nacl.NewKey()
	} else {
		key, err = nacl.Load(c.SecretKey)
		if err != nil {
			server.Logger.Error("Error getting secret key", "err", err)
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
				server.Logger.Error("Invalid port", "err", err, "port", port)
				os.Exit(2)
			}
			c.Port = &iPort
		} else {
			c.Port = &DefaultPort
		}
	}
	trips, err := gobike.LoadDir(*tripDirectory)
	if err != nil {
		server.Logger.Error("Could not load trips", "err", err, "directory", *tripDirectory)
		os.Exit(2)
	}
	mux, err := server.NewServeMux(trips)
	if err != nil {
		server.Logger.Error("Could not initialize server", "err", err)
		os.Exit(2)
	}
	mux = handlers.UUID(mux)                             // add UUID header
	mux = handlers.Server(mux, "gobike-server/"+Version) // add Server header
	mux = handlers.Log(mux)                              // log requests/responses
	mux = handlers.Duration(mux)                         // add Duration header
	addr := ":" + strconv.Itoa(*c.Port)
	if c.HTTPOnly {
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			server.Logger.Error("Error listening", "addr", addr, "err", err)
			os.Exit(2)
		}
		server.Logger.Info("Started server", "time", time.Since(start).Round(100*time.Microsecond),
			"protocol", "http", "port", *c.Port)
		http.Serve(ln, mux)
	} else {
		mux = handlers.STS(mux) // set Strict-Transport-Security header
		if c.CertFile == "" {
			c.CertFile = "certs/leaf.pem"
		}
		if _, err := os.Stat(c.CertFile); os.IsNotExist(err) {
			server.Logger.Error("Could not find a cert file; generate using 'make generate_cert'", "file", c.CertFile)
			os.Exit(2)
		}
		if c.KeyFile == "" {
			c.KeyFile = "certs/leaf.key"
		}
		if _, err := os.Stat(c.KeyFile); os.IsNotExist(err) {
			server.Logger.Error("Could not find a key file; generate using 'make generate_cert'", "file", c.KeyFile)
			os.Exit(2)
		}
		server.Logger.Info("Starting server", "time", time.Since(start).Round(100*time.Microsecond), "protocol", "https", "port", *c.Port)
		listenErr := http.ListenAndServeTLS(addr, c.CertFile, c.KeyFile, mux)
		server.Logger.Error("server shut down", "err", listenErr)
	}
}
