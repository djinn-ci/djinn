// Package model provides basic interfaces for modelling data from the database.
package model

import (
	"database/sql"
	"database/sql/driver"
	"strings"

	"github.com/andrewpillar/thrall/errors"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"
)

type Binder interface {
	// Bind the given models to the implementation. This would typically be
	// used for binding related models to a model, or to a Store if you want
	// to constrain the queries performed via a Store.
	Bind(...Model)
}

// LoaderFunc is the callback that is called when a model is to be loaded into
// its relating model. This takes an integer, that represents an index in a
// slice of models, and the model being loaded, for example,
//
//     fn := func(i int, m Model) {
//         p := posts[i]
//         _, id := m.Primary()
//         if p.UserID == id {
//             p.User = m
//         }
//     }
type LoaderFunc func(int, Model)

// RelationFunc is a callback that is returned from the Relation function. This
// will perform the actual loading of the model's relationships.
type RelationFunc func(Loader, ...Model) error

type Loader interface {
	// Load will load models under the given key for the given slice of values.
	// The LoaderFunc will be invoked for each model that has been retrieved
	// from the database.
	Load(string, []interface{}, LoaderFunc) error
}

// Loaders is a type for storing a Loader by their respective name, for loading
// model relationships.
type Loaders map[string]Loader

// Model interface wraps the basic methods that a model will have. This assumes
// that models implementing this interface use 64 bit integers for their
// primary keys.
type Model interface {
	Binder

	// SetPrimary will set the value of the primary key.
	SetPrimary(int64)

	// Primary will return the name of the column for the primary key, and the
	// column's value.
	Primary() (string, int64)

	// IsZero will determine if the model is a zero value. This should return
	// true on underlying nil types.
	IsZero() bool

	// JSON will return a map of the fields from the Model that should be used
	// for JSON representation. The given string will be used as the address of
	// the server from which the Model can be accessed. This will be used to
	// set any URL fields that may be set in the returned map.
	JSON(string) map[string]interface{}

	// Endpoint will return the endpoint used to access the model. This will
	// append the given variadic list of strings to the returned endpoint.
	Endpoint(...string) string

	// Values will return a map of the model's values. This will be called
	// during calls to Store.Create, and Store.Update.
	Values() map[string]interface{}
}

type selectFunc func(interface{}, string, ...interface{}) error

// Store is a simple struct for performing SELECT, INSERT, UPDATE, and
// DELETE queries on tables.
type Store struct {
	*sqlx.DB
}

// Paginator stores information about a paginated table. This is typically
// used for performing subsequent queries to retrieve the offset records from
// the table.
type Paginator struct {
	Next   int64
	Prev   int64
	Offset int64
	Page   int64
	Pages  []int64
}

var (
	PageLimit int64 = 25
	ErrNotFound     = errors.New("not found")
)

// getInt64 returns the underlying int64 value from the given interface, and if
// the value was of int64. This assumes the given interface value is either
// int64 or sql.NullInt64.
func getInt64(val interface{}) (int64, bool) {
	switch val.(type) {
	case int64:
		return val.(int64), true
	case sql.NullInt64:
		return val.(sql.NullInt64).Int64, true
	default:
		return int64(0), false
	}
}

// getKey gets key from the given model, if the key is id then the primary
// will be returned, since this won't exist in the Values map.
func getKey(key string, m Model) interface{} {
	col, val := m.Primary()

	if key == col {
		return val
	}
	return m.Values()[key]
}

// NewLoaders creates a new empty Loaders store.
func NewLoaders() Loaders { return Loaders(make(map[string]Loader)) }

// Bind returns a LoaderFunc that checks to see if the key on the target model,
// specified via a, at index i matches the key on the model being loaded,
// specified via b. If so, then that model is bound to the target model. This
// typically assumes that both the keys being retrieved have an underlying type
// of int64.
func Bind(a, b string, mm ...Model) func(int, Model) {
	return func(i int, r Model) {
		if i > len(mm) || len(mm) == 0 {
			return
		}

		m := mm[i]
		if CompareKeys(getKey(a, m), getKey(b, r)) {
			m.Bind(r)
		}
	}
}

