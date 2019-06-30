package model

import (
	"testing"
)

type testQuery struct {
	expectedQuery string
	expectedArgs  []interface{}
	query         Query
}

func checkQueries(testQueries []testQuery, t *testing.T) {
	for _, tq := range testQueries {
		if tq.query.Build() != tq.expectedQuery {
			t.Fatalf(
				"query not as expected:\n\texpected = '%s'\n\tactual = '%s'",
				tq.expectedQuery,
				tq.query.Build(),
			)
		}

		if len(tq.query.Args()) != len(tq.expectedArgs) {
			t.Fatalf(
				"query args len not as expected:\n\texpected = '%d'\n\tactual = '%d'",
				len(tq.expectedArgs),
				len(tq.query.Args()),
			)
		}
	}
}

func TestSelect(t *testing.T) {
	testQueries := []testQuery{
		{
			"SELECT * FROM users WHERE username = $1",
			[]interface{}{"me"},
			Select(Columns("*"), Table("users"), WhereEq("username", "me")),
		},
		{
			"SELECT id, username FROM users WHERE id IN ($1, $2, $3, $4) AND username = $5",
			[]interface{}{10, 11, 12, 13, "me"},
			Select(
				Columns("id", "username"),
				Table("users"),
				WhereIn("id", 10, 11, 12, 13),
				WhereEq("username", "me"),
			),
		},
		{
			"SELECT * FROM posts WHERE user_id IN (SELECT id FROM users WHERE username = $1)",
			[]interface{}{"me"},
			Select(
				Columns("*"),
				Table("posts"),
				WhereInQuery("user_id",
					Select(
						Columns("id"),
						Table("users"),
						WhereEq("username", "me"),
					),
				),
			),
		},
		{
			"SELECT * FROM posts ORDER BY created_at, updated_at ASC",
			[]interface{}{},
			Select(
				Columns("*"),
				Table("posts"),
				OrderAsc("created_at", "updated_at"),
			),
		},
		{
			"SELECT * FROM posts WHERE user_id = $1 ORDER BY created_at DESC LIMIT 5",
			[]interface{}{10},
			Select(
				Columns("*"),
				Table("posts"),
				WhereEq("user_id", 10),
				OrderDesc("created_at"),
				Limit(5),
			),
		},
		{
			"SELECT * FROM users WHERE email = $1 OR username = $2",
			[]interface{}{"me@domain.com", "me"},
			Select(
				Columns("*"),
				Table("users"),
				Or(
					WhereEq("email", "me@domain.com"),
					WhereEq("username", "me"),
				),
			),
		},
		{
			"SELECT * FROM users WHERE (email = $1 OR username = $2) AND password = $3",
			[]interface{}{"me@domain.com", "me", "secret"},
			Select(
				Columns("*"),
				Table("users"),
				Or(
					WhereEq("email", "me@domain.com"),
					WhereEq("username", "me"),
				),
				WhereEq("password", "secret"),
			),
		},
	}

	checkQueries(testQueries, t)
}

func TestInsert(t *testing.T) {
	testQueries := []testQuery{
		{
			"INSERT INTO users (username, email, password) VALUES ($1, $2, $3)",
			[]interface{}{"me", "me@domain.com", "secret"},
			Insert(
				Columns("username", "email", "password"),
				Table("users"),
				Values("me", "me@domain.com", "secret"),
			),
		},
	}

	checkQueries(testQueries, t)
}

func TestUpdate(t *testing.T) {
	testQueries := []testQuery{
		{
			"UPDATE users SET username = $1 WHERE id = $2",
			[]interface{}{"me", 10},
			Update(
				Table("users"),
				Set("username", "me"),
				WhereEq("id", 10),
			),
		},
		{
			"UPDATE users SET username = $1, email = $2 WHERE id = $3",
			[]interface{}{"me", "me@domain.com", 10},
			Update(
				Table("users"),
				Set("username", "me"),
				Set("email", "me@domain.com"),
				WhereEq("id", 10),
			),
		},
		{
			"UPDATE users SET username = $1, updated_at = NOW() WHERE id = $2",
			[]interface{}{"me", 10},
			Update(
				Table("users"),
				Set("username", "me"),
				SetRaw("updated_at", "NOW()"),
				WhereEq("id", 10),
			),
		},
	}

	checkQueries(testQueries, t)
}

func TestDelete(t *testing.T) {
	testQueries := []testQuery{
		{
			"DELETE FROM users WHERE email = $1 AND username = $2",
			[]interface{}{"me@domain.com", "me"},
			Delete(
				Table("users"),
				WhereEq("email", "me@domain.com"),
				WhereEq("username", "me"),
			),
		},
	}

	checkQueries(testQueries, t)
}
