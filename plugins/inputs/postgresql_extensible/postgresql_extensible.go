package postgresql_extensible

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	_ "github.com/jackc/pgx/stdlib" //to register stdlib from PostgreSQL Driver and Toolkit

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/internal"
	"github.com/influxdata/telegraf/plugins/inputs"
	"github.com/influxdata/telegraf/plugins/inputs/postgresql"
)

type Postgresql struct {
	Servers           []string
	Outputaddress     string
	MaxLifetime       internal.Duration
	IgnoredDatabases  []string
	AdditionalTags    []string
	Timestamp      string
	Query          query
	Debug          bool

	Log telegraf.Logger
}

type query []struct {
	Sqlquery          string
	Script            string
	Version           int
	Withignoredbname  bool
	Tagvalue    string
	Measurement string
	Timestamp   string
}

var ignoredColumns = map[string]bool{"stats_reset": true}

var sampleConfig = `
  ## specify address via a url matching:
  ##   postgres://[pqgotest[:password]]@localhost[/dbname]\
  ##       ?sslmode=[disable|verify-ca|verify-full]
  ## or a simple string:
  ##   host=localhost user=pqgotest password=... sslmode=... dbname=app_production
  #
  ## All connection parameters are optional.  #
  ## Without the dbname parameter, the driver will default to a database
  ## with the same name as the user. This dbname is just for instantiating a
  ## connection with the server and doesn't restrict the databases we are trying
  ## to grab metrics for.
  #
  servers = ["host=localhost user=postgres sslmode=disable"]

  ## connection configuration.
  ## maxlifetime - specify the maximum lifetime of a connection.
  ## default is forever (0s)
  max_lifetime = "0s"

  ## A custom name for the database that will be used as the "server" tag in the
  ## measurement output, and we'll add port to the outputaddress by default.
  ## If not specified, the connection host and port are used.
  # outputaddress = "db01"

  ## A  list of databases to explicitly ignore.  If not specified, metrics for all
  ## databases are gathered.  Do NOT use with the 'databases' option.
  # ignored_databases = ["postgres", "template0", "template1"]

  ## Define the toml config where the sql queries are stored
  ## New queries can be added, if the withdbname is set to true and there is no
  ## databases defined in the 'databases field', the sql query is ended by a
  ## 'is not null' in order to make the query succeed.
  ## Example :
  ## The sqlquery : "SELECT * FROM pg_stat_database where datname" become
  ## "SELECT * FROM pg_stat_database where datname IN ('postgres', 'pgbench')"
  ## because the databases variable was set to ['postgres', 'pgbench' ] and the
  ## withdbname was true. Be careful that if the withdbname is set to false you
  ## don't have to define the where clause (aka with the dbname) the tagvalue
  ## field is used to define custom tags (separated by commas)
  ## The optional "measurement" value can be used to override the default
  ## output measurement name ("postgresql").
  ##
  ## The script option can be used to specify the .sql file path.
  ## If script and sqlquery options specified at same time, sqlquery will be used 
  ##
  ## the tagvalue field is used to define custom tags (separated by comas).
  ## the query is expected to return columns which match the names of the
  ## defined tags. The values in these columns must be of a string-type,
  ## a number-type or a blob-type.
  ## 
  ## The timestamp field is used to override the data points timestamp value. By
  ## default, all rows inserted with current time. By setting a timestamp column,
  ## the row will be inserted with that column's value. 
  ##
  ## Structure :
  ## [[inputs.postgresql_extensible.query]]
  ##   sqlquery string
  ##   version string
  ##   withdbname boolean
  ##   tagvalue string (comma separated)
  ##   measurement string
  ##   timestamp string
  [[inputs.postgresql_extensible.query]]
    sqlquery="SELECT * FROM pg_stat_database"
    version=901
    withdbname=false
    tagvalue=""
    measurement=""
  [[inputs.postgresql_extensible.query]]
    sqlquery="SELECT * FROM pg_stat_bgwriter"
    version=901
    withdbname=false
    tagvalue="postgresql.stats"
`

