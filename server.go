package main

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"

	sideapi "github.com/kubernetes-sigs/dashboard-metrics-scraper/pkg/api"
	sidecfg "github.com/kubernetes-sigs/dashboard-metrics-scraper/pkg/config"
	sidedb "github.com/kubernetes-sigs/dashboard-metrics-scraper/pkg/database"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsclient "k8s.io/metrics/pkg/client/clientset/versioned"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

func main() {
	var kubeconfig *string
	var configFile *string
	var metricResolution *time.Duration
	var metricDuration *time.Duration
	var logLevel *string
	var logToStdErr *bool
	var metricNamespace *[]string

	log.SetFormatter(&log.JSONFormatter{})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	log.SetLevel(log.InfoLevel)

	kubeconfig = flag.String("kubeconfig", "", "The path to the kubeconfig used to connect to the Kubernetes API server and the Kubelets (defaults to in-cluster config)")
	configFile = flag.StringP("config", "c", "", "Path to the YAML configuration file. If not provided, defaults to SQLite.")
	metricResolution = flag.Duration("metric-resolution", 1*time.Minute, "The resolution at which dashboard-metrics-scraper will poll metrics.")
	metricDuration = flag.Duration("metric-duration", 15*time.Minute, "The duration after which metrics are purged from the database.")
	logLevel = flag.String("log-level", "info", "The log level")
	logToStdErr = flag.Bool("logtostderr", true, "Log to stderr")
	// When running in a scoped namespace, disable Node lookup and only capture metrics for the given namespace(s)
	metricNamespace = flag.StringSliceP("namespace", "n", []string{getEnv("POD_NAMESPACE", "")}, "The namespace to use for all metric calls. When provided, skip node metrics. (defaults to cluster level metrics)")

	flag.Parse()

	if *logToStdErr {
		log.SetOutput(os.Stderr)
	}

	level, err := log.ParseLevel(*logLevel)
	if err != nil {
		log.Fatal(err)
	} else {
		log.SetLevel(level)
	}

	// Load configuration file if provided
	var cfg *sidecfg.Config
	if *configFile != "" {
		cfg, err = sidecfg.LoadConfig(*configFile)
		if err != nil {
			log.Fatalf("Unable to load config file: %s", err)
		}
		log.Infof("Loaded configuration from: %s", *configFile)
	} else {
		cfg = sidecfg.DefaultConfig()
		log.Info("Using default SQLite configuration")
	}

	// This should only be run in-cluster so...
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		log.Fatalf("Unable to generate a client config: %s", err)
	}

	log.Infof("Kubernetes host: %s", config.Host)
	log.Infof("Namespace(s): %s", *metricNamespace)

	// Generate the metrics client
	clientset, err := metricsclient.NewForConfig(config)
	if err != nil {
		log.Fatalf("Unable to generate a clientset: %s", err)
	}

	// Create database connection based on configuration
	var db *sql.DB
	var backend sidedb.DatabaseBackend

	switch cfg.Database.Type {
	case "mysql":
		log.Infof("Using MySQL database: %s:%s/%s",
			cfg.Database.MySQL.Host,
			cfg.Database.MySQL.Port,
			cfg.Database.MySQL.Database)

		db, err = sql.Open("mysql", cfg.Database.MySQL.GetDSN())
		if err != nil {
			log.Fatalf("Unable to open MySQL database: %s", err)
		}

		// Configure connection pool
		if cfg.Database.MySQL.MaxOpenConns > 0 {
			db.SetMaxOpenConns(cfg.Database.MySQL.MaxOpenConns)
		}
		if cfg.Database.MySQL.MaxIdleConns > 0 {
			db.SetMaxIdleConns(cfg.Database.MySQL.MaxIdleConns)
		}
		db.SetConnMaxLifetime(5 * time.Minute)

		// Test connection
		if err := db.Ping(); err != nil {
			log.Fatalf("Unable to ping MySQL database: %s", err)
		}

		backend = sidedb.BackendFactory("mysql")

	case "sqlite3":
		log.Infof("Using SQLite database: %s", cfg.Database.SQLite3.DBFile)

		db, err = sql.Open("sqlite3", cfg.Database.SQLite3.DBFile)
		if err != nil {
			log.Fatalf("Unable to open SQLite database: %s", err)
		}

		backend = sidedb.BackendFactory("sqlite3")

	default:
		log.Warnf("Unknown database type '%s', defaulting to SQLite", cfg.Database.Type)

		db, err = sql.Open("sqlite3", cfg.Database.SQLite3.DBFile)
		if err != nil {
			log.Fatalf("Unable to open SQLite database: %s", err)
		}

		backend = sidedb.BackendFactory("sqlite3")
	}

	defer db.Close()

	err = backend.CreateDatabase(db)
	if err != nil {
		log.Fatalf("Unable to initialize database tables: %s", err)
	}

	go func() {
		r := mux.NewRouter()

		sideapi.Manager(r, db)
		addr := cfg.Server.Address
		if addr == "" {
			addr = ":8000"
		}
		log.Infof("Starting server on %s", addr)
		log.Fatal(http.ListenAndServe(addr, handlers.CombinedLoggingHandler(os.Stdout, r)))
	}()

	ticker := time.NewTicker(*metricResolution)
	quit := make(chan struct{})

	for {
		select {
		case <-quit:
			ticker.Stop()
			return

		case <-ticker.C:
			err = update(clientset, db, backend, metricDuration, metricNamespace)
			if err != nil {
				break
			}
		}
	}
}

/**
* Update the Node and Pod metrics in the provided DB
 */
func update(client *metricsclient.Clientset, db *sql.DB, backend sidedb.DatabaseBackend, metricDuration *time.Duration, metricNamespace *[]string) error {
	nodeMetrics := &v1beta1.NodeMetricsList{}
	podMetrics := &v1beta1.PodMetricsList{}
	ctx := context.TODO()
	var err error

	// If no namespace is provided, make a call to the Node
	if len(*metricNamespace) == 1 && (*metricNamespace)[0] == "" {
		// List node metrics across the cluster
		nodeMetrics, err = client.MetricsV1beta1().NodeMetricses().List(ctx, v1.ListOptions{})
		if err != nil {
			log.Errorf("Error scraping node metrics: %s", err)
			return err
		}
	}

	// List pod metrics across the cluster, or for a given namespace
	for _, namespace := range *metricNamespace {
		pod, err := client.MetricsV1beta1().PodMetricses(namespace).List(ctx, v1.ListOptions{})
		if err != nil {
			log.Errorf("Error scraping '%s' for pod metrics: %s", namespace, err)
			return err
		}
		podMetrics.TypeMeta = pod.TypeMeta
		podMetrics.ListMeta = pod.ListMeta
		podMetrics.Items = append(podMetrics.Items, pod.Items...)
	}

	// Insert scrapes into DB using the appropriate backend
	err = backend.UpdateDatabase(db, nodeMetrics, podMetrics)
	if err != nil {
		log.Errorf("Error updating database: %s", err)
		return err
	}

	// Delete rows outside of the metricDuration time
	err = backend.CullDatabase(db, metricDuration)
	if err != nil {
		log.Errorf("Error culling database: %s", err)
		return err
	}

	log.Infof("Database updated: %d nodes, %d pods", len(nodeMetrics.Items), len(podMetrics.Items))
	return nil
}

/**
* Lookup the environment variable provided and set to default value if variable isn't found
 */
func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		value = fallback
	}
	return value
}
