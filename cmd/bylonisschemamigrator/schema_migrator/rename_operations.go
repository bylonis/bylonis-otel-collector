package schemamigrator

import (
	"strings"
)

// RenameTableOperation is used to represent the RENAME TABLE statement in the SQL.
// It uses IF EXISTS, so running it again after a successful rename is a no-op.
// In an Atomic database the rename only touches metadata: no data is moved.
type RenameTableOperation struct {
	cluster  string
	Database string
	Table    string
	NewTable string
}

func (r RenameTableOperation) OnCluster(cluster string) Operation {
	r.cluster = cluster
	return &r
}

func (r RenameTableOperation) WithReplication() Operation {
	// no-op
	return &r
}

func (r RenameTableOperation) ShouldWaitForDistributionQueue() (bool, string, string) {
	return false, r.Database, r.Table
}

func (r RenameTableOperation) IsMutation() bool {
	return false
}

func (r RenameTableOperation) IsIdempotent() bool {
	return true
}

func (r RenameTableOperation) IsLightweight() bool {
	return true
}

func (r RenameTableOperation) ForceMigrate() bool {
	return false
}

func (r RenameTableOperation) ToSQL() string {
	var sql strings.Builder
	sql.WriteString("RENAME TABLE IF EXISTS ")
	sql.WriteString(r.Database)
	sql.WriteString(".")
	sql.WriteString(r.Table)
	sql.WriteString(" TO ")
	sql.WriteString(r.Database)
	sql.WriteString(".")
	sql.WriteString(r.NewTable)
	if r.cluster != "" {
		sql.WriteString(" ON CLUSTER ")
		sql.WriteString(r.cluster)
	}
	return sql.String()
}

// CreateTableAsOperation is used to represent the CREATE TABLE ... AS other_table
// ENGINE = ... statement in the SQL: the new table copies the columns of AsTable.
// With OrReplace it becomes CREATE OR REPLACE TABLE, which swaps an existing table
// atomically (Atomic databases only); otherwise it uses IF NOT EXISTS.
type CreateTableAsOperation struct {
	cluster   string
	Database  string
	Table     string
	AsTable   string
	Engine    TableEngine
	OrReplace bool
}

func (c CreateTableAsOperation) OnCluster(cluster string) Operation {
	c.cluster = cluster
	c.Engine = c.Engine.OnCluster(cluster)
	return &c
}

func (c CreateTableAsOperation) WithReplication() Operation {
	c.Engine = c.Engine.WithReplication()
	return &c
}

func (c CreateTableAsOperation) ShouldWaitForDistributionQueue() (bool, string, string) {
	// No wait: WaitForDistributionQueue sleeps up to 2 minutes even with an empty
	// queue, and a replace runs right after a rename, while inserts are failing.
	return false, c.Database, c.Table
}

func (c CreateTableAsOperation) IsMutation() bool {
	return false
}

func (c CreateTableAsOperation) IsIdempotent() bool {
	return true
}

func (c CreateTableAsOperation) IsLightweight() bool {
	return true
}

func (c CreateTableAsOperation) ForceMigrate() bool {
	return false
}

func (c CreateTableAsOperation) ToSQL() string {
	var sql strings.Builder
	if c.OrReplace {
		sql.WriteString("CREATE OR REPLACE TABLE ")
	} else {
		sql.WriteString("CREATE TABLE IF NOT EXISTS ")
	}
	sql.WriteString(c.Database)
	sql.WriteString(".")
	sql.WriteString(c.Table)
	if c.cluster != "" {
		sql.WriteString(" ON CLUSTER ")
		sql.WriteString(c.cluster)
	}
	sql.WriteString(" AS ")
	sql.WriteString(c.Database)
	sql.WriteString(".")
	sql.WriteString(c.AsTable)
	sql.WriteString(" ENGINE = ")
	sql.WriteString(c.Engine.ToSQL())
	return sql.String()
}
