package pg

import (
	"fmt"
	"github.com/jackc/pgconn"
	"os"
	"strings"

	"github.com/jackc/pgx"
	"github.com/jackc/pgx/v4/pgxpool"
	"go.uber.org/zap"
)

type Conn struct {
	connParams Dsn
	endpoint   string
	conn       *pgxpool.Pool
	logger     *zap.SugaredLogger
}

func NewConn(connParams Dsn, logger *zap.SugaredLogger) (c *Conn) {
	c = &Conn{
		connParams: connParams,
		logger:     logger,
	}
	c.endpoint = fmt.Sprintf("%s:%s", c.Host(), c.Port())

	return c
}

func (c *Conn) DSN() (dsn string) {
	pairs := make([]string, 0, len(c.connParams))
	for key, value := range c.connParams {
		pairs = append(pairs, fmt.Sprintf("%s=%s", key, connectStringValue(value)))
	}

	return strings.Join(pairs[:], " ")
}

func (c *Conn) Host() string {
	value, ok := c.connParams["host"]
	if ok {
		return value
	}

	value = os.Getenv("PGHOST")
	if value != "" {
		return value
	}

	return "localhost"
}

func (c *Conn) Port() string {
	value, ok := c.connParams["port"]
	if ok {
		return value
	}

	value = os.Getenv("PGPORT")
	if value != "" {
		return value
	}

	return "5432"
}

func (c *Conn) Connect() (err error) {
	if c.conn != nil {
		return
	}

	c.logger.Debugf("Connecting to %s (%v)", c.endpoint, c.DSN())

	poolConfig, err := pgxpool.ParseConfig(c.DSN())
	if err != nil {
		log.Panicf("Unable to parse DSN (%s): %e", c.DSN(), err)
	}

	c.conn, err = pgxpool.ConnectConfig(ctx, poolConfig)
	if err != nil {
		c.conn = nil

		return err
	}

	return nil
}

func (c *Conn) runQueryExec(query string, args ...interface{}) (affected int64, err error) {
	c.logger.Debugf("Running query `%s` on %s", query, c.endpoint)

	var ct pgconn.CommandTag
	if err = c.Connect(); err != nil {
		return 0, err
	} else if ct, err = c.conn.Exec(ctx, query, args...); err != nil {
		return 0, err
	} else {
		return ct.RowsAffected(), nil
	}
}

func (c *Conn) runQueryExists(query string, args ...interface{}) (exists bool, err error) {
	c.logger.Debugf("Running query `%s` on %s", query, c.endpoint)

	err = c.Connect()
	if err != nil {
		return false, err
	}

	var answer string
	err = c.conn.QueryRow(ctx, query, args...).Scan(&answer)

	if err == nil {
		c.logger.Debugf("Query `%s` returns rows for %s", query, c.endpoint)
		return true, nil
	} else if err.Error() == pgx.ErrNoRows.Error() {
		c.logger.Debugf("Query `%s` returns no rows for %s", query, c.endpoint)
		return false, nil
	} else if err == nil {
		c.logger.Debugf("Query `%s` returns rows for %s", query, c.endpoint)
		return true, nil
	} else {
		return false, err
	}
}

func (c *Conn) GetRows(query string, args ...interface{}) ([]map[string]interface{}, error) {
	if err := c.Connect(); err != nil {
		return nil, err
	}
	log.Debugf("Running SQL: %s with args %v", query, args)
	result, err := c.conn.Query(ctx, query, args...)
	defer result.Close()
	if err != nil {
		result.Close()
		return nil, err
	}
	var hdr []string
	for _, col := range result.FieldDescriptions() {
		hdr = append(hdr, string(col.Name))
	}
	var answer []map[string]interface{}
	for result.Next() {
		row := make(map[string]interface{})
		var values []interface{}
		if values, err = result.Values(); err != nil {
			return nil, err
		} else {
			for i, value := range values {
				row[hdr[i]] = value
			}
		}
		answer = append(answer, row)
	}
	return answer, nil
}

func (c *Conn) IsPrimary() (bool, error) {
	return c.runQueryExists("select 'primary' where not pg_is_in_recovery()")
}

func (c *Conn) IsStandby() (bool, error) {
	return c.runQueryExists("select 'standby' where pg_is_in_recovery()")
}
