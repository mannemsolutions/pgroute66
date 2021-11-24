package pg

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"os"
	"strings"
)

type Conn struct {
	connParams Dsn
	endpoint string
	pool     *pgxpool.Pool
}

func NewConn(connParams Dsn) (c *Conn) {
	c = &Conn{
		connParams: connParams,
	}
	c.endpoint = fmt.Sprintf("%s:%s", c.Host(), c.Port())

	return c
}

func (c *Conn) DSN() (dsn string) {
	var pairs []string
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

func (c *Conn) Connect() (pool *pgxpool.Conn, err error) {
	log.Debugf("Connecting to %s (%v)", c.endpoint, c.DSN())
	if c.pool == nil {
		log.Debugf("Creating pool for %s (%v)", c.endpoint, c.DSN())
		c.pool, err = pgxpool.Connect(context.Background(), c.DSN())
	}
	return c.pool.Acquire(context.Background())
}

func (c *Conn) runQueryExists(query string, args ...interface{}) (exists bool, err error) {
	conn, err := c.Connect()
	if err != nil {
		log.Debugf("Error connecting to %s", c.endpoint)
		return false, err
	}
	log.Debugf("Running query `%s` on %s", query, c.endpoint)
	var answer string
	err = conn.QueryRow(context.Background(), query, args...).Scan(&answer)
	conn.Release()
	if err == pgx.ErrNoRows {
		log.Debugf("Query `%s` returns no rows for %s", query, c.endpoint)
		return false, nil
	}
	if err == nil {
		log.Debugf("Query `%s` returns rows for %s", query, c.endpoint)
		return true, nil
	}
	return false, err
}

func (c *Conn) IsPrimary() (bool, error) {
	return c.runQueryExists("select 'primary' where not pg_is_in_recovery()")
}

func (c *Conn) IsStandby() (bool, error) {
	return c.runQueryExists("select 'standby' where pg_is_in_recovery()")
}
