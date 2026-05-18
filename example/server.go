package example

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"

	sideapi "github.com/kubernetes-sigs/dashboard-metrics-scraper/pkg/api"
	sidedb "github.com/kubernetes-sigs/dashboard-metrics-scraper/pkg/database"
)

func test() {
	dbFile := "/tmp/test_metrics.db"

	// 删除旧的测试数据库
	os.Remove(dbFile)

	// 创建数据库连接
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		log.Fatalf("Unable to open Sqlite database: %s", err)
	}
	defer db.Close()

	// 初始化数据库表
	err = sidedb.CreateDatabase(db)
	if err != nil {
		log.Fatalf("Unable to initialize database tables: %s", err)
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

	// 插入 node 测试数据（不同时间点）
	nodeData := []struct {
		uid    string
		name   string
		cpu    int64
		memory int64
		time   time.Time
	}{
		{"node-uid-1", "node-1", 100, 512000, now.Add(-30 * time.Minute)},
		{"node-uid-1", "node-1", 200, 624000, now.Add(-25 * time.Minute)},
		{"node-uid-1", "node-1", 150, 576000, now.Add(-20 * time.Minute)},
		{"node-uid-1", "node-1", 300, 700000, now.Add(-15 * time.Minute)},
		{"node-uid-1", "node-1", 250, 650000, now.Add(-10 * time.Minute)},
		{"node-uid-1", "node-1", 180, 590000, now.Add(-5 * time.Minute)},
		{"node-uid-1", "node-1", 220, 620000, now},

		{"node-uid-2", "node-2", 120, 480000, now.Add(-30 * time.Minute)},
		{"node-uid-2", "node-2", 180, 520000, now.Add(-20 * time.Minute)},
		{"node-uid-2", "node-2", 220, 580000, now.Add(-10 * time.Minute)},
		{"node-uid-2", "node-2", 190, 550000, now},
	}

	for _, data := range nodeData {
		_, err := db.Exec(
			"INSERT INTO nodes (uid, name, cpu, memory, time) VALUES (?, ?, ?, ?, ?)",
			data.uid, data.name, data.cpu, data.memory, data.time.Format("2006-01-02T15:04:05Z"),
		)
		if err != nil {
			return err
		}
	}

	// 插入 pod 测试数据（不同时间点）
	podData := []struct {
		uid       string
		namespace string
		name      string
		container string
		cpu       int64
		memory    int64
		time      time.Time
	}{
		{"pod-uid-1", "default", "pod-1", "container-1", 50, 128000, now.Add(-30 * time.Minute)},
		{"pod-uid-1", "default", "pod-1", "container-1", 60, 135000, now.Add(-25 * time.Minute)},
		{"pod-uid-1", "default", "pod-1", "container-1", 55, 130000, now.Add(-20 * time.Minute)},
		{"pod-uid-1", "default", "pod-1", "container-1", 70, 140000, now.Add(-15 * time.Minute)},
		{"pod-uid-1", "default", "pod-1", "container-1", 65, 138000, now.Add(-10 * time.Minute)},
		{"pod-uid-1", "default", "pod-1", "container-1", 58, 132000, now.Add(-5 * time.Minute)},
		{"pod-uid-1", "default", "pod-1", "container-1", 62, 136000, now},

		{"pod-uid-2", "default", "pod-2", "container-1", 40, 100000, now.Add(-30 * time.Minute)},
		{"pod-uid-2", "default", "pod-2", "container-1", 45, 110000, now.Add(-20 * time.Minute)},
		{"pod-uid-2", "default", "pod-2", "container-1", 50, 120000, now.Add(-10 * time.Minute)},
		{"pod-uid-2", "default", "pod-2", "container-1", 48, 115000, now},

		{"kube-pod-uid", "kube-system", "kube-pod", "container-1", 30, 80000, now.Add(-15 * time.Minute)},
		{"kube-pod-uid", "kube-system", "kube-pod", "container-1", 35, 85000, now},
	}

	for _, data := range podData {
		_, err := db.Exec(
			"INSERT INTO pods (uid, namespace, name, container, cpu, memory, time) VALUES (?, ?, ?, ?, ?, ?, ?)",
			data.uid, data.namespace, data.name, data.container, data.cpu, data.memory, data.time.Format("2006-01-02T15:04:05Z"),
		)
		if err != nil {
			return err
		}
	}

	fmt.Println("Test data inserted:")
	fmt.Printf("  - %d node records\n", len(nodeData))
	fmt.Printf("  - %d pod records\n", len(podData))
	fmt.Printf("  - Time range: %s to %s\n",
		now.Add(-30*time.Minute).Format("2006-01-02T15:04:05Z"),
		now.Format("2006-01-02T15:04:05Z"))

	return nil
}
