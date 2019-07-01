package model

import (
	"bytes"
	"fmt"
	"strings"
)

type Statement uint8

type Option func(q Query) Query

type order struct {
	columns   []string
	direction string
}

type where struct {
	col string
	op  string
	val interface{}
	cat string
}

type set struct {
	col string
	val interface{}
}

type Query struct {
	statement Statement
	table     string
	columns   []string
	wheres    []where
	sets      []set
	order     order
	limit     int
	returning []string
	args      []interface{}
}

const (
	SELECT Statement = iota
	INSERT
	UPDATE
	DELETE
)

func Returning(cols ...string) Option {
	return func(q Query) Query {
		if q.statement == INSERT || q.statement == UPDATE || q.statement == DELETE {
			q.returning = append(q.returning, cols...)
		}

		return q
	}
}

func Select(opts ...Option) Query {
	q := Query{
		statement: SELECT,
	}

	for _, opt := range opts {
		q = opt(q)
	}

	return q
}

func Insert(opts ...Option) Query {
	q := Query{
		statement: INSERT,
	}

	for _, opt := range opts {
		q = opt(q)
	}

	return q
}

func Update(opts ...Option) Query {
	q := Query{
		statement: UPDATE,
	}

	for _, opt := range opts {
		q = opt(q)
	}

	return q
}

func Delete(opts ...Option) Query {
	q := Query{
		statement: DELETE,
	}

	for _, opt := range opts {
		q = opt(q)
	}

	return q
}

func Values(vals ...interface{}) Option {
	return func(q Query) Query {
		if q.statement == INSERT {
			q.args = vals
		}

		return q
	}
}

func Columns(cols ...string) Option {
	return func(q Query) Query {
		q.columns = cols

		return q
	}
}

func Limit(l int) Option {
	return func(q Query) Query {
		if q.statement == SELECT {
			q.limit = l
		}

		return q
	}
}

// Or will apply all given options to the query, and modify any newly added
// WHERE clauses to be treated as an OR, instead of the default AND.
func Or(opts ...Option) Option {
	return func(q Query) Query {
		l := len(q.wheres)

		for _, opt := range opts {
			q = opt(q)
		}

		diff := len(q.wheres) - l

		if l == diff {
			return q
		}

		changed := make([]where, 0)

		for i, w := range q.wheres {
			if i >= l {
				w.cat = " OR "
			}

			changed = append(changed, w)
		}

		q.wheres = changed

		return q
	}
}

func OrderAsc(cols ...string) Option {
	return func(q Query) Query {
		q.order.columns = cols
		q.order.direction = "ASC"

		return q
	}
}

func OrderDesc(cols ...string) Option {
	return func(q Query) Query {
		q.order.columns = cols
		q.order.direction = "DESC"

		return q
	}
}

func Set(col string, val interface{}) Option {
	return func(q Query) Query {
		q = SetRaw(col, fmt.Sprintf("$%d", len(q.args) + 1))(q)
		q.args = append(q.args, val)

		return q
	}
}

func SetRaw(col string, val interface{}) Option {
	return func(q Query) Query {
		if q.statement != UPDATE {
			return q
		}

		s := set{
			col: col,
			val: val,
		}

		q.sets = append(q.sets, s)

		return q
	}
}

func Table(table string) Option {
	return func(q Query) Query {
		q.table = table

		return q
	}
}

func WhereEq(col string, val interface{}) Option {
	return func(q Query) Query {
		w := where{
			col: col,
			op:  "=",
			val: fmt.Sprintf("$%d", len(q.args) + 1),
			cat: " AND ",
		}

		q.wheres = append(q.wheres, w)
		q.args = append(q.args, val)

		return q
	}
}

func WhereIn(col string, vals ...interface{}) Option {
	return func(q Query) Query {
		if len(vals) == 0 {
			return q
		}

		in := make([]string, len(vals), len(vals))
		larg := len(q.args)

		for i := range vals {
			in[i] = fmt.Sprintf("$%d", larg + i + 1)
		}

		val := "(" + strings.Join(in, ", ") + ")"

		w := where{
			col: col,
			op:  "IN",
			val: val,
			cat: " AND ",
		}

		q.wheres = append(q.wheres, w)
		q.args = append(q.args, vals...)

		return q
	}
}

