package postgresql

import (
	"bytes"
	"fmt"
	"strings"

	// register in driver.
	_ "github.com/jackc/pgx/stdlib"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/internal"
	"github.com/influxdata/telegraf/plugins/inputs"
)

type Postgresql struct {
	Servers          []string
	Outputaddress    string
	MaxLifetime      internal.Duration
	IgnoredDatabases []string
}

var ignoredColumns = map[string]bool{"stats_reset": true}

var sampleConfig = `
  ## specify address via a url matching:
  ##   postgres://[pqgotest[:password]]@localhost[/dbname]\
  ##       ?sslmode=[disable|verify-ca|verify-full]
  ## or a simple string:
  ##   host=localhost user=pqgotest password=... sslmode=... dbname=app_production
  ##
  ## All connection parameters are optional.
  ##
  ## Without the dbname parameter, the driver will default to a database
  ## with the same name as the user. This dbname is just for instantiating a
  ## connection with the server and doesn't restrict the databases we are trying
  ## to grab metrics for.
  ##
  servers = ["host=localhost user=postgres sslmode=disable"]
  ## A custom name for the database that will be used as the "server" tag in the
  ## measurement output. If not specified, a default one generated from
  ## the connection address is used.

  ## A custom name for the database that will be used as the "server" tag in the
  ## measurement output, and we'll add port to the outputaddress by default.
  ## If not specified, the connection host and port are used.
  # outputaddress = "db01"

  ## connection configuration.
  ## maxlifetime - specify the maximum lifetime of a connection.
  ## default is forever (0s)
  max_lifetime = "0s"

  ## A  list of databases to explicitly ignore.  If not specified, metrics for all
  ## databases are gathered.  Do NOT use with the 'databases' option.
  # ignored_databases = ["postgres", "template0", "template1"]

`

func (p *Postgresql) SampleConfig() string {
	return sampleConfig
}

func (p *Postgresql) Description() string {
	return "Read metrics from one or many postgresql servers"
}

func (p *Postgresql) IgnoredColumns() map[string]bool {
	return ignoredColumns
}

func (p *Postgresql) Gather(acc telegraf.Accumulator) error {
	if len(p.Servers) == 0 {
		return fmt.Errorf("no servers")
	}

	for _, serverAddress := range p.Servers {
		acc.AddError(p.gatherServer(serverAddress, acc))
	}

	return nil
}

func (p *Postgresql) gatherServer(address string, acc telegraf.Accumulator) error {
	var (
		err     error
		query   string
		columns []string
	)

	s := Service{
			Address: address,
			Outputaddress: p.Outputaddress,
			MaxIdle: 1,
			MaxOpen: 1,
			MaxLifetime: p.MaxLifetime,
			IsPgBouncer: false,
		}

	err = s.Start()
	if err != nil {
		return err
	}
	defer s.Stop()

	if len(p.IgnoredDatabases) == 0 {
		query = `SELECT * FROM pg_stat_database`
	} else if len(p.IgnoredDatabases) != 0 {
		query = fmt.Sprintf(`SELECT * FROM pg_stat_database WHERE datname NOT IN ('%s')`,
			strings.Join(p.IgnoredDatabases, "','"))
	}

	rows, err := s.DB.Query(query)
	if err != nil {
		return err
	}

	defer rows.Close()

	// grab the column information from the result
	if columns, err = rows.Columns(); err != nil {
		return err
	}

	for rows.Next() {
		err = p.accRow(s, rows, acc, columns)
		if err != nil {
			return err
		}
	}

	query = `SELECT * FROM pg_stat_bgwriter`

	bgWriterRow, err := s.DB.Query(query)
	if err != nil {
		return err
	}

	defer bgWriterRow.Close()

	// grab the column information from the result
	if columns, err = bgWriterRow.Columns(); err != nil {
		return err
	}

	for bgWriterRow.Next() {
		err = p.accRow(s, bgWriterRow, acc, columns)
		if err != nil {
			return err
		}
	}

	return bgWriterRow.Err()
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func (p *Postgresql) accRow(s Service, row scanner, acc telegraf.Accumulator, columns []string) error {
	var columnVars []interface{}
	var dbname bytes.Buffer

	// this is where we'll store the column name with its *interface{}
	columnMap := make(map[string]*interface{})

	for _, column := range columns {
		columnMap[column] = new(interface{})
	}

	// populate the array of interface{} with the pointers in the right order
	for i := 0; i < len(columnMap); i++ {
		columnVars = append(columnVars, columnMap[columns[i]])
	}

	// deconstruct array of variables and send to Scan
	err := row.Scan(columnVars...)

	if err != nil {
		return err
	}
	if columnMap["datname"] != nil {
		// extract the database name from the column map
		if dbNameStr, ok := (*columnMap["datname"]).(string); ok {
			dbname.WriteString(dbNameStr)
		} else {
			// PG 12 adds tracking of global objects to pg_stat_database
			dbname.WriteString("postgres_global")
		}
	} else {
		dbname.WriteString("postgres")
	}

	metas := s.GetConnMeta()
	tags := map[string]string{"server": fmt.Sprintf("%s:%s", metas["host"], metas["port"]), "db": dbname.String()}

	fields := make(map[string]interface{})
	for col, val := range columnMap {
		_, ignore := ignoredColumns[col]
		if !ignore {
			fields[col] = *val
		}
	}
	acc.AddFields("postgresql", fields, tags)

	return nil
}

func init() {
	inputs.Add("postgresql", func() telegraf.Input {
		return &Postgresql{}
	})
}
