package internal

import (
	"encoding/base64"
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
	Cert string `yaml:"b64cert"`
	Key  string `yaml:"b64key"`
}

func (rsc RouteSSLConfig) Enabled() bool {
	if rsc.Cert != "" && rsc.Key != "" {
		return true
	}
	return false
}

func (rsc RouteSSLConfig) KeyBytes() ([]byte, error) {
	if ! rsc.Enabled() {
		return nil, fmt.Errorf("cannot get CertBytes when SSL is not enabled")
	}
	return base64.StdEncoding.DecodeString(rsc.Key)
}

func (rsc RouteSSLConfig) MustKeyBytes() []byte {
	kb, err := rsc.KeyBytes()
	if err != nil {
		log.Fatal("could not decrypt SSL key", err)
	}
	return kb
}

func (rsc RouteSSLConfig) CertBytes() ([]byte, error) {
	if ! rsc.Enabled() {
		return nil, fmt.Errorf("cannot get CertBytes when SSL is not enabled")
	}
	return base64.StdEncoding.DecodeString(rsc.Cert)
}

func (rsc RouteSSLConfig) MustCertBytes() []byte {
	cb, err := rsc.CertBytes()
	if err != nil {
		log.Fatal("could not decrypt SSL Cert", err)
	}
	return cb
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
	port := rc.Port
	if port == 0 {
		if rc.Ssl.Enabled() {
			port = 8443
		} else {
			port = 8080
		}
	}
	if rc.Bind == "" {
		return fmt.Sprintf("localhost:%d", port)
	}
	return fmt.Sprintf("%s:%d", rc.Bind, port)
}