func WhereIs(col string, val interface{}) Option {
	return func(q Query) Query {
		w := where{
			col: col,
			op:  "IS",
			val: val,
			cat: "AND",
		}

		q.wheres = append(q.wheres, w)

		return q
	}
}

func WhereInQuery(col string, q1 Query) Option {
	return func(q2 Query) Query {
		val := "(" + q1.Build() + ")"

		w := where{
			col: col,
			op:  "IN",
			val: val,
			cat: " AND ",
		}

		q2.wheres = append(q2.wheres, w)
		q2.args = append(q2.args, q1.Args()...)

		return q2
	}
}

func WhereLike(col string, val interface{}) Option {
	return func(q Query) Query {
		w := where{
			col: col,
			op:  "LIKE",
			val: fmt.Sprintf("$%d", len(q.args) + 1),
			cat: " AND ",
		}

		q.wheres = append(q.wheres, w)
		q.args = append(q.args, val)

		return q
	}
}

func (q Query) buildReturning() string {
	if len(q.returning) == 0 {
		return ""
	}

	return " RETURNING " + strings.Join(q.returning, ", ")
}

func (q Query) buildWheres() string {
	if len(q.wheres) == 0 {
		return ""
	}

	buf := bytes.NewBufferString(" WHERE ")

	wheres := make([]string, 0)
	end := len(q.wheres) - 1

	for i, w := range q.wheres {
		wheres = append(wheres, fmt.Sprintf("%s %s %v", w.col, w.op, w.val))

		if i != end {
			next := q.wheres[i + 1]

			if next.cat != w.cat {
				buf.WriteString("(" + strings.Join(wheres, w.cat) + ")" + next.cat)
				wheres = make([]string, 0)
			}

			continue
		}

		buf.WriteString(strings.Join(wheres, w.cat))
	}

	return buf.String()
}

func (q Query) buildSelect() string {
	buf := bytes.NewBufferString("SELECT ")

	buf.WriteString(strings.Join(q.columns, ", "))
	buf.WriteString(" FROM " + q.table)
	buf.WriteString(q.buildWheres())

	if len(q.order.columns) > 0 && q.order.direction != "" {
		fmt.Fprintf(buf, " ORDER BY %s %s", strings.Join(q.order.columns, ", "), q.order.direction)
	}

	if q.limit > 0 {
		fmt.Fprintf(buf, " LIMIT %d", q.limit)
	}

	return buf.String()
}

func (q Query) buildInsert() string {
	buf := &bytes.Buffer{}

	fmt.Fprintf(buf, "INSERT INTO %s (%s)", q.table, strings.Join(q.columns, ", "))

	vals := make([]string, len(q.columns), len(q.columns))

	for i := range q.columns {
		vals[i] = fmt.Sprintf("$%d", i + 1)
	}

	fmt.Fprintf(buf, " VALUES (%s)", strings.Join(vals, ", "))
	buf.WriteString(q.buildReturning())

	return buf.String()
}

func (q Query) buildUpdate() string {
	buf := bytes.NewBufferString("UPDATE " + q.table)

	sets := make([]string, 0)

	for _, s := range q.sets {
		sets = append(sets, fmt.Sprintf("%s = %v", s.col, s.val))
	}

	buf.WriteString(" SET ")
	buf.WriteString(strings.Join(sets, ", "))
	buf.WriteString(q.buildWheres())
	buf.WriteString(q.buildReturning())

	return buf.String()
}

func (q Query) buildDelete() string {
	buf := bytes.NewBufferString("DELETE FROM " + q.table)
	buf.WriteString(q.buildWheres())
	buf.WriteString(q.buildReturning())

	return buf.String()
}

func (q Query) Args() []interface{} {
	return q.args
}

func (q Query) Build() string {
	switch q.statement {
		case SELECT:
			return q.buildSelect()
		case INSERT:
			return q.buildInsert()
		case UPDATE:
			return q.buildUpdate()
		case DELETE:
			return q.buildDelete()
		default:
			return ""
	}
}
