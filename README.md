# dashboard-metrics-scraper

Small binary to scrape and store a small window of metrics from the Metrics Server in Kubernetes.

**IMPORTANT: Metrics scraper codebase was moved to the Kubernetes Dashboard repository. You can find it [here](https://github.com/kubernetes/dashboard/tree/master/modules/metrics-scraper).**

## Command-Line Arguments
| Flag  | Description  | Default  |
|---|---|---|
| kubeconfig  | The path to the kubeconfig used to connect to the Kubernetes API server and the Kubelets (defaults to in-cluster config)  |  |
| metric-resolution | The resolution at which dashboard-metrics-scraper will poll metrics.  | `1m` |
| metric-duration | The duration after which metrics are purged from the database. | `15m` |
| log-level | The log level. | `info` |
| logtostderr | Log to standard error. | `true` |
| namespace | The namespace to use for all metric calls. When provided, skip node metrics. | defaults to cluster level metrics |
| config | Path to the config file. | |



## Community, discussion, contribution, and support

Learn how to engage with the Kubernetes community on the [community page](http://kubernetes.io/community/).

You can reach the maintainers of this project at:

- [Slack](http://slack.k8s.io/)
- [Mailing List](https://groups.google.com/forum/#!forum/kubernetes-dev)

### Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](code-of-conduct.md).

[owners]: https://git.k8s.io/community/contributors/guide/owners.md
[Creative Commons 4.0]: https://git.k8s.io/website/LICENSE


## config.yaml 格式

```yaml
database:
  type: "mysql"

  sqlite3:
    db-file: "/tmp/metrics.db"

  mysql:
    username: "root"
    password: "your_password"
    host: "localhost"
    port: "3306"
    database: "pixiu_metrics"
    charset: "utf8mb4"
    max-open-conns: 25
    max-idle-conns: 10

server:
  address: ":8000"
```


## 功能说明

支持类似 Grafana 的 `from` 和 `to` 查询参数方式来过滤时间范围。

## 数据库后端支持

当前支持以下数据库：

| 数据库 | Driver     |
|-------|------------|
| SQLite | `sqlite3` | 
| MySQL  | `mysql`   | 

## 时间格式支持

| 格式类型 | 示例 |
|---------|------|
| Unix Timestamp（毫秒） | `from=1779193798000` |
| RFC3339 | `from=2026-05-19T12:30:00Z` |
| 相对时间 | `from=now-30m&to=now` |
| 对齐时间 | `from=now/w&to=now` |


### 扩展性

只需实现 `DatabaseBackend` 接口并在 `BackendFactory()` 中添加新的数据库后端：

```go
type PostgreSQLBackend struct{}

func (p *PostgreSQLBackend) CreateDatabase(db *sql.DB) error {
    // 实现...
}

func (p *PostgreSQLBackend) UpdateDatabase(db *sql.DB, ...) error {
    // 实现...
}

func (p *PostgreSQLBackend) CullDatabase(db *sql.DB, ...) error {
    // 实现...
}

// 在 BackendFactory 中添加
case "postgres":
    return &PostgreSQLBackend{}
```

### 优化数据库

**索引优化**

```sql
-- 为时间字段添加索引
CREATE INDEX idx_nodes_time ON nodes(time);
CREATE INDEX idx_pods_time ON pods(time);
```