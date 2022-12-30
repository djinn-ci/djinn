package database

import (
	"context"

	"djinn-ci.com/errors"

	"github.com/andrewpillar/query"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

var ErrPermission = errors.New("permision denied")

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

type ParamMode uint

const (
	ModeCreate ParamMode = 1 << iota
	ModeUpdate
)

type Param struct {
	Mode  ParamMode
	Value any
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

type GModel interface {
	Primary() (string, any)

	Scan(r *Row) error

	Bind(m Model)

	JSON(addr string) map[string]any

	Endpoint(elem ...string) string

	Params() map[string]Param
}

type Store[M GModel] struct {
	*pgxpool.Pool

	table string
	new   func() M
}

func NewStore[M GModel](pool *pgxpool.Pool, table string, new func() M) *Store[M] {
	return &Store[M]{
		Pool:  pool,
		table: table,
		new:   new,
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

	if !rows.Next() {
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

func (s *Store[M]) Select(ctx context.Context, cols []string, opts ...query.Option) ([]M, error) {
	opts = append([]query.Option{
		query.From(s.table),
	}, opts...)

	q := query.Select(query.Columns(cols...), opts...)

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

func (s *Store[M]) Create(ctx context.Context, m M) error {
	params := m.Params()

	cols := make([]string, 0, len(params))
	vals := make([]any, 0, len(params))

	for name, param := range params {
		if param.Has(ModeCreate) {
			cols = append(cols, name)
			vals = append(vals, param.Value)
		}
	}

	col, _ := m.Primary()

	q := query.Insert(
		s.table,
		query.Columns(cols...),
		query.Values(vals...),
		query.Returning(col),
	)

	rows, err := s.Query(ctx, q.Build(), q.Args()...)

	if err != nil {
		return errors.Err(err)
	}

	if !rows.Next() {
		return pgx.ErrNoRows
	}

	row := s.makeRow(rows)
	row.Scan = rows.Scan

	if err := m.Scan(&row); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s *Store[M]) Update(ctx context.Context, m M) error {
	params := m.Params()

	opts := make([]query.Option, 0, len(params)+1)

	for name, param := range params {
		if param.Has(ModeUpdate) {
			opts = append(opts, query.Set(name, query.Arg(param.Value)))
		}
	}

	col, val := m.Primary()

	opts = append(opts, query.Where(col, "=", query.Arg(val)))

	q := query.Update(s.table, opts...)

	if _, err := s.Exec(ctx, q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s *Store[M]) Delete(ctx context.Context, m M) error {
	col, val := m.Primary()

	q := query.Delete(s.table, query.Where(col, "=", query.Arg(val)))

	if _, err := s.Exec(ctx, q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}
	return nil
}
