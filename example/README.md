# Example: Time Range Filtering

这个示例演示了如何使用时间范围过滤功能查询 metrics 数据。

## 功能说明

支持类似 Grafana 的 `from` 和 `to` 查询参数方式来过滤时间范围：

```
?from={StartTime}&to={EndTime}
```

支持以下时间格式：

### 1. Unix Timestamp（毫秒）
```
?from=1716075607000&to=1716077407000
```

### 2. RFC3339 格式
```
?from=2026-05-19T12:30:00Z&to=2026-05-19T12:45:00Z
```

### 3. Grafana 相对时间格式
```
?from=now-1h&to=now
?from=now-30m&to=now
?from=now-2d&to=now
?from=now-7d&to=now-1d
```

支持的时间单位：
- `s` - 秒
- `m` - 分钟
- `h` - 小时
- `d` - 天
- `w` - 周
- `M` - 月
- `Y` - 年

### 4. Grafana 对齐时间格式
```
?from=now/w&to=now    # 本周开始到现在
?from=now/M&to=now    # 本月开始到现在
?from=now/Y&to=now    # 本年开始到现在
```

## 如何运行

```bash
cd example
go run server.go
```

服务将在 `http://localhost:8080` 上启动，并自动插入测试数据。

## API 使用示例

### 1. 获取所有数据（不带时间过滤）
```bash
curl http://localhost:8080/api/v1/dashboard/nodes/node-1/metrics/cpu
```

### 2. 使用相对时间（最近20分钟）
```bash
curl "http://localhost:8080/api/v1/dashboard/nodes/node-1/metrics/cpu?from=now-20m&to=now"
```

### 3. 使用 Unix Timestamp（毫秒）
```bash
curl "http://localhost:8080/api/v1/dashboard/nodes/node-1/metrics/cpu?from=1716075607000&to=1716077407000"
```

### 4. 使用 RFC3339 格式
```bash
curl "http://localhost:8080/api/v1/dashboard/nodes/node-1/metrics/cpu?from=2026-05-19T12:30:00Z&to=2026-05-19T12:45:00Z"
```

### 5. 查询 pod 数据
```bash
curl "http://localhost:8080/api/v1/dashboard/namespaces/default/pod-list/pod-1/metrics/cpu?from=now-1h&to=now"
```

### 6. 查询 memory 指标
```bash
curl "http://localhost:8080/api/v1/dashboard/nodes/node-1/metrics/memory?from=now-30m&to=now"
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

## 测试数据

示例服务自动插入以下测试数据：

- **node-1**: 7 条 CPU 数据，时间范围 `15:19` - `15:49`
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
          "timestamp": "2026-05-18T15:30:00Z",
          "value": 150
        },
        {
          "timestamp": "2026-05-18T15:35:00Z",
          "value": 200
        }
      ],
      "dataPoints": [],
      "uids": ["node-1"]
    }
  ]
}
```

## 与 Grafana 的兼容性

这种 `from` 和 `to` 参数方式与 Grafana 的时间范围选择器完全兼容。

### Grafana 变量示例

在 Grafana 中，可以使用以下变量来动态传递时间范围：

```
/api/v1/dashboard/nodes/$node/metrics/cpu?from=$__from&to=$__to
```

Grafana 会自动将 `$__from` 和 `$__to` 替换为当前面板的时间范围。

### Grafana 查询配置示例

1. 在 Grafana 中添加一个 "Simple JSON" 数据源
2. 设置 URL 为：`http://your-server:8080/api/v1/dashboard`
3. 在查询中使用：
   - **Path**: `nodes/$node/metrics/cpu`
   - **Parameters**: `from=$__from&to=$__to`

### 时间范围示例

| Grafana 时间范围 | URL 参数 |
|-----------------|---------|
| Last 5 minutes | `from=now-5m&to=now` |
| Last 15 minutes | `from=now-15m&to=now` |
| Last 30 minutes | `from=now-30m&to=now` |
| Last 1 hour | `from=now-1h&to=now` |
| Last 6 hours | `from=now-6h&to=now` |
| Last 24 hours | `from=now-24h&to=now` |
| Last 7 days | `from=now-7d&to=now` |
| This week | `from=now/w&to=now` |
| This month | `from=now/M&to=now` |
| This year | `from=now/Y&to=now` |
