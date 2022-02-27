// Package database provides basic interfaces for modelling data from the
// database and working with relations.
package database

import (
	"context"
	"database/sql"
	"net/url"
	"strconv"
	"strings"

	"djinn-ci.com/errors"

	"github.com/andrewpillar/query"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

// Loader is the interface that wraps the Load method. This should be
// implemented on stores.
//
// Load will get the values under the foreign key, fk, from the given models.
// The underlying store should then get all models where the primary key, pk,
// is in the list of values extracted with fk. Then, each of these models
// should be iterated through and be bound to each model.
//
// This is typically used for loading in models based on relations between
// tables where fk and pk define the relations.
type Loader interface {
	Load(fk, pk string, mm ...Model) error
}

// RelationFunc is a callback that will return the foreign key, primary key
// and Loader for loading a model's relation.
type RelationFunc func() (string, string, Loader)

// Relation returns a RelationFunc that will return the foreign key, primary
// key, and Loader that were given.
func Relation(fk, pk string, ld Loader) RelationFunc {
	return func() (string, string, Loader) {
		return fk, pk, ld
	}
}

// Int64 will attempt to type assert the given interface to either int64 or
// sql.NullInt64 and return the underlying value along with whether or not it
// was successful.
func Int64(val interface{}) (int64, bool) {
	switch v := val.(type) {
	case int64:
		return v, true
	case sql.NullInt64:
		return v.Int64, v.Valid
	default:
		return 0, false
	}
}

// Bind will bind the loaded models to the target model using the given foreign
// and primary keys as the relation between the two.
func Bind(fk, pk string, loaded, targets []Model) {
	mtab := make(map[int64][]Model)

	for _, m := range loaded {
		i64, ok := Int64(m.Values()[pk])

		if !ok {
			continue
		}
		mtab[i64] = append(mtab[i64], m)
	}

	for _, t := range targets {
		i64, ok := Int64(t.Values()[fk])

		if !ok {
			continue
		}

		for _, m := range mtab[i64] {
			t.Bind(m)
		}
	}
}

// LoadRelations will invoke each of the given RelationFuncs and use the
// returned values to load the relation for each of the models.
func LoadRelations(mm []Model, relations ...RelationFunc) error {
	for _, rel := range relations {
		fk, pk, ld := rel()

		if err := ld.Load(fk, pk, mm...); err != nil {
			return errors.Err(err)
		}
	}
	return nil
}

// GetModelFunc is a function that will get a Model for the given id along with
// whether or not one could be found.
type GetModelFunc func(int64) (Model, bool, error)

type Model interface {
	// Dest should return a slice of pointers into which a row can be scanned.
	Dest() []interface{}

	// Bind the given Model to the current Model. This should only bind the
	// given Model if it relates to the Model being bound to.
	Bind(m Model)

	// JSON will return a map[string]interface{} representation of the current
	// Model. This is intended for JSON encoding. The given string is used as
	// the address of the server from which the JSON response is served from.
	// The address is used to set any URL fields that may be set in the map.
	JSON(addr string) map[string]interface{}

	// Endpoint will return the URI from which the Model is served from. The
	// given list will be appended to the end of the endpoint.
	Endpoint(...string) string

	// Values returns a map of the values of the current Model. This should
	// serve as a direct mapping of the underlying row in the table.
	Values() map[string]interface{}
}

// ErrNotFound indicates when a database row could not be found.
var ErrNotFound = errors.New("not found")

// Values returns all of the values for the given key from the given slice of
// models.
func Values(key string, mm []Model) []interface{} {
	vals := make([]interface{}, 0, len(mm))

	for _, m := range mm {
		if val, ok := m.Values()[key]; ok {
			vals = append(vals, val)
		}
	}
	return vals
}

// List returns a query.Expr for a list of values. If the given list of values
// is empty, then a query.List expression is simply returned, only containing
// a single -1 in the list. This will allow for situations where queries that
// use WHERE IN (...) to still build in a valid way.
func List(vals ...interface{}) query.Expr {
	if len(vals) == 0 {
		return query.List(-1)
	}
	return query.List(vals...)
}

// Search returns a query option for adding a WHERE LIKE clause to a query for
// the given column and pattern. If the pattern is empty, then no option is
// applied.
func Search(col, pattern string) query.Option {
	return func(q query.Query) query.Query {
		if pattern == "" {
			return q
		}
		return query.Where("LOWER("+col+")", "LIKE", query.Arg("%"+strings.ToLower(pattern)+"%"))(q)
	}
}

// Chown will update the user_id column of all rows in the given table. The
// old id, from, is used to determine which rows to update, and the new id
// to, is what the rows are updated with.
func Chown(db Pool, table string, from, to int64) error {
	q := query.Update(
		table,
		query.Set("user_id", query.Arg(to)),
		query.Where("user_id", "=", query.Arg(from)),
	)

	if _, err := db.Exec(q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}
	return nil
}

type Pool struct {
	*pgxpool.Pool
}

// Poolect returns a database connection to a PostgreSQL database using the
// given dsn. Once the connection is open a subsequent Ping is made to the
// database to check the connectivity.
func Connect(ctx context.Context, dsn string) (Pool, error) {
	pool, err := pgxpool.Connect(ctx, dsn)

	if err != nil {
		return Pool{}, errors.Err(err)
	}

	if err := pool.Ping(ctx); err != nil {
		return Pool{}, errors.Err(err)
	}
	return Pool{
		Pool: pool,
	}, nil
}

func (db Pool) QueryRow(sql string, args ...interface{}) pgx.Row {
	return db.Pool.QueryRow(context.Background(), sql, args...)
}

func (db Pool) Query(sql string, args ...interface{}) (pgx.Rows, error) {
	return db.Pool.Query(context.Background(), sql, args...)
}

func (db Pool) Exec(sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return db.Pool.Exec(context.Background(), sql, args...)
}

// Get will query a single row in the database in the given table, and attempt
// to scan the values returned into the given model. The returned  bool value
// denotes whether a row was found.
func (db Pool) Get(table string, m Model, opts ...query.Option) (bool, error) {
	opts = append([]query.Option{
		query.From(table),
	}, opts...)

	q := query.Select(query.Columns("*"), opts...)

	if err := db.QueryRow(q.Build(), q.Args()...).Scan(m.Dest()...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, errors.Err(err)
	}
	return true, nil
}

// All will query multiple rows in the database in the given table. The given
// function is called to scan the rows values.
func (db Pool) All(table string, new func() Model, opts ...query.Option) error {
	opts = append([]query.Option{
		query.From(table),
	}, opts...)

	q := query.Select(query.Columns("*"), opts...)

	rows, err := db.Query(q.Build(), q.Args()...)

	if err != nil {
		return errors.Err(err)
	}

	for rows.Next() {
		m := new()

		if err := rows.Scan(m.Dest()...); err != nil {
			return errors.Err(err)
		}
	}
	return nil
}

// Paginator stores information about a paginated table. This is typically
// used for performing subsequent queries to retrieve the offset rows from
// the table.
type Paginator struct {
	Next   int64
	Prev   int64
	Offset int64
	Page   int64
	Limit  int64
	Pages  []int64
}

// PageLimit is the default number of rows we want to have returned per page.
var PageLimit int64 = 25

// Paginate returns a Paginator that marks the current page of a table that is
// being viewed. The size of the page is specified via limit.
func (db Pool) Paginate(table string, page, limit int64, opts ...query.Option) (Paginator, error) {
	if page <= 0 {
		page = 1
	}

	p := Paginator{
		Page:  page,
		Limit: limit,
	}

	opts = append([]query.Option{
		query.From(table),
	}, opts...)

	q := query.Select(query.Count("*"), opts...)

	var count int64

	if err := db.QueryRow(q.Build(), q.Args()...).Scan(&count); err != nil {
		return p, errors.Err(err)
	}

	pages := count / limit

	if count%limit != 0 {
		pages++
	}

	if p.Page > count {
		p.Page = pages
	}

	if p.Page != 0 {
		p.Offset = (p.Page - 1) * limit
	}

	for i := int64(0); i < pages; i++ {
		p.Pages = append(p.Pages, i+1)
	}

	p.Next = p.Page + 1
	p.Prev = p.Page - 1

	if p.Prev < 1 {
		p.Prev = 1
	}

	if p.Next > pages {
		p.Next = pages
	}
	return p, nil
}

func pageURL(url *url.URL, page int64) string {
	q := url.Query()
	q.Set("page", strconv.FormatInt(page, 10))

	url.RawQuery = q.Encode()

	return url.String()
}

// EncodeToLink returns the current Paginator encoded to a Link header value
// for the given URL.
func (p Paginator) EncodeToLink(url *url.URL) string {
	prev := pageURL(url, p.Prev)
	next := pageURL(url, p.Next)

	return `<` + prev + `>; rel="prev", <` + next + `>; rel="next"`
}
