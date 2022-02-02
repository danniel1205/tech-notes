# Distributed monitoring system

This contains notes after the research on Prometheus. Google Monarch has a completely different design, see
[this article](../../../how-google-xxx-series/monarch-planet-scale-in-memory-time-series-database/readme.md) for more details.

## Architecture

![](resources/prometheus-architecture.png)

### Prometheus server

- Running on single node, in K8S its replica=1.
- No native support for HA. ([Ref](https://medium.com/miro-engineering/prometheus-high-availability-and-fault-tolerance-strategy-long-term-storage-with-victoriametrics-82f6f3f0409e))
- **Pull Model**: Scrape/Pull metrics with a configurable interval.
- Support federation: Instance A scrape metrics from Instance B.

### PushGateway

- Metrics from short-lived job could be sent to PushGateway.
- Metrics are stored on disk. In K8S, we could enable PV. ([Ref](https://www.metricfire.com/blog/prometheus-pushgateways-everything-you-need-to-know/))

### AlertManager

- Prometheus server pushes alerts to AlertManager.
- Support HA with native clustering support. ([Ref](https://prometheus.io/docs/alerting/latest/alertmanager/#high-availability))

## TSDB within Prometheus Server

The following is based on V3 design. ([Prometheus TSDB from scratch](https://fabxc.org/tsdb/))

![](resources/prometheus-tsdb-file-structure.png)

- `b-000001` is a block(2h time window). ([Ref](https://prometheus.io/docs/prometheus/latest/storage/#on-disk-layout))
- `chunks/000001` holds the raw data for various series:
  ```text
  ^
  series
  |
  | {__name__="request_total", method="GET",  ["t1:v1","t2:v2"]}
  | {__name__="request_total", method="POST", ["t1:v1","t2:v2"]}
  | {__name__="error_total",   method="GET",  ["t1:v1","t2:v2"]}
  | {__name__="error_total",   method="POST", ["t1:v1","t2:v2"]}
  -----------------------------------------------------time---->
  ```
- Each block has an inverted index file.
- Each block has an metadata file.

![](resources/prometheus-tsdb-blocks.png)

- Most recent block is held in memory with WAL to prevent failure.
- Old blocks could be compacted by a background process.

### Good to know

- SSDs are known for fast random writes, they actually can’t modify individual bytes but only write in pages of 4KiB or
  more. This means writing a 16 byte sample is equivalent to writing a full 4KiB page.
  This behavior is part of what is known as [write amplification](https://en.wikipedia.org/wiki/Write_amplification).
- [mmap(2)](https://en.wikipedia.org/wiki/Mmap), a system call that allows us to transparently back a virtual memory
  region by file contents. This means we can treat all contents of our database as if they were in memory without occupying
  any physical RAM. Only if we access certain byte ranges in our database files, the operating system lazily loads pages
  from disk.

## References

- [Prometheus TSDB from scratch](https://fabxc.org/tsdb/)
- [Prometheus Storage](https://prometheus.io/docs/prometheus/latest/storage/#on-disk-layout)