// CompareKeys compares the two given interface values, assuming they are
// either of type int64, or sql.NullInt64.
func CompareKeys(a, b interface{}) bool {
	var (
		fk int64
		pk int64
		ok bool
	)

	if fk, ok = getInt64(a); !ok {
		return false
	}

	if pk, ok = getInt64(b); !ok {
		return false
	}
	return fk == pk
}

// MapKey returns a slice of values for the given key from the given slice
// of models.
func MapKey(key string, mm []Model) []interface{} {
	vals := make([]interface{}, 0, len(mm))

	for _, m := range mm {
		col, val := m.Primary()

		if key == col {
			vals = append(vals, val)
			continue
		}

		if val, ok := m.Values()[key]; ok {
			vals = append(vals, val)
		}
	}
	return vals
}

// LoadRelation loads all of the given relations from the given map, for all
// of the given models, using the respective Loader from the given Loaders
// type.
func LoadRelations(rr map[string]RelationFunc, loaders Loaders, mm ...Model) error {
	for relation, fn := range rr {
		l, ok := loaders.Get(relation)

		if !ok {
			continue
		}

		if err := fn(l, mm...); err != nil {
			return errors.Err(err)
		}
	}
	return nil
}

// Relation is used for defining relations between models. This returns a
// RelationFunc, which when called will invoke the given Loader against the
// given models. The returned callback will use the defined foreign key, and
// primary key for actually performing the relationship loading, and binding.
func Relation(fk, pk string) RelationFunc {
	return func(l Loader, mm ...Model) error {
		return errors.Err(l.Load(pk, MapKey(fk, mm), Bind(fk, pk, mm...)))
	}
}

// Scan the given interface value into a slice of bytes.
func Scan(val interface{}) ([]byte, error) {
	if val == nil {
		return []byte{}, nil
	}

	str, err := driver.String.ConvertValue(val)

	if err != nil {
		return []byte{}, errors.Err(err)
	}

	switch str.(type) {
	case string:
		return []byte(str.(string)), nil
	case []byte:
		return str.([]byte), nil
	default:
		return []byte{}, errors.New("failed to Scan bytes - not a string or byte slice")
	}
}

// Search returns a WHERE LIKE clause for the given column and pattern. If the
// pattern is empty then no WHERE LIKE clause is returned.
func Search(col, pattern string) query.Option {
	return func(q query.Query) query.Query {
		if pattern == "" {
			return q
		}
		return query.Where("LOWER("+col+")", "LIKE", "%"+strings.ToLower(pattern)+"%")(q)
	}
}

// Slice converts a slice of models of length l, into a slice of Model.
// The given callback takes the current index of the new Model slice as
// its only argument. It is expected for this index to be used to return the
// original type that implements the Model interface from a source slice.
func Slice(l int, get func(int) Model) []Model {
	mm := make([]Model, l, l)

	for i := range mm {
		mm[i] = get(i)
	}
	return mm
}

// Where returns a WHERE clause on the given Model if the given Model is
// non-zero. The args variadic argument is used to specify the column, and
// value to use for the WHERE clause. The first item in the argument is the
// column on which the WHERE clause is performed. The second item is the value
// to use in thw WHERE clause. If no second item is given then the primary
// key of the given model is used instead.
func Where(m Model, args ...string) query.Option {
	return func(q query.Query) query.Query {
		if len(args) < 1 || m == nil || m.IsZero() {
			return q
		}
		var val interface{}

		col := args[0]

		if len(args) > 1 {
			val = m.Values()[args[1]]
		} else {
			_, val = m.Primary()
		}
		return query.Where(col, "=", val)(q)
	}
}

// OrWhere returns an OR WHERE clause on the given Model. This operates the
// same way as Where, only the returned clause is different.
func OrWhere(m Model, args ...string) query.Option {
	return func(q query.Query) query.Query {
		if len(args) < 1 || m == nil || m.IsZero() {
			return q
		}
		var val interface{}

		col := args[0]

		if len(args) > 1 {
			val = m.Values()[args[1]]
		} else {
			_, val = m.Primary()
		}
		return query.OrWhere(col, "=", val)(q)
	}
}

// Put adds a Loader of the given name to the underlying map.
func (m *Loaders) Put(name string, l Loader) {
	if (*m) == nil {
		(*m) = make(map[string]Loader)
	}
	(*m)[name] = l
}

// Get returns a Loader of the given name.
func (m *Loaders) Get(name string) (Loader, bool) {
	if m == nil {
		return nil, false
	}
	l, ok := (*m)[name]
	return l, ok
}

