# dashboard-metrics-scraper

Small binary to scrape and store a small window of metrics from the Metrics Server in Kubernetes.

**IMPORTANT: Metrics scraper codebase was moved to the Kubernetes Dashboard repository. You can find it [here](https://github.com/kubernetes/dashboard/tree/master/modules/metrics-scraper).**

## Command-Line Arguments
| Flag  | Description  | Default  |
|---|---|---|
| kubeconfig  | The path to the kubeconfig used to connect to the Kubernetes API server and the Kubelets (defaults to in-cluster config)  |  |
| db-file  | What file to use as a SQLite3 database.  |  `/tmp/metrics.db` |
| metric-resolution | The resolution at which dashboard-metrics-scraper will poll metrics.  | `1m` |
| metric-duration | The duration after which metrics are purged from the database. | `15m` |
| log-level | The log level. | `info` |
| logtostderr | Log to standard error. | `true` |
| namespace | The namespace to use for all metric calls. When provided, skip node metrics. | defaults to cluster level metrics |

## Community, discussion, contribution, and support

Learn how to engage with the Kubernetes community on the [community page](http://kubernetes.io/community/).

You can reach the maintainers of this project at:

- [Slack](http://slack.k8s.io/)
- [Mailing List](https://groups.google.com/forum/#!forum/kubernetes-dev)

### Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](code-of-conduct.md).

[owners]: https://git.k8s.io/community/contributors/guide/owners.md
[Creative Commons 4.0]: https://git.k8s.io/website/LICENSE


## 修改内容,增加时间范围筛选

URL 格式：

```
/namespaces/{Namespace}/pod-list/{Name}/
metrics/{MetricName}/{StartTime}-{EndTime}
/nodes/{Name}/metrics/{MetricName}/
{StartTime}-{EndTime}
```
时间格式采用 RFC3339 格式： 2006-01-02T15:04:05Z

示例：

```
/namespaces/default/pod-list/my-pod/
metrics/cpu/
2024-01-01T00:00:00Z-2024-01-02T00:00:00Z
/nodes/node-1/metrics/memory/
2024-05-01T10:00:00Z-2024-05-01T12:00:00Z
```

~/go/src/github.com/154650362/pixiu-metrics-scraper [143] $ echo "=== 测试时间范围过滤（15:30-15:45）===" && curl -s "http://localhost:8080/api/v1/dashboard/nodes/node-1/metrics/cpu/2026-05-18T15:30:00Z-2026-05-18T15:45:00Z" 
| python3 -m json.tool
=== 测试时间范围过滤（15:30-15:45）===
{
    "items": [
        {
            "dataPoints": [],
            "metricPoints": [
                {
                    "timestamp": "2026-05-18T15:31:17Z",
                    "value": 150
                },
                {
                    "timestamp": "2026-05-18T15:36:17Z",
                    "value": 300
                },
                {
                    "timestamp": "2026-05-18T15:41:17Z",
                    "value": 250
                }
            ],
            "metricName": "cpu",
            "uids": [
                "node-1"
            ]
        }
    ]
}

(TraeAI-4) ~/go/src/github.com/154650362/pixiu-metrics-scraper [0] $ echo "=== 测试 pod 的时间范围过滤（15:30-15:45）===" && curl -s "http://localhost:8080/api/v1/dashboard/namespaces/default/pod-list/pod-1/metrics/cpu/2026-05-18T15:30:
00Z-2026-05-18T15:45:00Z" | python3 -m json.tool
=== 测试 pod 的时间范围过滤（15:30-15:45）===
{
    "items": [
        {
            "dataPoints": [],
            "metricPoints": [
                {
                    "timestamp": "2026-05-18T15:31:17Z",
                    "value": 55
                },
                {
                    "timestamp": "2026-05-18T15:36:17Z",
                    "value": 70
                },
                {
                    "timestamp": "2026-05-18T15:41:17Z",
                    "value": 65
                }
            ],
            "metricName": "cpu",
            "uids": [
                "pod-1"
            ]
        }
    ]
}
