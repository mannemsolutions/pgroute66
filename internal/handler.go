package internal

import (
	"context"
	"encoding/base64"
	"sort"

	"github.com/mannemsolutions/pgroute66/pkg/pg"
	"go.uber.org/zap/zapcore"
)

type PgRouteHandler struct {
	connections RouteConnections
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
	pg.InitContext(context.Background())
}

func NewPgRouteHandler() *PgRouteHandler {
	var err error

	prh := PgRouteHandler{
		connections: make(map[string]*pg.Conn),
	}

	prh.config, err = NewConfig()
	if err != nil {
		log.Fatal("Cannot parse config", err)
	}

	initLogger(prh.config.LogFile)
	if prh.config.Debug() {
		atom.SetLevel(zapcore.DebugLevel)
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

		prh.connections[name] = pg.NewConn(dsn, log)
	}

	return &prh
}

func (prh PgRouteHandler) groupConnections(group string) RouteConnections {
	return prh.connections.FilteredConnections(prh.config.GroupHosts(group))
}

func (prh PgRouteHandler) GetStandbys(group string) (standbys []string) {
	for name, conn := range prh.connections.FilteredConnections(prh.config.GroupHosts(group)) {
		isStandby, err := conn.IsStandby()
		if err != nil {
			log.Debugf("Could not get state of standby %s, %s", name, err.Error())
		}

		if isStandby {
			standbys = append(standbys, name)
		}
	}

	sort.Strings(standbys)

	return standbys
}

func (prh PgRouteHandler) GetPrimaries(group string) (primaries []string) {
	for name, conn := range prh.connections.FilteredConnections(prh.config.GroupHosts(group)) {
		isPrimary, err := conn.IsPrimary()
		if err != nil {
			log.Debugf("Could not get state of primary %s, %s", name, err.Error())
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
			log.Debugf("Could not get state of node %s, %s", name, err.Error())

			return "unavailable"
		} else if isPrimary {
			return "primary"
		} else {
			return "standby"
		}
	}

	return "invalid"
}
