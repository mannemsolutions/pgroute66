package internal

import (
	"flag"
	"fmt"
	"github.com/mannemsolutions/pgroute66/pkg/pg"
	"go.uber.org/zap/zapcore"
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

/*
 * This module reads the config file and returns a config object with all entries from the config yaml file.
 */

const (
	envConfName     = "PGROUTE66CONFIG"
	defaultConfFile = "/etc/pgroute66/config.yaml"
)

type RouteHostsConfig map[string]pg.Dsn

type RouteSSLConfig struct {
	Cert string `yaml:"cert"`
	Key  string `yaml:"key"`
}

type RouteConfig struct {
	Hosts RouteHostsConfig `yaml:"hosts"`
	Bind string `yaml:"bind"`
	Port int `yaml:"port"`
	Ssl RouteSSLConfig `yaml:"ssl"`
	LogLevel zapcore.Level `yaml:"loglevel"`
	Verbosity string `yaml:"verbosity"`
}

func NewConfig() (config RouteConfig, err error) {
	var configFile string
	var debug bool
	var version bool
	flag.BoolVar(&debug, "d", false, "Add debugging output")
	flag.BoolVar(&version, "v", false, "Show version information")
	flag.StringVar(&configFile, "c", os.Getenv(envConfName), "Path to configfile")

	flag.Parse()
	if version {
		fmt.Println(appVersion)
		os.Exit(0)
	}
	if configFile == "" {
		configFile = defaultConfFile
	}
	configFile, err = filepath.EvalSymlinks(configFile)
	if err != nil {
		return config, err
	}

	// This only parsed as yaml, nothing else
	// #nosec
	yamlConfig, err := ioutil.ReadFile(configFile)
	if err != nil {
		return config, err
	}
	err = yaml.Unmarshal(yamlConfig, &config)
	if debug {
		config.LogLevel = zapcore.DebugLevel
	}
	return config, err
}

func (rc RouteConfig) BindTo() string {
	if rc.Bind == "" {
		return fmt.Sprintf("localhost:%d", rc.Port)
	}
	return fmt.Sprintf("%s:%d", rc.Bind, rc.Port)
}
