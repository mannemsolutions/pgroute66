package internal

import (
	"context"
	"encoding/base64"
	"fmt"
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

func (prh PgRouteHandler) UpdateNodeAvailability() {
	for nodeName, conn := range prh.connections {
		if isPrimary, err := conn.IsPrimary(); err != nil {
			log.Errorf("failed to check if node %s is primary: %e", nodeName, err)
		} else if !isPrimary {
			continue
		} else if err = conn.AvUpdateDuration(); err != nil {
			log.Errorf("failed to update availability info on node %s: %e", nodeName, err)
			return
		} else {
			log.Infof("updating availability info on node %s", nodeName)
			return
		}
	}
}

func (prh PgRouteHandler) CreateAvailabilityTable() {
	for nodeName, conn := range prh.connections {
		if isPrimary, err := conn.IsPrimary(); err != nil {
			log.Errorf("failed to check if node %s is primary: %e", nodeName, err)
		} else if !isPrimary {
			continue
		} else if err = conn.AvcCreateTable(); err != nil {
			log.Errorf("failed to create availability table on node %s: %e", nodeName, err)
			return
		} else {
			log.Infof("creating availability table on node %s", nodeName)
			return
		}
	}
}

func (prh PgRouteHandler) GetNodeAvailability(name string, limit float64) string {
	prh.CreateAvailabilityTable()
	defer prh.UpdateNodeAvailability()
	if node, exists := prh.connections[name]; exists {
		if err := node.AvCheckDuration(limit); err == nil {
			log.Infof("availability of node %s is within limits", name)
			return "ok"
		} else if aErr, ok := err.(pg.AvcDurationExceededError); !ok {
			log.Errorf("unexpeced error occurred while retrieving availability of %s: %e", name, err)
			return err.Error()
		} else {
			log.Infof("Availability limit exceeded for %s: %e", name, aErr)
			return fmt.Sprintf("exceeded (%s)", aErr.String())
		}
	}

	return "invalid"
}
