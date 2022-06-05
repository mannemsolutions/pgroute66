package internal

import (
	"encoding/base64"
	"log"
	"os"
	"sort"

	"github.com/mannemsolutions/pgroute66/pkg/pg"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type PgRouteHandler struct {
	connections map[string]*pg.Conn
	logger      *zap.SugaredLogger
	atom        zap.AtomicLevel
	config      RouteConfig
}

/*
With Gin, there is no winning with gochecknoglobals.
This seems like a proper way to go.
Also see https://github.com/gothinkster/golang-gin-realworld-example-app/issues/15 for background
*/
//nolint
var globalHandler *PgRouteHandler

func Initialize() {
	if globalHandler == nil {
		globalHandler = NewPgRouteHandler()
	}
}

func NewPgRouteHandler() *PgRouteHandler {
	var err error

	prh := PgRouteHandler{
		connections: make(map[string]*pg.Conn),
	}

	prh.atom = zap.NewAtomicLevel()
	encoderCfg := zap.NewDevelopmentEncoderConfig()
	encoderCfg.EncodeTime = zapcore.RFC3339TimeEncoder
	prh.logger = zap.New(zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderCfg),
		zapcore.Lock(os.Stdout),
		prh.atom,
	)).Sugar()

	prh.config, err = NewConfig()
	if err != nil {
		prh.logger.Fatal("Cannot parse config", err)
	}

	if prh.config.Debug() {
		prh.atom.SetLevel(zapcore.DebugLevel)
	}

	for name, dsn := range prh.config.Hosts {
		if b64password, exists := dsn["b64password"]; exists {
			sDec, err := base64.StdEncoding.DecodeString(b64password)
			if err != nil {
				log.Panicf("Could not decode b64password %s, %s", b64password, err.Error())
			}

			dsn["password"] = string(sDec)

			delete(dsn, "b64password")
		}

		prh.connections[name] = pg.NewConn(dsn, prh.logger)
	}

	return &prh
}

func (prh PgRouteHandler) GetStandbys() (standbys []string) {
	for name, conn := range prh.connections {
		isStandby, err := conn.IsStandby()
		if err != nil {
			prh.logger.Debugf("Could not get state of standby %s, %s", name, err.Error())
		}

		if isStandby {
			standbys = append(standbys, name)
		}
	}

	sort.Strings(standbys)

	return standbys
}

func (prh PgRouteHandler) GetPrimaries() (primaries []string) {
	for name, conn := range prh.connections {
		isPrimary, err := conn.IsPrimary()
		if err != nil {
			prh.logger.Debugf("Could not get state of primary %s, %s", name, err.Error())
		}

		if isPrimary {
			primaries = append(primaries, name)
		}
	}

	sort.Strings(primaries)

	return primaries
}

func (prh PgRouteHandler) GetNodeStatus(name string) string {
	if node, exists := prh.connections[name]; exists {
		isPrimary, err := node.IsPrimary()
		if err != nil {
			prh.logger.Debugf("Could not get state of node %s, %s", name, err.Error())

			return "unavailable"
		} else if isPrimary {
			return "primary"
		} else {
			return "standby"
		}
	}

	return "invalid"
}
