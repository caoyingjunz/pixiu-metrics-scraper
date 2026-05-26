package database

import (
	"database/sql"
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

type MySQLBackend struct{}

func (m *MySQLBackend) CreateDatabase(db *sql.DB) error {
	// MySQL does not support multiple statements in a single Exec call by default
	// Execute each CREATE TABLE statement separately

	// Create nodes table
	_, err := db.Exec(`create table if not exists nodes (uid varchar(255), name varchar(255), cpu varchar(255), memory varchar(255), storage varchar(255), time datetime)`)
	if err != nil {
		return err
	}

	// Create pods table
	_, err = db.Exec(`create table if not exists pods (uid varchar(255), name varchar(255), namespace varchar(255), container varchar(255), cpu varchar(255), memory varchar(255), storage varchar(255), time datetime)`)
	if err != nil {
		return err
	}

	return nil
}

func (m *MySQLBackend) UpdateDatabase(db *sql.DB, nodeMetrics *v1beta1.NodeMetricsList, podMetrics *v1beta1.PodMetricsList) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare("insert into nodes(uid, name, cpu, memory, storage, time) values(?, ?, ?, ?, ?, NOW())")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, v := range nodeMetrics.Items {
		_, err = stmt.Exec(v.UID, v.Name, v.Usage.Cpu().MilliValue(), v.Usage.Memory().MilliValue()/1000, v.Usage.StorageEphemeral().MilliValue()/1000)
		if err != nil {
			return err
		}
	}

	stmt, err = tx.Prepare("insert into pods(uid, name, namespace, container, cpu, memory, storage, time) values(?, ?, ?, ?, ?, ?, ?, NOW())")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, v := range podMetrics.Items {
		for _, u := range v.Containers {
			_, err = stmt.Exec(v.UID, v.Name, v.Namespace, u.Name, u.Usage.Cpu().MilliValue(), u.Usage.Memory().MilliValue()/1000, u.Usage.StorageEphemeral().MilliValue()/1000)
			if err != nil {
				return err
			}
		}
	}

	err = tx.Commit()

	if err != nil {
		rberr := tx.Rollback()
		if rberr != nil {
			return rberr
		}
		return err
	}

	return nil
}

func (m *MySQLBackend) CullDatabase(db *sql.DB, window *time.Duration) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	cutoffTime := time.Now().UTC().Add(-*window)

	nodestmt, err := tx.Prepare("delete from nodes where time <= ?;")
	if err != nil {
		return err
	}

	defer nodestmt.Close()
	res, err := nodestmt.Exec(cutoffTime)
	if err != nil {
		return err
	}

	affected, _ := res.RowsAffected()
	log.Debugf("Cleaning up nodes: %d rows removed", affected)

	podstmt, err := tx.Prepare("delete from pods where time <= ?;")
	if err != nil {
		return err
	}

	defer podstmt.Close()
	res, err = podstmt.Exec(cutoffTime)
	if err != nil {
		return err
	}

	affected, _ = res.RowsAffected()
	log.Debugf("Cleaning up pods: %d rows removed", affected)
	err = tx.Commit()

	if err != nil {
		rberr := tx.Rollback()
		if rberr != nil {
			return rberr
		}
		return err
	}

	return nil
}
