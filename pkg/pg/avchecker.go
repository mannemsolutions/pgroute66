package pg

import "fmt"

const (
	AvcSchema = "public"
	AvcTable  = "pgr66_avc"
	AvcColumn = "pgr66_avc"
)

type AvcDurationExceededError struct {
	max      float64
	actually float64
}

func (der AvcDurationExceededError) Error() string {
	return fmt.Sprintf("should have taken %f msec, but actually took %f msec", der.max, der.actually)
}

func (c *Conn) avcTable() error {
	fullTableName := fmt.Sprintf("%s.%s", identifierNameSql(AvcSchema), identifierNameSql(AvcTable))
	if exists, err := c.runQueryExists("select relname from pg_class where relname = $1 and relnamespace in "+
		"(select oid from pg_namespace where nspname=$2)",
		AvcTable, AvcSchema); err != nil {
		return fmt.Errorf("failed to check for table %s", fullTableName)
	} else if exists {
		return nil
	}
	log.Infof("Creating table")
	if _, err := c.runQueryExec(fmt.Sprintf("create table %s (%s timestamp)",
		fullTableName, identifierNameSql(AvcColumn))); err != nil {
		return fmt.Errorf("failed to create table %s", fullTableName)
	}
	if affected, err := c.runQueryExec(fmt.Sprintf("insert into %s values(now())", fullTableName)); err != nil {
		return fmt.Errorf("failed to create table %s", fullTableName)
	} else if affected != 1 {
		return fmt.Errorf("unexpected result while inserting into table %s", fullTableName)
	}
	return nil
}

func (c *Conn) avCheckerDuration() (float64, error) {
	fullColName := identifierNameSql(AvcColumn)
	fullTableName := fmt.Sprintf("%s.%s", identifierNameSql(AvcSchema), identifierNameSql(AvcTable))
	qry := fmt.Sprintf("select extract('epoch' from (now()-%s)) duration from %s", fullColName, fullTableName)
	if result, err := c.GetRows(qry); err != nil {
		log.Errorf("failed to retrieve duration from postgres: %e", err)
		return 0, err
	} else if len(result) != 1 {
		return 0, fmt.Errorf("unexpected result while checking for duration (%d != 1)", len(result))
	} else if value, valueOk := result[0]["duration"]; !valueOk {
		return 0, fmt.Errorf("unexpected result while checking for duration (%d != 1)", len(result))
	} else if mSec, mSecOk := value.(float64); !mSecOk {
		return 0, fmt.Errorf("unexpected result type checking for duration (%T != float64)", value)
	} else {
		return mSec, nil
	}
}

func (c *Conn) AvChecker(max float64) error {
	if err := c.avcTable(); err != nil {
		return err
	} else if since, cdErr := c.avCheckerDuration(); cdErr != nil {
		return cdErr
	} else if since > max {
		return AvcDurationExceededError{
			max:      max,
			actually: since,
		}
	}
	return nil
}

/*
	cur.execute('BEGIN')
	cur.execute('update public.avchecker set last = now()')
	cur.execute('COMMIT')
	cur.execute('select last from public.avchecker')
	row=next(cur)
	new = row[0]
	if last:
	delta = new-last
	if delta.total_seconds() >= (SLEEPTIME*1.5):
	print(delta, flush=True)
	last = new
*/

/*
#!/usr/bin/env python3
import os
import psycopg2
import time
cn=None
last = None
SLEEPTIME=float(os.environ.get('AVCHECKER_SLEEPTIME', '5'))
while True:
    try:
        time.sleep(SLEEPTIME)
        if not cn:
            cn = psycopg2.connect('')
            cur = cn.cursor()
            cur.execute('select count(*) from pg_class where relname = %s and relnamespace in (select oid from pg_namespace where nspname=%s) ', ('avchecker', 'public'))
            if next(cur)[0] == 0:
                print('Creating table')
                cur.execute('create table public.avchecker(last timestamp)')
                cur.execute('insert into public.avchecker values(now())')
        cur.execute('BEGIN')
        cur.execute('update public.avchecker set last = now()')
        cur.execute('COMMIT')
        cur.execute('select last from public.avchecker')
        row=next(cur)
        new = row[0]
        if last:
            delta = new-last
            if delta.total_seconds() >= (SLEEPTIME*1.5):
                print(delta, flush=True)
        last = new
    except (psycopg2.InternalError, psycopg2.OperationalError, psycopg2.DatabaseError) as err:
        print(str(err).split('\n')[0])
        cn = None
*/
