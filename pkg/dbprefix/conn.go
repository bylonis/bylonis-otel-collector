package dbprefix

import (
	"context"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// Open is clickhouse.Open for code that uses the database names: it points the
// connection's default database at the configured prefix and returns a
// connection that rewrites database names on the fly.
func Open(opts *clickhouse.Options) (driver.Conn, error) {
	if opts != nil {
		opts.Auth.Database = Rewrite(opts.Auth.Database)
	}
	conn, err := clickhouse.Open(opts)
	if err != nil {
		return nil, err
	}
	return Wrap(conn), nil
}

// Wrap returns a connection that rewrites database names in queries and string
// args to the configured prefix. It always wraps: even with the default prefix,
// SQL saved with the legacy signoz_<db> names must keep working.
func Wrap(conn driver.Conn) driver.Conn {
	return &rewritingConn{Conn: conn, prefix: prefix}
}

type rewritingConn struct {
	driver.Conn
	prefix string
}

func (c *rewritingConn) query(q string) string { return rewriteWith(c.prefix, q) }

func (c *rewritingConn) args(args []any) []any {
	var out []any
	for i, a := range args {
		s, ok := a.(string)
		if !ok {
			continue
		}
		if r := rewriteWith(c.prefix, s); r != s {
			if out == nil {
				out = append([]any(nil), args...)
			}
			out[i] = r
		}
	}
	if out == nil {
		return args
	}
	return out
}

func (c *rewritingConn) Select(ctx context.Context, dest any, query string, args ...any) error {
	return c.Conn.Select(ctx, dest, c.query(query), c.args(args)...)
}

func (c *rewritingConn) Query(ctx context.Context, query string, args ...any) (driver.Rows, error) {
	return c.Conn.Query(ctx, c.query(query), c.args(args)...)
}

func (c *rewritingConn) QueryRow(ctx context.Context, query string, args ...any) driver.Row {
	return c.Conn.QueryRow(ctx, c.query(query), c.args(args)...)
}

func (c *rewritingConn) PrepareBatch(ctx context.Context, query string, opts ...driver.PrepareBatchOption) (driver.Batch, error) {
	return c.Conn.PrepareBatch(ctx, c.query(query), opts...)
}

func (c *rewritingConn) Exec(ctx context.Context, query string, args ...any) error {
	return c.Conn.Exec(ctx, c.query(query), c.args(args)...)
}

func (c *rewritingConn) AsyncInsert(ctx context.Context, query string, wait bool, args ...any) error {
	return c.Conn.AsyncInsert(ctx, c.query(query), wait, c.args(args)...)
}
