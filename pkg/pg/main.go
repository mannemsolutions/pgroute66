package pg

import (
	"context"
	"go.uber.org/zap"
)

var (
	ctx context.Context
	log *zap.SugaredLogger
)

type Dsn map[string]string

func InitContext(c context.Context) {
	ctx = c
}

func InitLogger(logger *zap.SugaredLogger) {
	log = logger
}
