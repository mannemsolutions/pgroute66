package internal

import (
	"encoding/base64"
	"github.com/mannemsolutions/pgroute66/pkg/pg"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

var (
	log  *zap.SugaredLogger
	atom zap.AtomicLevel
	handler *PgRouteHandler
	config RouteConfig
)

func Initialize() {
	var err error
	atom = zap.NewAtomicLevel()
	encoderCfg := zap.NewDevelopmentEncoderConfig()
	encoderCfg.EncodeTime = zapcore.RFC3339TimeEncoder
	log = zap.New(zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderCfg),
		zapcore.Lock(os.Stdout),
		atom,
	)).Sugar()

	pg.Initialize(log)
	handler, err = NewPgRouteHandler()
	if err != nil {
		log.Panic("Failed to initialize handler, ", err)
	}
}

type PgRouteHandler map[string]pg.Conn

func NewPgRouteHandler() (*PgRouteHandler, error) {
	var err error
	config, err = NewConfig()
	if err != nil {
		return nil, err
	}

	atom.SetLevel(config.LogLevel)

	prh := PgRouteHandler{}
	for name, dsn := range config.Hosts {
		if b64password, exists := dsn["b64password"]; exists {
			sDec, err := base64.StdEncoding.DecodeString(b64password)
			if err != nil {
				log.Panicf("Could not decode b64password %s, %s", b64password, err.Error())
			}
			dsn["password"] = string(sDec)
			delete(dsn, "b64password")
		}
		prh[name] = *(pg.NewConn(dsn))
	}

	return &prh, nil
}

func (prh PgRouteHandler) GetStandbys() (standbys []string) {
	for name, conn := range prh {
		isStandby, err := conn.IsStandby()
		if err != nil {
			log.Debugf("Could not get state of standby %s, %s", name, err.Error())
		}
		if isStandby {
			standbys = append(standbys, name)
		}
	}
	return standbys
}

func (prh PgRouteHandler) GetPrimaries() (primaries []string) {
	for name, conn := range prh {
		isPrimary, err := conn.IsPrimary()
		if err != nil {
			log.Debugf("Could not get state of primary %s, %s", name, err.Error())
		}
		if isPrimary {
			primaries = append(primaries, name)
		}
	}
	return primaries
}

func (prh PgRouteHandler) GetNodeStatus(name string) (string) {
	if node, exists := prh[name]; exists {
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