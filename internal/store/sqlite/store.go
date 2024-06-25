package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/mattn/go-sqlite3"
	_ "github.com/mattn/go-sqlite3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sajoniks/GoShort/internal/store/interface"
	"github.com/sajoniks/GoShort/internal/trace"
	"sync"
	"time"
)

type sqliteUrlStore struct {
	db      *sql.DB
	mx      sync.RWMutex
	metrics StoreMetricsService
}

func (s *sqliteUrlStore) Close() {
	s.db.Close()
}

type StoreMetricsService interface {
	RecordReadLockTime(d time.Duration)
	RecordWriteLockTime(d time.Duration)
}

type noOpSqliteMetrics struct {
}

func (n noOpSqliteMetrics) RecordReadLockTime(d time.Duration) {
}

func (n noOpSqliteMetrics) RecordWriteLockTime(d time.Duration) {
}

func NewNoOpMetrics() StoreMetricsService {
	return &noOpSqliteMetrics{}
}

type Metrics struct {
	lockTime *prometheus.GaugeVec
}

func (m *Metrics) RecordReadLockTime(d time.Duration) {
	m.lockTime.With(prometheus.Labels{"type": "read"}).Set(d.Seconds())
}

func (m *Metrics) RecordWriteLockTime(d time.Duration) {
	m.lockTime.With(prometheus.Labels{"type": "write"}).Set(d.Seconds())
}

func NewStoreMetrics(reg prometheus.Registerer) *Metrics {
	s := &Metrics{
		lockTime: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "persist",
			Subsystem: "sqlite3",
			Name:      "lock_wait_time",
			Help:      "amount of time waiting for lock be available",
		}, []string{"type"}),
	}
	reg.MustRegister(s.lockTime)
	return s
}

func NewSqliteStore(connString string, metrics StoreMetricsService) (urlstore.CloseableStore, error) {
	db, err := sql.Open("sqlite3", connString)
	if err != nil {
		return nil, trace.WrapError(err)
	}

	stmt, err := db.Prepare(`
		CREATE TABLE IF NOT EXISTS urls(
    		id INTEGER PRIMARY KEY AUTOINCREMENT,
    		alias TEXT NOT NULL,
    		url TEXT NOT NULL,
    		CHECK(trim(alias, ' ') <> '' AND trim(url, ' ') <> ''),
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

	s := &sqliteUrlStore{db: db, metrics: metrics}
	return s, nil
}

func (s *sqliteUrlStore) GetURL(alias string) (string, error) {

	t1 := time.Now()
	s.mx.RLock()
	defer s.mx.RUnlock()
	t2 := time.Since(t1)

	s.metrics.RecordReadLockTime(t2)

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

func (s *sqliteUrlStore) SaveURL(src, alias string) (string, error) {

	t1 := time.Now()
	s.mx.Lock()
	defer s.mx.Unlock()
	t2 := time.Since(t1)

	s.metrics.RecordWriteLockTime(t2)

	if len(alias) == 0 {
		return "", trace.WrapError(urlstore.ErrAliasEmpty)
	}
	if len(src) == 0 {
		return "", trace.WrapError(urlstore.ErrUrlEmpty)
	}
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
		if errors.As(err, &sqliteErr) && errors.Is(sqliteErr.ExtendedCode, sqlite3.ErrConstraintCheck) {
			return "", trace.WrapError(urlstore.ErrUrlExists)
		}
		return "", trace.WrapError(err)
	}

	return fmt.Sprint(res.LastInsertId()), nil
}
