package internal

import "github.com/mannemsolutions/pgroute66/pkg/pg"

type RouteConnections map[string]*pg.Conn

func (rcs RouteConnections) FilteredConnections(filter []string) RouteConnections {
	log.Debugf("filtering on name: %s", filter)
	fcs := make(RouteConnections)
	for _, host := range filter {
		if conn, ok := rcs[host]; ok {
			fcs[host] = conn
		}
	}
	return fcs
}
