---
title: 质量状态
sidebar: false
aside: false
---

<script setup>
import { data } from '../../quality-status.data.js'
</script>

<QualityStatusPage :data="data" locale="zh" />

## 数据源

本页只从腾讯云 COS 加载最新公开 nightly 状态 `https://cubesandbox-1253970226.cos.ap-singapore.myqcloud.com/page-data/quality-status.json`，以及逐晚历史 `https://cubesandbox-1253970226.cos.ap-singapore.myqcloud.com/page-data/quality-status-history.json`。

浏览器端刷新需要 COS 对象允许文档站域名跨域 `GET` 请求。

只应发布聚合后的 nightly E2E 和性能数据。不要在公开 JSON 中包含内部环境 ID、run ID、日志、凭据或私有 Web UI 链接。

## JSON 约定

公开 JSON 对象应包含以下顶层字段：

```json
{
  "generatedAt": "2026-09-14T17:30:00+08:00",
  "date": "2026-09-14",
  "commit": "ba77166b93fa995837a4bca0ae33b51d69c0494c",
  "status": "passed",
  "e2e": {
    "status": "passed",
    "tests": { "total": 128, "passed": 126, "failed": 0, "errors": 0, "skipped": 2 },
    "modules": [{ "module": "lifecycle", "total": 32, "passed": 32, "failed": 0, "errors": 0, "skipped": 0 }]
  },
  "performance": {
    "status": "passed",
    "baseline": {
      "name": "v0.3.0 baseline",
      "url": "https://cubesandbox.com/blog/posts/2026-06-01-cubesandbox-perf-benchmark",
      "url_zh": "https://cubesandbox.com/zh/blog/posts/2026-06-01-cubesandbox-perf-benchmark"
    },
    "counts": { "critical": 0, "warning": 1, "similar": 8, "improved": 2 },
    "thresholds": { "similar_pct": 5, "warning_pct": 5, "critical_pct": 10 },
    "sections": [{ "key": "3.2", "title": "3.2 Cold-Start Latency and Concurrency Scaling", "status": "passed" }],
    "metrics": [{
      "key": "3.2-c1-create-avg",
      "section": "3.2",
      "label": "3.2 c=1 create avg",
      "unit": "ms",
      "better": "lower",
      "baseline": 47.8,
      "baseline_text": "47.8 ms",
      "current": 40.4,
      "current_text": "40.4 ms",
      "delta_pct": -15.5,
      "verdict": "improved"
    }]
  }
}
```

最新对象里的 `e2e.modules` 是数组。`performance.thresholds.similar_pct` 用来画走势图的阴影带，`performance.sections[].key` 用来给指标分组。嵌套的 `performance.comparison` 可以提供 `counts`、`metrics`、`thresholds`、`sections` 和 `note`；里面缺的字段回退到 `performance`。

历史是另一个对象：

```json
{
  "generatedAt": "2026-10-16T09:33:00+08:00",
  "nights": [{
    "date": "2026-10-16",
    "commit": "5741deb433b7e2b9c9d2dcadb1c87214de74bd24",
    "status": "passed",
    "e2e": {
      "status": "passed",
      "tests": { "total": 316, "passed": 290, "failed": 0, "errors": 0, "skipped": 26 },
      "modules": {
        "lifecycle": { "total": 32, "passed": 32, "failed": 0, "errors": 0, "skipped": 0 }
      }
    },
    "performance": {
      "status": "critical",
      "counts": { "critical": 1, "warning": 0, "similar": 0, "improved": 0 },
      "values": { "3.2-c1-create-avg": 40.4 },
      "worst": { "label": "3.2 c=1 create avg", "delta_pct": 12.0 }
    }
  }]
}
```

`nights[].e2e.modules` 是按模块名索引的对象，不是数组。`nights[].performance.values` 以 `metrics[].key` 为键，用来画走势。`status: "no_run"` 表示当晚没有定时运行。页面会先按 `date` 从旧到新排序，再画夜间墙、连续通过天数和走势。

Status 页面会按当前语言选择基线链接。中文页优先使用 `baseline.url_zh`，英文页优先使用 `baseline.url` 或 `baseline.url_en`。如果 JSON 只有一个 URL，页面对 `cubesandbox.com` 路径添加或去掉 `/zh` 前缀，并把 GitHub 的 `docs/blog/posts/...` 与 `docs/zh/blog/posts/...` 互改。
