package internal

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/mannemsolutions/pgroute66/pkg/pg"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	GHStatusInvalid     = "invalid"
	GHStatusOk          = "ok"
	GHStatusPrimary     = "primary"
	GHStatusStandby     = "standby"
	GHStatusUnavailable = "unavailable"
)

const (
	pgrOpenMode   = os.O_APPEND | os.O_CREATE | os.O_WRONLY
	pgrCreateMode = 0o644
)

type PgRouteHandler struct {
	log         *zap.SugaredLogger
	atom        zap.AtomicLevel
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
}

func NewPgRouteHandler() *PgRouteHandler {
	var err error

	prh := PgRouteHandler{
		connections: make(map[string]*pg.Conn),
	}

	prh.config, err = NewConfig()
	if err != nil {
		prh.initLogger("")
		prh.log.Fatal("Cannot parse config", err)
	}

	prh.initLogger(prh.config.LogFile)
	prh.enableDebug(prh.config.Debug())

	for name, dsn := range prh.config.Hosts {
		if b64password, exists := dsn["b64password"]; exists {
			sDec, err := base64.StdEncoding.DecodeString(b64password)
			if err != nil {
				prh.log.Panicf("Could not decode b64password %s, %s", b64password, err.Error())
			}

			dsn["password"] = string(sDec)

			delete(dsn, "b64password")
		}

		prh.connections[name] = pg.NewConn(dsn, prh.log)
	}

	return &prh
}

func (prh PgRouteHandler) GetStandbys(group string) (standbys []string) {
	for name, conn := range prh.connections.FilteredConnections(prh.config.GroupHosts(group)) {
		isStandby, err := conn.IsStandby(context.Background())
		if err != nil {
			prh.log.Debugf("Could not get state of standby %s, %s", name, err.Error())
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
		isPrimary, err := conn.IsPrimary(context.Background())
		if err != nil {
			prh.log.Debugf("Could not get state of primary %s, %s", name, err.Error())
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
		isPrimary, err := node.IsPrimary(context.Background())
		if err != nil {
			prh.log.Debugf("Could not get state of node %s, %s", name, err.Error())

			return GHStatusUnavailable
		} else if isPrimary {
			return GHStatusPrimary
		} else {
			return GHStatusStandby
		}
	}

	return GHStatusInvalid
}

func (prh PgRouteHandler) UpdateNodeAvailability() {
	for nodeName, conn := range prh.connections {
		if isPrimary, err := conn.IsPrimary(context.Background()); err != nil {
			prh.log.Errorf("failed to check if node %s is primary: %e", nodeName, err)
		} else if !isPrimary {
			continue
		} else if err = conn.AvUpdateDuration(context.Background()); err != nil {
			prh.log.Errorf("failed to update availability info on node %s: %e", nodeName, err)

			return
		} else {
			prh.log.Infof("updating availability info on node %s", nodeName)

			return
		}
	}
}

func (prh PgRouteHandler) CreateAvailabilityTable() {
	for nodeName, conn := range prh.connections {
		if isPrimary, err := conn.IsPrimary(context.Background()); err != nil {
			prh.log.Errorf("failed to check if node %s is primary: %e", nodeName, err)
		} else if !isPrimary {
			continue
		} else if err = conn.AvcCreateTable(context.Background()); err != nil {
			prh.log.Errorf("failed to create availability table on node %s: %e", nodeName, err)

			return
		} else {
			prh.log.Infof("creating availability table on node %s", nodeName)

			return
		}
	}
}

func (prh PgRouteHandler) GetNodeAvailability(name string, limit float64) string {
	prh.CreateAvailabilityTable()
	defer prh.UpdateNodeAvailability()

	if node, exists := prh.connections[name]; exists {
		if err := node.AvCheckDuration(context.Background(), limit); err == nil {
			prh.log.Infof("availability of node %s is within limits", name)

			return GHStatusOk
		} else if aErr, ok := err.(pg.AvcDurationExceededError); !ok {
			prh.log.Errorf("unexpeced error occurred while retrieving availability of %s: %e", name, err)

			return err.Error()
		} else {
			prh.log.Infof("Availability limit exceeded for %s: %e", name, aErr)

			return fmt.Sprintf("exceeded (%s)", aErr.String())
		}
	}

	return GHStatusInvalid
}

func (prh *PgRouteHandler) initLogger(logFilePath string) {
	prh.atom = zap.NewAtomicLevel()
	// First, define our level-handling logic.
	highPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zapcore.ErrorLevel
	})
	lowPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl < zapcore.ErrorLevel && lvl >= prh.atom.Level()
	})

	// High-priority output should also go to standard error, and low-priority
	// output should also go to standard out.
	consoleDebugging := zapcore.Lock(os.Stdout)
	consoleErrors := zapcore.Lock(os.Stderr)

	// Optimize the Kafka output for machine consumption and the console output
	// for human operators.
	// encoderCfg := zap.NewDevelopmentEncoderConfig()
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.EncodeTime = zapcore.RFC3339TimeEncoder
	consoleEncoder := zapcore.NewConsoleEncoder(encoderCfg)

	// Join the outputs, encoders, and level-handling functions into zapcore.Cores, then tee the cores together.
	var core zapcore.Core

	if logFilePath != "" {
		fileEncoder := zapcore.NewConsoleEncoder(encoderCfg)

		if logFile, err := os.OpenFile(filepath.Clean(logFilePath), pgrOpenMode, pgrCreateMode); err != nil {
			prh.initLogger("")
			prh.log.Panicf("error while opening logfile: %s", err)
		} else {
			writer := zapcore.AddSync(logFile)
			core = zapcore.NewTee(
				zapcore.NewCore(fileEncoder, writer, prh.atom),
				zapcore.NewCore(consoleEncoder, consoleErrors, highPriority),
				zapcore.NewCore(consoleEncoder, consoleDebugging, lowPriority),
			)
		}
	} else {
		core = zapcore.NewTee(
			zapcore.NewCore(consoleEncoder, consoleErrors, highPriority),
			zapcore.NewCore(consoleEncoder, consoleDebugging, lowPriority),
		)
	}

	prh.log = zap.New(core).Sugar()
}

func (prh *PgRouteHandler) enableDebug(debug bool) {
	if debug {
		prh.atom.SetLevel(zap.DebugLevel)
	}

	prh.log.Debug("Debug logging enabled")
}
