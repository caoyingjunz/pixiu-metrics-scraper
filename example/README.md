# Example: Time Range Filtering and Multiple Database Backends

这个示例演示了如何使用时间范围过滤功能查询 metrics 数据，以及如何支持多种数据库后端（SQLite、MySQL）。

## 功能说明

支持类似 Grafana 的 `from` 和 `to` 查询参数方式来过滤时间范围。

## 数据库后端支持

当前支持以下数据库：

| 数据库 | Driver 名称 | 文件 |
|-------|------------|------|
| SQLite | `sqlite3` | `sqlite.go` |
| MySQL | `mysql` | `mysql.go` |

## 时间格式支持

| 格式类型 | 示例 |
|---------|------|
| Unix Timestamp（毫秒） | `from=1779193798000` |
| RFC3339 | `from=2026-05-19T12:30:00Z` |
| 相对时间 | `from=now-30m&to=now` |
| 对齐时间 | `from=now/w&to=now` |

## 如何运行

### SQLite (默认)

```bash
cd example
go run server.go
```

### MySQL

#### 1. 安装 MySQL 并创建数据库

```bash
# 登录 MySQL
mysql -u root -p

# 创建数据库
CREATE DATABASE metrics_db CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

#### 2. 修改 MySQL 连接配置

编辑 `example/mysql_server.go` 中的连接参数：

```go
const (
    mysqlUser     = "root"      // 你的 MySQL 用户名
    mysqlPassword = "password"  // 你的 MySQL 密码
    mysqlHost     = "localhost" // MySQL 主机
    mysqlPort     = "3306"      // MySQL 端口
    mysqlDatabase = "metrics_db"// 数据库名称
)
```

#### 3. 运行 MySQL 示例

```bash
cd example
go run mysql_server.go
```

## API 使用示例

### 获取所有数据（不带时间过滤）

```bash
curl http://localhost:8080/api/v1/dashboard/nodes/node-1/metrics/cpu
```

### 使用相对时间（最近30分钟）

```bash
curl "http://localhost:8080/api/v1/dashboard/nodes/node-1/metrics/cpu?from=now-30m&to=now"
```

### 使用 Unix Timestamp（毫秒）

```bash
curl "http://localhost:8080/api/v1/dashboard/nodes/node-1/metrics/cpu?from=1779193798000&to=1779195652000"
```

## 完整 URL 格式

### Node Metrics

```
/api/v1/dashboard/nodes/{Name}/metrics/{MetricName}?from={StartTime}&to={EndTime}
```

### Pod Metrics

```
/api/v1/dashboard/namespaces/{Namespace}/pod-list/{Name}/metrics/{MetricName}?from={StartTime}&to={EndTime}
```

## 架构说明

### 目录结构

```
pkg/database/
├── database.go    # 接口定义 + 工厂方法
├── sqlite.go      # SQLite 实现
└── mysql.go       # MySQL 实现
```

### 核心设计

1. **接口抽象**：`DatabaseBackend` 接口定义所有数据库操作
2. **工厂模式**：`BackendFactory()` 根据驱动名称创建对应的后端实例
3. **向后兼容**：保留原有的函数签名，默认使用 SQLite

### 添加新数据库后端

只需实现 `DatabaseBackend` 接口并在 `BackendFactory()` 中添加新的 case：

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

## 与 Grafana 的兼容性

这种 `from` 和 `to` 参数方式与 Grafana 的时间范围选择器完全兼容。

### Grafana 查询配置

1. 在 Grafana 中添加 "Simple JSON" 数据源
2. 设置 URL 为：`http://your-server:8080/api/v1/dashboard`
3. 在查询中使用：
   - **Path**: `nodes/$node/metrics/cpu`
   - **Parameters**: `from=$__from&to=$__to`

## 测试数据

示例服务自动插入以下测试数据：

- **node-1**: 7 条 CPU 数据
- **node-2**: 4 条 CPU 数据
- **pod-1 (default)**: 7 条数据
- **pod-2 (default)**: 4 条数据
- **kube-pod (kube-system)**: 2 条数据

## 响应格式

```json
{
    "items": [
        {
            "metricName": "cpu",
            "metricPoints": [
                {
                    "timestamp": "2026-05-19T12:30:00Z",
                    "value": 150
                }
            ],
            "dataPoints": [],
            "uids": ["node-1"]
        }
    ]
}
```

## MySQL 最佳实践

1. **连接池配置**

```go
db.SetMaxOpenConns(25)    // 最大打开连接数
db.SetMaxIdleConns(10)   // 最大空闲连接数
db.SetConnMaxLifetime(5 * time.Minute) // 连接最大存活时间
```

2. **DSN 参数**

```
user:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True&loc=UTC
```

重要参数：
- `charset=utf8mb4`: 使用完整的 UTF-8
- `parseTime=True`: 解析时间类型
- `loc=UTC`: 使用 UTC 时区

3. **索引优化**

```sql
-- 为时间字段添加索引
CREATE INDEX idx_nodes_time ON nodes(time);
CREATE INDEX idx_pods_time ON pods(time);

-- 为查询字段添加索引
CREATE INDEX idx_nodes_name ON nodes(name);
CREATE INDEX idx_pods_name ON pods(name, namespace);
```

4. **分区表（大数据量）**

```sql
-- 按时间分区
CREATE TABLE nodes (
    uid varchar(255),
    name varchar(255),
    cpu varchar(255),
    memory varchar(255),
    storage varchar(255),
    time datetime
)
PARTITION BY RANGE (TO_DAYS(time)) (
    PARTITION p202401 VALUES LESS THAN (TO_DAYS('2024-02-01')),
    PARTITION p202402 VALUES LESS THAN (TO_DAYS('2024-03-01')),
    PARTITION p_future VALUES LESS THAN MAXVALUE
);
```
