package dbprefix

import (
	"context"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// Open is clickhouse.Open for code that uses the database names: with a
// non-default prefix it points the connection's default database at the new
// name and returns a connection that rewrites legacy names on the fly.
func Open(opts *clickhouse.Options) (driver.Conn, error) {
	if !IsDefault() && opts != nil {
		opts.Auth.Database = Rewrite(opts.Auth.Database)
	}
	conn, err := clickhouse.Open(opts)
	if err != nil {
		return nil, err
	}
	return Wrap(conn), nil
}

// Wrap returns conn unchanged for the default prefix, and otherwise a
// connection that rewrites legacy database names in queries and string args.
func Wrap(conn driver.Conn) driver.Conn {
	if IsDefault() {
		return conn
	}
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
