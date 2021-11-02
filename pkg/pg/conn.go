package pg

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4"
	"os"
	"strings"
)

type Conn struct {
	connParams Dsn
	endpoint string
	conn       *pgx.Conn
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

func (c *Conn) Connect() (err error) {
	if c.conn != nil {
		if c.conn.IsClosed() {
			c.conn = nil
		} else {
			log.Debugf("Already connected to %v", c.DSN())
			return nil
		}
	}
	log.Debugf("Connecting to %s (%v)", c.endpoint, c.DSN())
	c.conn, err = pgx.Connect(context.Background(), c.DSN())
	if err != nil {
		c.conn = nil
		return err
	}
	return nil
}

func (c *Conn) runQueryExists(query string, args ...interface{}) (exists bool, err error) {
	log.Debugf("Running query `%s` on %s", query, c.endpoint)
	err = c.Connect()
	if err != nil {
		return false, err
	}
	var answer string
	err = c.conn.QueryRow(context.Background(), query, args...).Scan(&answer)
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
