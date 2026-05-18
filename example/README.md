# Example: Time Range Filtering

这个示例演示了如何使用时间范围过滤功能查询 metrics 数据。

## 功能说明

URL 路径中的 `{Whatever}` 参数支持时间范围过滤，格式如下：
```
{StartTime}-{EndTime}
```

- 时间格式: RFC3339 (`2006-01-02T15:04:05Z`)
- 两个时间之间用 `-` 分隔

## 如何运行

```bash
cd example
go run server.go
```

服务将在 `http://localhost:8080` 上启动，并自动插入测试数据。

## API 使用示例

### 1. 获取所有数据（不带时间过滤）
```bash
curl http://localhost:8080/api/v1/dashboard/nodes/node-1/metrics/cpu/all
```

### 2. 按时间范围过滤数据
```bash
curl "http://localhost:8080/api/v1/dashboard/nodes/node-1/metrics/cpu/2026-05-18T15:30:00Z-2026-05-18T15:45:00Z"
```

### 3. 查询 pod 数据
```bash
curl "http://localhost:8080/api/v1/dashboard/namespaces/default/pod-list/pod-1/metrics/cpu/2026-05-18T15:30:00Z-2026-05-18T15:45:00Z"
```

### 4. 查询 memory 指标
```bash
curl "http://localhost:8080/api/v1/dashboard/nodes/node-1/metrics/memory/2026-05-18T15:30:00Z-2026-05-18T15:45:00Z"
```

## 完整 URL 格式

### Node Metrics
```
/api/v1/dashboard/nodes/{Name}/metrics/{MetricName}/{StartTime}-{EndTime}
```

### Pod Metrics
```
/api/v1/dashboard/namespaces/{Namespace}/pod-list/{Name}/metrics/{MetricName}/{StartTime}-{EndTime}
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
