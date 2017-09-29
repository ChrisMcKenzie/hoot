package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"

	"github.com/chrismckenzie/hoot/chat"
	"github.com/chrismckenzie/hoot/server"
)

type Config struct {
	Port    string `json:"port"`
	LogPath string `json:"logpath"`
}

const (
	DefaultConfigPath = "./defaultConfig.json"
	DefaultLogPath    = "./hoot.log"
	DefaultPort       = ":9000"
)

var (
	configPath    = flag.String("config", "", "path to config file")
	defaultConfig = &Config{DefaultPort, DefaultLogPath}
)

func main() {
	flag.Parse()

	c, err := readConfig(*configPath)
	if err != nil {
		log.Fatalf("unable to read config file: %s", err)
	}

	logFile, err := os.OpenFile(c.LogPath,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("unable to open log file: %s", err)
	}
	defer logFile.Close()

	// w := io.MultiWriter(os.Stdout, logFile)
	logger := log.New(logFile, "", log.LstdFlags)

	rm := chat.NewRoomManager(logger)

	srv := server.NewHootServer(c.Port, rm)
	log.Fatal(srv.ListenAndServe())
}

func readConfig(path string) (*Config, error) {
	c := defaultConfig
	// if no path then return the default
	if path == "" {
		return c, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(f)
	if err := dec.Decode(&c); err != nil {
		return nil, err
	}

	return c, nil
}
