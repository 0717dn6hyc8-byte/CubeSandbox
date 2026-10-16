---
title: Quality Status
sidebar: false
aside: false
---

<script setup>
import { data } from '../quality-status.data.js'
</script>

<QualityStatusPage :data="data" locale="en" />

## Data Source

This page reads only the latest public nightly status from `https://cubesandbox-1253970226.cos.ap-singapore.myqcloud.com/page-data/quality-status.json` and the night-by-night history from `https://cubesandbox-1253970226.cos.ap-singapore.myqcloud.com/page-data/quality-status-history.json`.

Browser-side refresh requires the COS object to allow cross-origin `GET` requests from the documentation domain.

Only aggregate nightly E2E and performance data should be published. Do not include internal environment IDs, run IDs, logs, credentials, or private Web UI links.

## JSON Contract

The public JSON object should contain the following top-level fields:

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

`e2e.modules` on the latest object is an array. `performance.thresholds.similar_pct` draws the sparkline band, and `performance.sections[].key` groups metrics. A nested `performance.comparison` object is accepted for `counts`, `metrics`, `thresholds`, `sections`, and `note`; fields missing there fall back to `performance`.

The history object is separate:

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

`nights[].e2e.modules` is an object keyed by module name, not an array. `nights[].performance.values` is keyed by `metrics[].key` and feeds the sparklines. `status: "no_run"` is a night with no scheduled run. The page sorts `nights` by `date` ascending before drawing the wall, the streak, and the trends.

The status page picks the baseline link from the current locale. The Chinese page prefers `baseline.url_zh`; the English page prefers `baseline.url` or `baseline.url_en`. If the JSON contains only one URL, the page adds or removes a `/zh` prefix on `cubesandbox.com` paths, and rewrites GitHub `docs/blog/posts/...` to or from `docs/zh/blog/posts/...`.