func (p *Postgresql) Init() error {
	var err error
	for i := range p.Query {
		if p.Query[i].Sqlquery == "" {
			p.Query[i].Sqlquery, err = ReadQueryFromFile(p.Query[i].Script)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *Postgresql) SampleConfig() string {
	return sampleConfig
}

func (p *Postgresql) Description() string {
	return "Read metrics from one or many postgresql servers"
}

func (p *Postgresql) IgnoredColumns() map[string]bool {
	return ignoredColumns
}

func ReadQueryFromFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	query, err := ioutil.ReadAll(file)
	if err != nil {
		return "", err
	}
	return string(query), err
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
		err        error
		sqlQuery   string
		queryAddon string
		dbVersion  int
		query      string
		tagValue   string
		measName   string
		timestamp  string
		columns    []string
	)

	s := postgresql.Service{
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

	// Retrieving the database version
	query = `SELECT setting::integer / 100 AS version FROM pg_settings WHERE name = 'server_version_num'`
	if err = s.DB.QueryRow(query).Scan(&dbVersion); err != nil {
		dbVersion = 0
	}

	// We loop in order to process each query
	// Query is not run if Database version does not match the query version.
	for i := range p.Query {
		sqlQuery = p.Query[i].Sqlquery
		tagValue = p.Query[i].Tagvalue
		timestamp = p.Query[i].Timestamp

		if p.Query[i].Measurement != "" {
			measName = p.Query[i].Measurement
		} else {
			measName = "postgresql"
		}

		if p.Query[i].Withignoredbname {
			if len(p.IgnoredDatabases) != 0 {
				queryAddon = fmt.Sprintf(` NOT IN ('%s')`, strings.Join(p.IgnoredDatabases, "','"))
			} else {
				queryAddon = " is not null"
			}
		} else {
			queryAddon = ""
		}
		sqlQuery += queryAddon

		if p.Query[i].Version <= dbVersion {
			rows, err := s.DB.Query(sqlQuery)
			if err != nil {
				p.Log.Error(err.Error())
				continue
			}

			defer rows.Close()

			// grab the column information from the result
			if columns, err = rows.Columns(); err != nil {
				p.Log.Error(err.Error())
				continue
			}

			p.AdditionalTags = nil
			if tagValue != "" {
				tagList := strings.Split(tagValue, ",")
				for t := range tagList {
					p.AdditionalTags = append(p.AdditionalTags, tagList[t])
				}
			}

			p.Timestamp = timestamp

			for rows.Next() {
				err = p.accRow(s, measName, rows, acc, columns)
				if err != nil {
					p.Log.Error(err.Error())
					break
				}
			}
		}
	}
	return nil
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func (p *Postgresql) accRow(s postgresql.Service, measName string, row scanner, acc telegraf.Accumulator, columns []string) error {
	var (
		err        error
		columnVars []interface{}
		dbname     bytes.Buffer
		timestamp  time.Time
	)

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
	if err = row.Scan(columnVars...); err != nil {
		return err
	}

	if c, ok := columnMap["datname"]; ok && *c != nil {
		// extract the database name from the column map
		switch datname := (*c).(type) {
		case string:
			dbname.WriteString(datname)
		default:
			dbname.WriteString("postgres")
		}
	} else {
		dbname.WriteString("postgres")
	}

	metas := s.GetConnMeta()

	// Process the additional tags
	tags := map[string]string{
		"server": fmt.Sprintf("%s:%s", metas["host"], metas["port"]),
		"db":     dbname.String(),
	}

	// set default timestamp to Now
	timestamp = time.Now()

	fields := make(map[string]interface{})
COLUMN:
	for col, val := range columnMap {
		p.Log.Debugf("Column: %s = %T: %v\n", col, *val, *val)
		_, ignore := ignoredColumns[col]
		if ignore || *val == nil {
			continue
		}

		if col == p.Timestamp {
			if v, ok := (*val).(time.Time); ok {
				timestamp = v
			}
			continue
		}

		for _, tag := range p.AdditionalTags {
			if col != tag {
				continue
			}
			switch v := (*val).(type) {
			case string:
				tags[col] = v
			case []byte:
				tags[col] = string(v)
			case int64, int32, int:
				tags[col] = fmt.Sprintf("%d", v)
			default:
				p.Log.Debugf("Failed to add %q as additional tag", col)
			}
			continue COLUMN
		}

		if v, ok := (*val).([]byte); ok {
			fields[col] = string(v)
		} else {
			fields[col] = *val
		}
	}
	acc.AddFields(measName, fields, tags, timestamp)
	return nil
}

func init() {
	inputs.Add("postgresql_extensible", func() telegraf.Input {
		return &Postgresql{}
	})
}
