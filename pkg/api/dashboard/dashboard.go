package provider

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/types"

	"github.com/gorilla/mux"
)

// DashboardRouter defines the usable API routes
func DashboardRouter(r *mux.Router, db *sql.DB) {
	r.Path("/nodes/{Name}/metrics/{MetricName}").HandlerFunc(nodeHandler(db))
	r.Path("/namespaces/{Namespace}/pod-list/{Name}/metrics/{MetricName}").HandlerFunc(podHandler(db))
	r.PathPrefix("/").HandlerFunc(defaultHandler)
}

func defaultHandler(w http.ResponseWriter, r *http.Request) {
	msg := fmt.Sprintf("%v - URL: %s", time.Now(), r.URL)
	_, err := w.Write([]byte(msg))
	if err != nil {
		log.Errorf("Error cannot write response: %v", err)
	}
}

func parseTimeRange(r *http.Request) (time.Time, time.Time) {
	var startTime, endTime time.Time

	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	if from != "" {
		startTime = parseGrafanaTime(from)
	}

	if to != "" {
		endTime = parseGrafanaTime(to)
	}

	return startTime, endTime
}

func parseGrafanaTime(timeStr string) time.Time {
	// 尝试解析 Unix timestamp（毫秒）- 仅支持13位毫秒格式
	if ts, err := strconv.ParseInt(timeStr, 10, 64); err == nil && len(timeStr) == 13 {
		return time.UnixMilli(ts).UTC()
	}

	// 尝试解析 RFC3339 格式
	layout := "2006-01-02T15:04:05Z"
	if t, err := time.Parse(layout, timeStr); err == nil {
		return t
	}

	// 解析 Grafana 相对时间格式
	return parseRelativeTime(timeStr)
}

func parseRelativeTime(timeStr string) time.Time {
	now := time.Now().UTC()

	if timeStr == "now" {
		return now
	}

	// 格式: now-5m, now-1h, now-2d, now-3w, now-1M, now-1Y
	if strings.HasPrefix(timeStr, "now-") {
		durationStr := timeStr[4:]
		return parseDurationFromNow(now, durationStr)
	}

	// 格式: now/w (本周开始), now/M (本月开始), now/Y (本年开始)
	if strings.HasSuffix(timeStr, "/w") {
		return startOfWeek(now)
	}
	if strings.HasSuffix(timeStr, "/M") {
		return startOfMonth(now)
	}
	if strings.HasSuffix(timeStr, "/Y") {
		return startOfYear(now)
	}

	return now
}

func parseDurationFromNow(now time.Time, durationStr string) time.Time {
	if durationStr == "" {
		return now
	}

	unit := durationStr[len(durationStr)-1]
	value, err := strconv.Atoi(durationStr[:len(durationStr)-1])
	if err != nil {
		return now
	}

	switch unit {
	case 's':
		return now.Add(-time.Duration(value) * time.Second)
	case 'm':
		return now.Add(-time.Duration(value) * time.Minute)
	case 'h':
		return now.Add(-time.Duration(value) * time.Hour)
	case 'd':
		return now.Add(-time.Duration(value) * 24 * time.Hour)
	case 'w':
		return now.Add(-time.Duration(value) * 7 * 24 * time.Hour)
	case 'M':
		return now.AddDate(0, -value, 0)
	case 'Y':
		return now.AddDate(-value, 0, 0)
	default:
		return now
	}
}

func startOfWeek(t time.Time) time.Time {
	weekday := t.Weekday()
	daysToSubtract := int(weekday)
	if weekday == time.Sunday {
		daysToSubtract = 7
	}
	return time.Date(t.Year(), t.Month(), t.Day()-daysToSubtract, 0, 0, 0, 0, time.UTC)
}

func startOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func startOfYear(t time.Time) time.Time {
	return time.Date(t.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
}

func nodeHandler(db *sql.DB) http.HandlerFunc {
	fn := func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		startTime, endTime := parseTimeRange(r)

		resp, err := getNodeMetrics(db, vars["MetricName"], ResourceSelector{
			Namespace:    "",
			ResourceName: vars["Name"],
			StartTime:    startTime,
			EndTime:      endTime,
		})

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, err := w.Write([]byte(fmt.Sprintf("Node Metrics Error - %v", err.Error())))
			if err != nil {
				log.Errorf("Error cannot write response: %v", err)
			}
		}

		j, err := json.Marshal(resp)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, err := w.Write([]byte(fmt.Sprintf("JSON Error - %v", err.Error())))
			if err != nil {
				log.Errorf("Error cannot write response: %v", err)
			}
		}

		_, err = w.Write(j)
		if err != nil {
			log.Errorf("Error cannot write response: %v", err)
		}
	}

	return fn
}

func podHandler(db *sql.DB) http.HandlerFunc {
	fn := func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		startTime, endTime := parseTimeRange(r)

		resp, err := getPodMetrics(db, vars["MetricName"], ResourceSelector{
			Namespace:    vars["Namespace"],
			ResourceName: vars["Name"],
			StartTime:    startTime,
			EndTime:      endTime,
		})

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, err := w.Write([]byte(fmt.Sprintf("Pod Metrics Error - %v", err.Error())))
			if err != nil {
				log.Errorf("Error cannot write response: %v", err)
			}
		}

		j, err := json.Marshal(resp)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, err := w.Write([]byte(fmt.Sprintf("JSON Error - %v", err.Error())))
			if err != nil {
				log.Errorf("Error cannot write response: %v", err)
			}
		}

		_, err = w.Write(j)
		if err != nil {
			log.Errorf("Error cannot write response: %v", err)
		}
	}

	return fn
}

