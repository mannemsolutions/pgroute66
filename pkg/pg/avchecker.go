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

func fullTableName() string {
	return fmt.Sprintf("%s.%s", identifierNameSql(AvcSchema), identifierNameSql(AvcTable))
}

func (der AvcDurationExceededError) Error() string {
	return fmt.Sprintf("should have taken %f msec, but actually took %f msec", der.max, der.actually)
}

func (der AvcDurationExceededError) String() string {
	return fmt.Sprintf("exceeded %f by %f msec", der.max, der.actually)
}

func (c *Conn) avcTableExists() (bool, error) {
	if exists, err := c.runQueryExists("select relname from pg_class where relname = $1 and relnamespace in "+
		"(select oid from pg_namespace where nspname=$2)",
		AvcTable, AvcSchema); err != nil {
		return false, fmt.Errorf("failed to check for table %s", fullTableName())
	} else if exists {
		return true, nil
	} else {
		return false, nil
	}
}

func (c *Conn) AvcCreateTable() error {
	log.Infof("Creating table")
	if exists, err := c.avcTableExists(); err != nil {
		log.Errorf("failed to check if table %s exists: %e", fullTableName(), err)
		return err
	} else if exists {
		return nil
	}
	if _, err := c.runQueryExec(fmt.Sprintf("create table %s (%s timestamp)",
		fullTableName(), identifierNameSql(AvcColumn))); err != nil {
		return fmt.Errorf("failed to create table %s", fullTableName())
	}
	if affected, err := c.runQueryExec(fmt.Sprintf("insert into %s values(now())", fullTableName())); err != nil {
		return fmt.Errorf("failed to create table %s", fullTableName())
	} else if affected != 1 {
		return fmt.Errorf("unexpected result while inserting into table %s", fullTableName)
	}
	return nil
}

func (c *Conn) avCheckerGetDuration() (float64, error) {
	fullColName := identifierNameSql(AvcColumn)
	if exists, err := c.avcTableExists(); err != nil {
		log.Errorf("failed to check if table %s exists: %e", fullTableName(), err)
		return 0, err
	} else if !exists {
		return -1, nil
	}
	qry := fmt.Sprintf("select extract('epoch' from (now()-%s)) duration from %s", fullColName, fullTableName())
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

func (c *Conn) AvUpdateDuration() error {
	var affected int64
	if isPrimary, err := c.IsPrimary(); err != nil {
		return err
	} else if !isPrimary {
		log.Infof("skipping update of %s on a standby database server", fullTableName())
		return nil
	} else if err = c.AvcCreateTable(); err != nil {
		return err
	} else if affected, err = c.runQueryExec(fmt.Sprintf("update %s set %s = now()",
		fullTableName(), identifierNameSql(AvcColumn))); err != nil {
		return err
	} else if affected != 1 {
		return fmt.Errorf("unexpecetedly updated %d rows instead of 1 for %s", affected, fullTableName())
	}
	return nil
}

func (c *Conn) AvCheckDuration(max float64) error {
	var err error
	var since float64
	if since, err = c.avCheckerGetDuration(); err != nil {
		return err
	} else if since < 0 {
		return fmt.Errorf("table %s does not exist", fullTableName())
	} else if since > max {
		return AvcDurationExceededError{
			max:      max,
			actually: since,
		}
	}
	return nil
}