func (s Store) doSelect(fn selectFunc, i interface{}, table string, opts ...query.Option) error {
	opts = append([]query.Option{
		query.Columns("*"),
		query.From(table),
	}, opts...)

	q := query.Select(opts...)

	err := fn(i, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return errors.Err(err)
}

// Get performs a SELECT query on the given table using the given query
// options. This will return a single record from the given table.
func (s Store) Get(i interface{}, table string, opts ...query.Option) error {
	return errors.Err(s.doSelect(s.DB.Get, i, table, opts...))
}

// All performs a SELECT query on the given table using the given query
// options. The given interface is expected to be a slice, which is then
// populated via sqlx.
func (s Store) All(i interface{}, table string, opts ...query.Option) error {
	return errors.Err(s.doSelect(s.DB.Select, i, table, opts...))
}

// Create performs an INSERT on the given table for each given model. The ID of
// the new record is set on the created model.
func (s Store) Create(table string, mm ...Model) error {
	if len(mm) == 0 {
		return nil
	}

	for _, m := range mm {
		modelVals := m.Values()

		cols := make([]string, 0, len(modelVals))
		vals := make([]interface{}, 0, len(modelVals))

		for k, v := range modelVals {
			cols = append(cols, k)
			vals = append(vals, v)
		}

		q := query.Insert(
			query.Into(table),
			query.Columns(cols...),
			query.Values(vals...),
			query.Returning("id"),
		)

		stmt, err := s.Prepare(q.Build())

		if err != nil {
			return errors.Err(err)
		}

		defer stmt.Close()

		row := stmt.QueryRow(q.Args()...)

		var id int64

		if err := row.Scan(&id); err != nil {
			return errors.Err(err)
		}
		m.SetPrimary(id)
	}
	return nil
}

// Update the given models in the given table. This expects the models given to
// share the same column for the primary key.
func (s Store) Update(table string, mm ...Model) error {
	for _, m := range mm {
		modelVals := m.Values()
		col, val := m.Primary()

		opts := []query.Option{query.Table(table)}

		for k, v := range modelVals {
			if k == col {
				continue
			}
			opts = append(opts, query.Set(k, v))
		}

		opts = append(opts, query.Where(col, "=", val))

		q := query.Update(opts...)

		stmt, err := s.Prepare(q.Build())

		if err != nil {
			return errors.Err(err)
		}

		defer stmt.Close()

		if _, err := stmt.Exec(q.Args()...); err != nil {
			return errors.Err(err)
		}
	}
	return nil
}

// Delete all the given models from the given table. This expects the models
// given to share the same column for the primary key.
func (s Store) Delete(table string, mm ...Model) error {
	if len(mm) == 0 {
		return nil
	}

	peek := mm[0]
	col, _ := peek.Primary()

	ids := make([]interface{}, 0, len(mm))

	for _, m := range mm {
		_, val := m.Primary()
		ids = append(ids, val)
	}

	q := query.Delete(query.From(table), query.Where(col, "IN", ids...))

	stmt, err := s.Prepare(q.Build())

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	_, err = stmt.Exec(q.Args()...)
	return errors.Err(err)
}

// Paginate the records in the table and return the paginator for the given
// page. The returned struct contains information about the paginated data, but
// not the data itself. It is expected for a subsequent All call to be made
// using the paginator information to get the desired data.
func (s Store) Paginate(table string, page int64, opts ...query.Option) (Paginator, error) {
	if page <= 0 {
		page = 1
	}

	p := Paginator{Page: page}

	opts = append([]query.Option{
		query.Count("*"),
		query.From(table),
	}, opts...)

	q := query.Select(opts...)

	stmt, err := s.Prepare(q.Build())

	if err != nil {
		return p, errors.Err(err)
	}

	defer stmt.Close()

	var count int64

	if err := stmt.QueryRow(q.Args()...).Scan(&count); err != nil {
		return p, errors.Err(err)
	}

	pages := int64(count) / int64(PageLimit)

	if count % PageLimit != 0 {
		pages++
	}

	if p.Page > count {
		p.Page = pages
	}

	if p.Page != 0 {
		p.Offset = (p.Page - 1) * PageLimit
	}

	for i := int64(0); i < pages; i++ {
		p.Pages = append(p.Pages, i + 1)
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