func getRows(db *sql.DB, table string, metricName string, selector ResourceSelector) (*sql.Rows, error) {
	var query string
	var values []interface{}
	var args []string
	orderBy := []string{"name", "time"}
	if metricName == "cpu" {
		query = "select sum(cpu), name, uid, time from %s "
	} else {
		//default to metricName == "memory/usage"
		// metricName = "memory"
		query = "select sum(memory), name, uid, time from %s "
	}

	if table == "pods" {
		orderBy = []string{"namespace", "name", "time"}
		args = append(args, "namespace=?")
		if selector.Namespace != "" {
			values = append(values, selector.Namespace)
		} else {
			values = append(values, "default")
		}
	}

	if selector.ResourceName != "" {
		if strings.ContainsAny(selector.ResourceName, ",") {
			subargs := []string{}
			for _, v := range strings.Split(selector.ResourceName, ",") {
				subargs = append(subargs, "?")
				values = append(values, v)
			}
			args = append(args, " name in ("+strings.Join(subargs, ",")+")")
		} else {
			values = append(values, selector.ResourceName)
			args = append(args, " name = ?")
		}
	}
	if selector.UID != "" {
		args = append(args, " uid = ?")
		values = append(values, selector.UID)
	}

	if !selector.StartTime.IsZero() {
		args = append(args, " time >= ?")
		values = append(values, selector.StartTime.Format("2006-01-02T15:04:05Z"))
	}

	if !selector.EndTime.IsZero() {
		args = append(args, " time <= ?")
		values = append(values, selector.EndTime.Format("2006-01-02T15:04:05Z"))
	}

	query = fmt.Sprintf(query+" where "+strings.Join(args, " and ")+" group by name, time order by %v;", table, strings.Join(orderBy, ", "))

	return db.Query(query, values...)
}

/*
getPodMetrics: With a database connection and a resource selector
Queries the database (SQLite/MySQL) and returns a list of pod metrics.
*/
func getPodMetrics(db *sql.DB, metricName string, selector ResourceSelector) (SidecarMetricResultList, error) {
	rows, err := getRows(db, "pods", metricName, selector)
	if err != nil {
		log.Errorf("Error getting pod metrics: %v", err)
		return SidecarMetricResultList{}, err
	}

	defer rows.Close()

	resultList := make(map[string]SidecarMetric)

	for rows.Next() {
		var metricValue string
		var pod string
		var metricTime string
		var uid string
		var newMetric MetricPoint
		err = rows.Scan(&metricValue, &pod, &uid, &metricTime)
		if err != nil {
			return SidecarMetricResultList{}, err
		}

		layout := "2006-01-02T15:04:05Z"
		t, err := time.Parse(layout, metricTime)
		if err != nil {
			return SidecarMetricResultList{}, err
		}

		v, err := strconv.ParseUint(metricValue, 10, 64)
		if err != nil {
			return SidecarMetricResultList{}, err
		}

		newMetric = MetricPoint{
			Timestamp: t,
			Value:     v,
		}

		if _, ok := resultList[pod]; ok {
			metricThing := resultList[pod]
			metricThing.AddMetricPoint(newMetric)
			resultList[pod] = metricThing
		} else {
			resultList[pod] = SidecarMetric{
				MetricName:   metricName,
				MetricPoints: []MetricPoint{newMetric},
				DataPoints:   []DataPoint{},
				UIDs: []types.UID{
					types.UID(pod),
				},
			}
		}
	}
	err = rows.Err()
	if err != nil {
		return SidecarMetricResultList{}, err
	}

	result := SidecarMetricResultList{}
	for _, v := range resultList {
		result.Items = append(result.Items, v)
	}

	return result, nil
}

/*
getNodeMetrics: With a database connection and a resource selector
Queries the database (SQLite/MySQL) and returns a list of node metrics.
*/
func getNodeMetrics(db *sql.DB, metricName string, selector ResourceSelector) (SidecarMetricResultList, error) {
	resultList := make(map[string]SidecarMetric)
	rows, err := getRows(db, "nodes", metricName, selector)

	if err != nil {
		log.Errorf("Error getting node metrics: %v", err)
		return SidecarMetricResultList{}, err
	}

	defer rows.Close()
	for rows.Next() {
		var metricValue string
		var node string
		var metricTime string
		var uid string
		var newMetric MetricPoint
		err = rows.Scan(&metricValue, &node, &uid, &metricTime)
		if err != nil {
			return SidecarMetricResultList{}, err
		}

		layout := "2006-01-02T15:04:05Z"
		t, err := time.Parse(layout, metricTime)
		if err != nil {
			return SidecarMetricResultList{}, err
		}

		v, err := strconv.ParseUint(metricValue, 10, 64)
		if err != nil {
			return SidecarMetricResultList{}, err
		}

		newMetric = MetricPoint{
			Timestamp: t,
			Value:     v,
		}

		if _, ok := resultList[node]; ok {
			metricThing := resultList[node]
			metricThing.AddMetricPoint(newMetric)
			resultList[node] = metricThing
		} else {
			resultList[node] = SidecarMetric{
				MetricName:   metricName,
				MetricPoints: []MetricPoint{newMetric},
				DataPoints:   []DataPoint{},
				UIDs: []types.UID{
					types.UID(node),
				},
			}
		}
	}
	err = rows.Err()
	if err != nil {
		return SidecarMetricResultList{}, err
	}

	result := SidecarMetricResultList{}
	for _, v := range resultList {
		result.Items = append(result.Items, v)
	}

	return result, nil
}
