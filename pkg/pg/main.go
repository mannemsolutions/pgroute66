package pg

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"strings"
)

var (
	ctx context.Context
	log *zap.SugaredLogger
)

type Dsn map[string]string

func connectStringValue(objectName string) (escaped string) {
	return fmt.Sprintf("'%s'", strings.Replace(objectName, "'", "\\'", -1))
}

func InitContext(c context.Context) {
	ctx = c
}

func InitLogger(logger *zap.SugaredLogger) {
	log = logger
}
