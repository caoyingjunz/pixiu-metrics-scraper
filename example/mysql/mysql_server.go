package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"

	sideapi "github.com/kubernetes-sigs/dashboard-metrics-scraper/pkg/api"
	sidedb "github.com/kubernetes-sigs/dashboard-metrics-scraper/pkg/database"
)

// MySQL 连接配置
const (
	mysqlUser     = "root"
	mysqlPassword = "root"
	mysqlHost     = "localhost"
	mysqlPort     = "3306"
	mysqlDatabase = "pixiu_metrics"
)

func main() {
	// 构建 MySQL DSN
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=UTC",
		mysqlUser,
		mysqlPassword,
		mysqlHost,
		mysqlPort,
		mysqlDatabase,
	)

	// 连接 MySQL 数据库
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Unable to open MySQL database: %s", err)
	}
	defer db.Close()

	// 设置数据库连接池
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	// 测试连接
	if err := db.Ping(); err != nil {
		log.Fatalf("Unable to ping MySQL database: %s", err)
	}

	log.Info("Successfully connected to MySQL database")

	// 初始化数据库表（使用 MySQL 后端）
	backend := sidedb.BackendFactory("mysql")
	err = backend.CreateDatabase(db)
	if err != nil {
		log.Fatalf("Unable to initialize MySQL database tables: %s", err)
	}

	// 插入测试数据
	err = insertTestData(db)
	if err != nil {
		log.Fatalf("Unable to insert test data: %s", err)
	}

	log.Info("Test data inserted successfully")

	// 启动 HTTP 服务器
	r := mux.NewRouter()
	sideapi.Manager(r, db)

	log.Info("Starting test server on :8080")
	log.Fatal(http.ListenAndServe(":8080", handlers.CombinedLoggingHandler(os.Stdout, r)))
}

func insertTestData(db *sql.DB) error {
	now := time.Now().UTC()

	// 插入 node 测试数据
	nodeData := []struct {
		uid     string
		name    string
		cpu     int64
		memory  int64
		storage int64
		time    time.Time
	}{
		{"node-uid-1", "node-1", 100, 512000, 1024000, now.Add(-30 * time.Minute)},
		{"node-uid-1", "node-1", 200, 624000, 1024000, now.Add(-25 * time.Minute)},
		{"node-uid-1", "node-1", 150, 576000, 1024000, now.Add(-20 * time.Minute)},
		{"node-uid-1", "node-1", 300, 700000, 1024000, now.Add(-15 * time.Minute)},
		{"node-uid-1", "node-1", 250, 650000, 1024000, now.Add(-10 * time.Minute)},
		{"node-uid-1", "node-1", 180, 590000, 1024000, now.Add(-5 * time.Minute)},
		{"node-uid-1", "node-1", 220, 620000, 1024000, now},

		{"node-uid-2", "node-2", 120, 480000, 1024000, now.Add(-30 * time.Minute)},
		{"node-uid-2", "node-2", 180, 520000, 1024000, now.Add(-20 * time.Minute)},
		{"node-uid-2", "node-2", 220, 580000, 1024000, now.Add(-10 * time.Minute)},
		{"node-uid-2", "node-2", 190, 550000, 1024000, now},
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	for _, data := range nodeData {
		_, err = tx.Exec(
			"insert into nodes (uid, name, cpu, memory, storage, time) values(?, ?, ?, ?, ?, ?)",
			data.uid, data.name, data.cpu, data.memory, data.storage, data.time,
		)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	// 插入 pod 测试数据
	podData := []struct {
		uid       string
		namespace string
		name      string
		container string
		cpu       int64
		memory    int64
		storage   int64
		time      time.Time
	}{
		{"pod-uid-1", "default", "pod-1", "container-1", 50, 128000, 256000, now.Add(-30 * time.Minute)},
		{"pod-uid-1", "default", "pod-1", "container-1", 60, 135000, 256000, now.Add(-25 * time.Minute)},
		{"pod-uid-1", "default", "pod-1", "container-1", 55, 130000, 256000, now.Add(-20 * time.Minute)},
		{"pod-uid-1", "default", "pod-1", "container-1", 70, 140000, 256000, now.Add(-15 * time.Minute)},
		{"pod-uid-1", "default", "pod-1", "container-1", 65, 138000, 256000, now.Add(-10 * time.Minute)},
		{"pod-uid-1", "default", "pod-1", "container-1", 58, 132000, 256000, now.Add(-5 * time.Minute)},
		{"pod-uid-1", "default", "pod-1", "container-1", 62, 136000, 256000, now},

		{"pod-uid-2", "default", "pod-2", "container-1", 40, 100000, 200000, now.Add(-30 * time.Minute)},
		{"pod-uid-2", "default", "pod-2", "container-1", 45, 110000, 200000, now.Add(-20 * time.Minute)},
		{"pod-uid-2", "default", "pod-2", "container-1", 50, 120000, 200000, now.Add(-10 * time.Minute)},
		{"pod-uid-2", "default", "pod-2", "container-1", 48, 115000, 200000, now},

		{"kube-pod-uid", "kube-system", "kube-pod", "container-1", 30, 80000, 160000, now.Add(-15 * time.Minute)},
		{"kube-pod-uid", "kube-system", "kube-pod", "container-1", 35, 85000, 160000, now},
	}

	for _, data := range podData {
		_, err = tx.Exec(
			"insert into pods (uid, namespace, name, container, cpu, memory, storage, time) values(?, ?, ?, ?, ?, ?, ?, ?)",
			data.uid, data.namespace, data.name, data.container, data.cpu, data.memory, data.storage, data.time,
		)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	fmt.Println("Test data inserted:")
	fmt.Printf("  - %d node records\n", len(nodeData))
	fmt.Printf("  - %d pod records\n", len(podData))
	fmt.Printf("  - Time range: %s to %s\n",
		now.Add(-30*time.Minute).Format("2006-01-02T15:04:05Z"),
		now.Format("2006-01-02T15:04:05Z"))

	return nil
}
