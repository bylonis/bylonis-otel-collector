package schemamigrator

import "strings"

// InsertIntoTable represents an INSERT INTO statement.
// InsertIntoTable can only be used for INSERTs into ReplacingMergeTree tables as of now.
type InsertIntoTable struct {
	cluster string

	Database    string
	Table       string
	Columns     []string
	LightWeight bool
	Synchronous bool
	// Values should be a pre-formatted string representing the VALUES clause,
	// for example: "('col1', 123), ('col2', 456)".
	// The caller is responsible for proper formatting and escaping.
	Values string
}

// OnCluster is a no-op for INSERTs (ClickHouse does not support INSERT ... ON CLUSTER).
func (i InsertIntoTable) OnCluster(cluster string) Operation {
	i.cluster = cluster
	return &i
}

// WithReplication is a no-op for this operation.
func (i InsertIntoTable) WithReplication() Operation {
	return &i
}

// ShouldWaitForDistributionQueue returns false as INSERTs do not start DDL queues.
func (i InsertIntoTable) ShouldWaitForDistributionQueue() (bool, string, string) {
	return false, i.Database, i.Table
}

// IsMutation returns false (does not alter existing data parts).
func (i InsertIntoTable) IsMutation() bool { return false }

func (i InsertIntoTable) IsIdempotent() bool { return false }

// IsLightweight returns true.
func (i InsertIntoTable) IsLightweight() bool { return i.LightWeight }

func (i InsertIntoTable) ForceMigrate() bool {
	return i.Synchronous
}

func (i InsertIntoTable) ToSQL() string {
	var sql strings.Builder
	sql.WriteString("INSERT INTO ")
	sql.WriteString(i.Database)
	sql.WriteString(".")
	sql.WriteString(i.Table)

	if len(i.Columns) > 0 {
		sql.WriteString(" (")
		sql.WriteString(strings.Join(i.Columns, ", "))
		sql.WriteString(")")
	}

	sql.WriteString(" VALUES ")
	sql.WriteString(i.Values)
	return sql.String()
}

// InsertIntoSelect representa un INSERT INTO … SELECT para rellenar una tabla
// con datos que ya existen (bylonis#887). Es una operación pesada, así que corre
// en la fase async del migrador, y esa fase solo ejecuta operaciones
// idempotentes: quien la usa tiene que escribir Query de forma que repetirla no
// duplique datos (por ejemplo, con una guarda sobre la tabla de destino o
// insertando en un ReplacingMergeTree).
type InsertIntoSelect struct {
	Database string
	Table    string
	Columns  []string
	Query    string
}

// OnCluster is a no-op: ClickHouse does not support INSERT ... ON CLUSTER.
// Insert into a Distributed table to reach every shard.
func (i InsertIntoSelect) OnCluster(string) Operation { return &i }

// WithReplication is a no-op for this operation.
func (i InsertIntoSelect) WithReplication() Operation { return &i }

// ShouldWaitForDistributionQueue returns false as INSERTs do not start DDL queues.
func (i InsertIntoSelect) ShouldWaitForDistributionQueue() (bool, string, string) {
	return false, i.Database, i.Table
}

// IsMutation returns false (does not alter existing data parts).
func (i InsertIntoSelect) IsMutation() bool { return false }

// IsIdempotent returns true: Query must be written so that re-running it is safe.
func (i InsertIntoSelect) IsIdempotent() bool { return true }

// IsLightweight returns false: it reads and writes data, so it runs async.
func (i InsertIntoSelect) IsLightweight() bool { return false }

func (i InsertIntoSelect) ForceMigrate() bool { return false }

func (i InsertIntoSelect) ToSQL() string {
	var sql strings.Builder
	sql.WriteString("INSERT INTO ")
	sql.WriteString(i.Database)
	sql.WriteString(".")
	sql.WriteString(i.Table)
	if len(i.Columns) > 0 {
		sql.WriteString(" (")
		sql.WriteString(strings.Join(i.Columns, ", "))
		sql.WriteString(")")
	}
	sql.WriteString(" ")
	sql.WriteString(i.Query)
	return sql.String()
}
