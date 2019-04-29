package model

import (
	"fmt"
	"net"
	"time"

	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/errors"

	"github.com/jmoiron/sqlx"

	_ "github.com/lib/pq"
)

var sourceFmt = "host=%s port=%s dbname=%s user=%s password=%s sslmode=disable"

type model struct {
	*sqlx.DB

	ID        int64      `db:"id"`
	CreatedAt *time.Time `db:"created_at"`
	UpdatedAt *time.Time `db:"updated_at"`
}

type Store struct {
	*sqlx.DB
}

func NewStore(db *sqlx.DB) *Store {
	return &Store{
		DB: db,
	}
}

func Connect(addr, dbname, username, password string) (*sqlx.DB, error) {
	host, port, err := net.SplitHostPort(addr)

	if err != nil {
		return nil, errors.Err(err)
	}

	source := fmt.Sprintf(sourceFmt, host, port, dbname, username, password)

	log.Debug.Printf("opening postgresql connection with '%s'\n", source)

	db, err := sqlx.Open("postgres", source)

	if err != nil {
		return nil, errors.Err(err)
	}

	log.Debug.Println("testing connection to database")

	return db, errors.Err(db.Ping())
}
