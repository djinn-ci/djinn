package database

import (
	"context"
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"djinn-ci.com/errors"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil/v2"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

var (
	ErrNoRows     = pgx.ErrNoRows
	ErrExists     = errors.New("database: record exists")
	ErrPermission = errors.New("database: permission denied")
)

type (
	Tx   = pgx.Tx
	Pool = pgxpool.Pool

	Value = driver.Value
)

// Any is a catch-all type for scanning in any arbitrary value from a query.
type Any struct {
	Value any
}

func (a *Any) Scan(v any) error {
	a.Value = v
	return nil
}

type Bytea []byte

func (b Bytea) String() string {
	return hex.EncodeToString(b)
}

func (b Bytea) MarshalJSON() ([]byte, error) {
	if len(b) == 0 {
		return []byte("null"), nil
	}
	return json.Marshal(b.String())
}

// Null represents a nullable type of T. This would represent columns in a
// table that can be NULL.
type Null[T any] struct {
	Elem  T
	Valid bool
}

func (n Null[T]) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}

	b, err := json.Marshal(n.Elem)

	if err != nil {
		return nil, errors.Err(err)
	}
	return b, nil
}

func (n *Null[T]) Scan(v any) error {
	if v == nil {
		n.Valid = false
		return nil
	}

	el, ok := v.(T)

	if !ok {
		return fmt.Errorf("database: Null[%T].Scan: cannot type assert %T to %T", n.Elem, v, n.Elem)
	}

	n.Elem = el
	n.Valid = true

	return nil
}

func (n Null[T]) Value() (Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Elem, nil
}

