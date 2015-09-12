package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"

	"gopkg.in/yaml.v1"
)

const defaultConfig = `
listen:
`

type TcpfwdConfig struct {
	Listen map[string]ListenConfig `yaml:"listen"`
}

type ListenConfig struct {
	Local  string `yaml:"local"`
	Remote string `yaml:"remote"`
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
	// TODO pending implementation:
	// - graphite support
	// populate Netmasks
	// cfg.parseNetworks()
	// if len(cfg.Graphite.Prefix) < 1 {
	// 	cfg.Graphite.Prefix = "tcpfwd"
	// }
	// if cfg.Graphite.Interval < 1 {
	// 	cfg.Graphite.Interval = 60
	// }

	return cfg
}
func main() {
	config := loadConfiguration("conf/tcpfwd.yaml")

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
