package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/mattn/go-sqlite3"
	_ "github.com/mattn/go-sqlite3"
	urlstore "github.com/sajoniks/GoShort/internal/store/interface"
	"github.com/sajoniks/GoShort/internal/trace"
)

type sqliteUrlStore struct {
	db *sql.DB
}

func (s *sqliteUrlStore) Close() {
	s.db.Close()
}

func NewSqliteStore(connString string) (urlstore.CloseableStore, error) {
	db, err := sql.Open("sqlite3", connString)
	if err != nil {
		return nil, trace.WrapError(err)
	}

	stmt, err := db.Prepare(`
		CREATE TABLE IF NOT EXISTS urls(
    		id INTEGER PRIMARY KEY AUTOINCREMENT,
    		alias TEXT NOT NULL,
    		url TEXT NOT NULL,
    		UNIQUE (alias, url));
		CREATE INDEX IF NOT EXISTS idx_alias ON urls(alias);
	`)
	if err != nil {
		return nil, trace.WrapError(err)
	}

	_, err = stmt.Exec()
	if err != nil {
		return nil, trace.WrapError(err)
	}

	return &sqliteUrlStore{db: db}, nil
}

func (s *sqliteUrlStore) SaveURL(src string, alias string) (string, error) {
	stmt, err := s.db.Prepare(`INSERT INTO urls (alias, url) VALUES (?, ?)`)
	if err != nil {
		return "", trace.WrapError(err)
	}
	res, err := stmt.Exec(alias, src)
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && errors.Is(sqliteErr.ExtendedCode, sqlite3.ErrConstraintUnique) {
			return "", trace.WrapError(urlstore.ErrUrlExists)
		}
		return "", trace.WrapError(err)
	}

	return fmt.Sprint(res.LastInsertId()), nil
}

func (s *sqliteUrlStore) GetURL(alias string) (string, error) {
	stmt, err := s.db.Prepare(`SELECT url FROM urls WHERE alias = ?`)
	if err != nil {
		return "", trace.WrapError(err)
	}
	var resultUrl string
	err = stmt.QueryRow(alias).Scan(&resultUrl)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", trace.WrapError(urlstore.ErrUrlNotFound)
		}
		return "", trace.WrapError(err)
	}

	return resultUrl, nil
}