func Connect(ctx context.Context, dsn string) (*Pool, error) {
	pool, err := pgxpool.Connect(ctx, dsn)

	if err != nil {
		return nil, errors.Err(err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, errors.Err(err)
	}
	return pool, nil
}

type ParamMode uint

const (
	ModeCreate ParamMode = 1 << iota
	ModeUpdate
)

type Param struct {
	Mode  ParamMode
	Value any
}

func ImmutableParam(v any) Param {
	return Param{
		Value: v,
	}
}

func CreateOnlyParam(v any) Param {
	return Param{
		Mode:  ModeCreate,
		Value: v,
	}
}

func UpdateOnlyParam(v any) Param {
	return Param{
		Mode:  ModeUpdate,
		Value: v,
	}
}

func CreateUpdateParam(v any) Param {
	return Param{
		Mode:  ModeCreate | ModeUpdate,
		Value: v,
	}
}

func (p Param) Has(mode ParamMode) bool {
	return (p.Mode & mode) == mode
}

type Params map[string]Param

// Only deletes every parameter in the underlying map, except for those given.
// If names is of length 1 and contains only "*", then no parameters are
// deleted.
func (p Params) Only(names ...string) {
	if len(names) == 1 {
		if names[0] == "*" {
			return
		}
	}

	set := make(map[string]struct{})

	for _, name := range names {
		set[name] = struct{}{}
	}

	for name := range p {
		if _, ok := set[name]; !ok {
			delete(p, name)
		}
	}
}

type Row struct {
	Columns []string
	Scan    func(dest ...any) error
}

func Scan(r *Row, valtab map[string]any) error {
	dest := make([]any, 0, len(valtab))

	for _, col := range r.Columns {
		if p, ok := valtab[col]; ok {
			dest = append(dest, p)
		}
	}

	if err := r.Scan(dest...); err != nil {
		return errors.Err(err)
	}
	return nil
}

type Model interface {
	Primary() (string, any)

	Scan(r *Row) error

	Params() Params

	Bind(m Model)

	MarshalJSON() ([]byte, error)

	Endpoint(elems ...string) string
}

func Map[M Model, T any](mm []M, fn func(M) T) []T {
	vals := make([]T, 0, len(mm))

	for _, m := range mm {
		vals = append(vals, fn(m))
	}
	return vals
}

type Loader interface {
	Load(ctx context.Context, from, to string, mm ...Model) error
}

type Relation struct {
	From   string
	To     string
	Loader Loader
}

func LoadRelations[M Model](ctx context.Context, mm []M, relations ...Relation) error {
	models := make([]Model, 0)

	for _, m := range mm {
		models = append(models, m)
	}

	for _, rel := range relations {
		if err := rel.Loader.Load(ctx, rel.From, rel.To, models...); err != nil {
			return errors.Err(err)
		}
	}
	return nil
}

func ModelLoader(pool *Pool, table string, new func() Model) Loader {
	return NewStore[Model](pool, table, new)
}

func Search(col, pattern string) query.Option {
	return func(q query.Query) query.Query {
		if pattern == "" {
			return q
		}
		return query.Where("LOWER("+col+")", "LIKE", query.Arg("%"+strings.ToLower(pattern)+"%"))(q)
	}
}

type Store[M Model] struct {
	*Pool

	table string
	new   func() M
}

func NewStore[M Model](pool *Pool, table string, new func() M) *Store[M] {
	return &Store[M]{
		Pool:  pool,
		table: table,
		new:   new,
	}
}

func FieldUnique[M Model](s *Store[M], col string) webutil.ValidatorFunc {
	return func(ctx context.Context, val any) error {
		_, ok, err := s.SelectOne(ctx, []string{col}, query.Where(col, "=", query.Arg(val)))

		if err != nil {
			return errors.Err(err)
		}

		if ok {
			return ErrExists
		}
		return nil
	}
}

func FieldUniqueExcept[M Model](s *Store[M], col string, except any) webutil.ValidatorFunc {
	return func(ctx context.Context, val any) error {
		_, ok, err := s.SelectOne(
			ctx,
			[]string{col},
			query.Where(col, "=", query.Arg(val)),
			query.Where(col, "!=", query.Arg(except)),
		)

		if err != nil {
			return err
		}

		if ok {
			return ErrExists
		}
		return nil
	}
}

func (s *Store[M]) makeRow(rows pgx.Rows) Row {
	descs := rows.FieldDescriptions()
	cols := make([]string, 0, len(descs))

	for _, desc := range descs {
		cols = append(cols, string(desc.Name))
	}
	return Row{Columns: cols}
}

func (s *Store[M]) SelectOne(ctx context.Context, cols []string, opts ...query.Option) (M, bool, error) {
	opts = append([]query.Option{
		query.From(s.table),
	}, opts...)

	q := query.Select(query.Columns(cols...), opts...)

	var zero M

	rows, err := s.Query(ctx, q.Build(), q.Args()...)

	if err != nil {
		return zero, false, errors.Err(err)
	}

	defer rows.Close()

	if err := rows.Err(); err != nil {
		return zero, false, errors.Err(err)
	}

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return zero, false, errors.Err(err)
		}
		return zero, false, nil
	}

	row := s.makeRow(rows)
	row.Scan = rows.Scan

	m := s.new()

	if err := m.Scan(&row); err != nil {
		return zero, false, errors.Err(err)
	}
	return m, true, nil
}

