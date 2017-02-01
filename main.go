package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"jus.tw.cx/jw-business-api/lib/logger"

	"github.com/ghodss/yaml"
	"github.com/prometheus/client_golang/prometheus"
)

const defaultConfig = `
listen:
`

type TcpfwdConfig struct {
	Metrics string                  `json:"metrics"`
	Listen  map[string]ListenConfig `json:"listen"`
}

type ListenConfig struct {
	Local  string `json:"local"`
	Remote string `json:"remote"`
}

func loadConfiguration(cfgFile string) TcpfwdConfig {
	var err error
	var buf []byte

	if _, err := os.Stat(cfgFile); err == nil {
		log.Println("Loading config from ", cfgFile)
		buf, err = ioutil.ReadFile(cfgFile)
		if err != nil {
			log.Println("Could not read config from ", cfgFile)
			buf = []byte(defaultConfig)
		}
	} else {
		log.Println("Loading default config, due to error ", err)
		buf = []byte(defaultConfig)
	}

	var cfg TcpfwdConfig
	err = yaml.Unmarshal(buf, &cfg)
	if err != nil {
		log.Panic("Could not load config file", err)
	}

	// WONT implement:
	// - allow/denty filter -> use iptables
	// - rate limiting -> use iptables

	return cfg
}

var (
	Conns *prometheus.CounterVec
	Bytes *prometheus.CounterVec
)

func init() {
	Conns = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tcp_connections_total",
			Help: "Number of TCP Connections established",
		},
		[]string{"name"},
	)
	prometheus.Register(Conns)
	Bytes = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tcp_bytes_total",
			Help: "Number of bytes transfered",
		},
		[]string{"name", "direction"},
	)
	prometheus.Register(Bytes)
}

func main() {
	config := loadConfiguration("conf/tcpfwd.yaml")

	go func() {
		http.Handle("/metrics", prometheus.Handler())
		http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "OK", http.StatusOK)
		})
		http.HandleFunc("/", http.NotFound)
		if err := http.ListenAndServe(config.Metrics, nil); err != nil {
			logger.Log("level", "error", "msg", "Failed to listen on management port", "err", err)
		}
	}()

	for k, v := range config.Listen {
		go tryListen(k, v.Local, v.Remote, true)
	}

	log.Printf("Started all listeners")

	exitChan := make(chan os.Signal, 10)
	signal.Notify(exitChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)

	<-exitChan
	log.Printf("Exiting due to signal")
	os.Exit(0)
}
