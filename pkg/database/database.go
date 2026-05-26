package database

import (
	"database/sql"
	"time"

	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

type DatabaseBackend interface {
	CreateDatabase(db *sql.DB) error
	UpdateDatabase(db *sql.DB, nodeMetrics *v1beta1.NodeMetricsList, podMetrics *v1beta1.PodMetricsList) error
	CullDatabase(db *sql.DB, window *time.Duration) error
}

func BackendFactory(driverName string) DatabaseBackend {
	switch driverName {
	case "mysql":
		return &MySQLBackend{}
	case "sqlite3":
		return &SQLiteBackend{}
	default:
		return &SQLiteBackend{}
	}
}

var DefaultBackend = &SQLiteBackend{}

func CreateDatabase(db *sql.DB) error {
	return DefaultBackend.CreateDatabase(db)
}

func UpdateDatabase(db *sql.DB, nodeMetrics *v1beta1.NodeMetricsList, podMetrics *v1beta1.PodMetricsList) error {
	return DefaultBackend.UpdateDatabase(db, nodeMetrics, podMetrics)
}

func CullDatabase(db *sql.DB, window *time.Duration) error {
	return DefaultBackend.CullDatabase(db, window)
}
