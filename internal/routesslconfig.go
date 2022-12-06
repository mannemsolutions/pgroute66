package internal

import (
	"encoding/base64"
	"fmt"
	"log"
)

/*
 * This module reads the config file and returns a config object with all entries from the config yaml file.
 */

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
	if !rsc.Enabled() {
		return nil, fmt.Errorf("cannot get CertBytes when SSL is not enabled")
	}

	return base64.StdEncoding.DecodeString(rsc.Key)
}

func (rsc RouteSSLConfig) MustKeyBytes() []byte {
	kb, err := rsc.KeyBytes()
	if err != nil {
		globalHandler.log.Fatal("could not decrypt SSL key", err)
	}

	return kb
}

func (rsc RouteSSLConfig) CertBytes() ([]byte, error) {
	if !rsc.Enabled() {
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