func (s *Store[M]) Distinct(ctx context.Context, expr, cols []string, opts ...query.Option) ([]M, error) {
	opts = append([]query.Option{
		query.From(s.table),
	}, opts...)

	q := query.SelectDistinctOn(expr, query.Columns(cols...), opts...)

	rows, err := s.Query(ctx, q.Build(), q.Args()...)

	if err != nil {
		return nil, errors.Err(err)
	}

	mm := make([]M, 0)

	row := s.makeRow(rows)

	for rows.Next() {
		row.Scan = rows.Scan

		m := s.new()

		if err := m.Scan(&row); err != nil {
			return nil, errors.Err(err)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Err(err)
	}
	return mm, nil
}

func (s *Store[M]) Select(ctx context.Context, cols []string, opts ...query.Option) ([]M, error) {
	opts = append([]query.Option{
		query.From(s.table),
	}, opts...)

	q := query.Select(query.Columns(cols...), opts...)

	rows, err := s.Query(ctx, q.Build(), q.Args()...)

	if err != nil {
		return nil, errors.Err(err)
	}

	defer rows.Close()

	mm := make([]M, 0)

	row := s.makeRow(rows)

	for rows.Next() {
		row.Scan = rows.Scan

		m := s.new()

		if err := m.Scan(&row); err != nil {
			return nil, errors.Err(err)
		}
		mm = append(mm, m)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Err(err)
	}
	return mm, nil
}

func (s *Store[M]) Get(ctx context.Context, opts ...query.Option) (M, bool, error) {
	m, ok, err := s.SelectOne(ctx, []string{"*"}, opts...)

	if err != nil {
		return m, false, errors.Err(err)
	}
	return m, ok, nil
}

func (s *Store[M]) All(ctx context.Context, opts ...query.Option) ([]M, error) {
	mm, err := s.Select(ctx, []string{"*"}, opts...)

	if err != nil {
		return nil, errors.Err(err)
	}
	return mm, nil
}

func value(val any) any {
	switch v := val.(type) {
	case Null[int64]:
		return v.Elem
	case Null[string]:
		return v.Elem
	case Null[time.Time]:
		return v.Elem
	}
	return val
}

func (s *Store[M]) Load(ctx context.Context, from, to string, mm ...M) error {
	if len(mm) == 0 {
		return nil
	}

	vals := Map[M, any](mm, func(m M) any {
		param := m.Params()[from]
		return param.Value
	})

	loaded, err := s.All(ctx, query.Where(to, "IN", query.List(vals...)))

	if err != nil {
		return errors.Err(err)
	}

	tab := make(map[any][]M)

	for _, m := range loaded {
		param := m.Params()[to]

		tab[param.Value] = append(tab[value(param.Value)], m)
	}

	for _, m := range mm {
		param := m.Params()[from]

		for _, relation := range tab[value(param.Value)] {
			m.Bind(relation)
		}
	}
	return nil
}

type Paginator[M Model] struct {
	limit  int
	offset int // offset when querying records
	pos    int // position in the Pages slice

	Items []M
	Pages []int
}

func (p *Paginator[M]) EncodeToLink(url *url.URL) string {
	q := url.Query()
	q.Set("page", strconv.Itoa(p.Prev()))

	url.RawQuery = q.Encode()

	prev := url.String()

	q.Set("page", strconv.Itoa(p.Next()))

	url.RawQuery = q.Encode()

	next := url.String()

	return `<` + prev + `>; rel="prev", <` + next + `>; rel="next"`
}

// Set changes the paginator's position to be that of the given page. If the
// given page cannot be found in the Pages slice, then the position is not
// changed.
func (p *Paginator[M]) Set(page int) {
	i := sort.Search(len(p.Pages), func(i int) bool {
		return p.Pages[i] >= page
	})

	if i != len(p.Pages) {
		p.pos = i
	}
}

func (p *Paginator[M]) Prev() int {
	if len(p.Pages) == 0 {
		return 0
	}

	pos := p.pos - 1

	if pos < 0 {
		pos = 0
	}
	return p.Pages[pos]
}

func (p *Paginator[M]) Page() int {
	if len(p.Pages) == 0 {
		return 0
	}
	return p.Pages[p.pos]
}

func (p *Paginator[M]) Next() int {
	if len(p.Pages) == 0 {
		return 0
	}

	pos := p.pos + 1

	if pos > len(p.Pages)-1 {
		pos = len(p.Pages) - 1
	}
	return p.Pages[pos]
}

func (p *Paginator[M]) Load(ctx context.Context, s *Store[M], opts ...query.Option) error {
	opts = append(opts, query.Limit(int64(p.limit)), query.Offset(int64(p.offset)))

	items, err := s.All(ctx, opts...)

	if err != nil {
		return errors.Err(err)
	}

	p.Items = items
	return nil
}

const PageLimit = 25

func (s *Store[M]) Paginate(ctx context.Context, page, limit int, opts ...query.Option) (*Paginator[M], error) {
	if page <= 0 {
		page = 1
	}

	opts = append([]query.Option{
		query.From(s.table),
	}, opts...)

	q := query.Select(query.Count("*"), opts...)

	var count int

	if err := s.QueryRow(ctx, q.Build(), q.Args()...).Scan(&count); err != nil {
		return nil, errors.Err(err)
	}

	p := Paginator[M]{
		limit: limit,
	}

	pages := count / limit

	if count%limit != 0 {
		pages++
	}

	if page > count {
		page = pages
	}
	if page != 0 {
		p.offset = (page - 1) * limit
	}

	p.Pages = make([]int, 0, pages)

	for i := 0; i < pages; i++ {
		if i+1 == page {
			p.pos = i
		}
		p.Pages = append(p.Pages, i+1)
	}
	return &p, nil
}

type queryFunc func(context.Context, string, ...any) (pgx.Rows, error)

func (s *Store[M]) doCreate(ctx context.Context, doQuery queryFunc, mm ...M) error {
	opts := make([]query.Option, 0, len(mm)+1)

	cols := make([]string, 0)
	vals := make([]any, 0)

	ret := ""

	for i, m := range mm {
		params := m.Params()

		for name, param := range params {
			if param.Has(ModeCreate) {
				if i == 0 {
					cols = append(cols, name)

					col, _ := m.Primary()
					ret = col
				}
				vals = append(vals, param.Value)
			}
		}

		opts = append(opts, query.Values(vals...))
		vals = vals[0:0]
	}

	if ret != "" {
		opts = append(opts, query.Returning(ret))
	}

	q := query.Insert(s.table, query.Columns(cols...), opts...)

	rows, err := doQuery(ctx, q.Build(), q.Args()...)

	if err != nil {
		return errors.Err(err)
	}

	row := s.makeRow(rows)
	i := 0

	for rows.Next() {
		row.Scan = rows.Scan

		if err := mm[i].Scan(&row); err != nil {
			return errors.Err(err)
		}
	}
	return nil
}

func (s *Store[M]) Create(ctx context.Context, mm ...M) error {
	if err := s.doCreate(ctx, s.Query, mm...); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s *Store[M]) CreateTx(ctx context.Context, tx pgx.Tx, mm ...M) error {
	if err := s.doCreate(ctx, tx.Query, mm...); err != nil {
		return errors.Err(err)
	}
	return nil
}

type execFunc func(context.Context, string, ...any) (pgconn.CommandTag, error)

func (s *Store[M]) doUpdate(ctx context.Context, doExec execFunc, m M, opts ...query.Option) error {
	params := m.Params()

	setopts := make([]query.Option, 0, len(params))

	for name, param := range params {
		if param.Has(ModeUpdate) {
			setopts = append(setopts, query.Set(name, query.Arg(param.Value)))
		}
	}

	q := query.Update(s.table, append(setopts, opts...)...)

	if _, err := doExec(ctx, q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s *Store[M]) UpdateMany(ctx context.Context, m M, opts ...query.Option) error {
	if err := s.doUpdate(ctx, s.Exec, m, opts...); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s *Store[M]) Update(ctx context.Context, mm ...M) error {
	for _, m := range mm {
		col, val := m.Primary()

		if err := s.UpdateMany(ctx, m, query.Where(col, "=", query.Arg(val))); err != nil {
			return errors.Err(err)
		}
	}
	return nil
}

func (s *Store[M]) Delete(ctx context.Context, mm ...M) error {
	vals := Map[M, any](mm, func(m M) any {
		_, val := m.Primary()
		return val
	})

	col, _ := mm[0].Primary()

	q := query.Delete(s.table, query.Where(col, "IN", query.List(vals...)))

	if _, err := s.Exec(ctx, q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}
	return nil
}
