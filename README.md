# Tech Notes

![header-pic](header-pic.png)

This repo contains the notes/tutorials from my personal tech exploration.

## Table of contents

### AI/ML Related

- [AI Inference Concepts](ai/ai-inference-concpets/readme.md)
  - [Disaggregated Serving Concepts](ai/ai-inference-concpets/disaggregated-serving.md)
- [Build simple local RAG](ai/local-rag/readme.md)
  - [Build local RAG](ai/local-rag/notebook.ipynb)
- [Try out building AI agent on local](ai/ai-agent/notebook.ipynb)

### CKA (Certified Kubernetes Administrator) Related

- [Create the K8S cluster on vSphere the hard way](cka/create-k8s-on-vsphere-hard-way.md)
- [Create user accounts](cka/create-user-accounts/create-user-accounts.md)
- [Init containers](cka/manage-pod/init-containers.md)
- [Manage custome resources](cka/manage-custome-resources/manage-custome-resources.md)
- [Manage pod networking](cka/manage-pod-networking/manage-pod-networking.md)
- [Manage scheduling](cka/manage-scheduling/manage-scheduling.md)
- [Mange cluster nodes](cka/manage-cluster-nodes/manage-cluster-nodes.md)
- [Monitoring K8S resources](cka/monitoring/monitor-k8s-resources.md)
- [Pod volumes](cka/manage-volumes/pod-volumes.md)
- [Sample exam](cka/sample-exam/sample-exam.md)

### Kubernetes CNCF Projects Related

- [Deep dive admission controllers](k8s/explore-admission-controllers/readme.md)
- [Explore Antrea](k8s/explore-antrea/readme.md)
- [Explore cert manager](k8s/explore-cert-manager/readme.md)
- [Explore cluster API for docker infrastructure](k8s/explore-capd/readme.md)
- [Explore custom controller leader election](k8s/explore-controller-leader-election/readme.md)
- [Explore DRA](k8s/explore-dra/readme.md)
- [Explore ETCD](k8s/explore-etcd/readme.md)
- [Explore Helm](k8s/explore-helm/readme.md)
  - [helm-client-in-controller](k8s/explore-helm/helm-client-in-controller/readme.md)
- [Explore k14s](k8s/explore-k14s/readme.md)
- [Explore kapp controller](k8s/explore-kapp-controller/readme.md)
- [Explore Karmarda (WIP)](k8s/explore-karmarda/readme.md)
- [Explore Kata Container](k8s/explore-kata-container/readme.md)
- [Explore kubevirt](k8s/explore-kubevirt/readme.md)
- [Explore KUDO](k8s/explore-kudo/readme.md)
- [Explore Kustomize](k8s/explore-kustomize/readme.md)
- [Explore pinniped](k8s/explore-pinniped/readme.md)
  - [Authentication in K8S](k8s/explore-pinniped/auth-in-k8s.md)
- [Explore secrets store csi](k8s/explore-secrets-store-csi/explore-secrets-store-csi.md)
- [Explore Tilt Dev's live updates](k8s/explore-tilt-dev/readme.md)
- [Explore Vitess](k8s/explore-vitess/readme.md)
- [Expore Rancher](k8s/explore-rancher/readme.md)
- [Hashicorp Nomad](k8s/hashicorp-nomad/readme.md)
- [How pod is created via Deployment with the network configured](k8s/how-pod-created-with-network-configured/readme.md)
- [kube-apiserver-server-chain](k8s/kube-apiserver-server-chain/readme.md)
- [Kubernetes LeaderWorkerSet (LWS) & DisaggregatedSet (DS)](k8s/explore-leaderworkerset/readme.md)
- [List-Watch](k8s/list-watch/readme.md)
- [Pod cold start performance](k8s/pod-cold-start-performance/readme.md)
- [StatefulSets](k8s/explore-statefulset-on-vsphere/explore-statefulset-on-vsphere.md)

### Kubernetes Related Papers

- [[Paper] Evaluating Kubernetes Performance for GAI Inference](k8s-papers/evaluating-k8s-perf-for-gai-inference/readme.md)

### General Knowledge Base

- [Compare Redis and Memcached](general-knowledge-base/compare-redis-memcached/readme.md)
- [Conflict-free Replicated Date Types](general-knowledge-base/conflict-free-replicated-data-types/readme.md)
- [Distributed consensus](general-knowledge-base/distributed-consensus/readme.md)
  - [Deep dive into config change in distributed system](general-knowledge-base/distributed-consensus/deep-dive-config-change.md)
  - [Raft distributed consensus](general-knowledge-base/distributed-consensus/raft-distributed-consensus.md)
- [Distributed hash table](general-knowledge-base/distributed-hash-table/readme.md)
- [Distributed lock](general-knowledge-base/distributed-lock/readme.md)
- [Explore Confidential VMs](general-knowledge-base/confidential-vm/readme.md)
- [Explore Google SaaS Runtime](general-knowledge-base/google-saas-runtime/readme.md)
- [Explore OAuth](general-knowledge-base/explore-oauth/readme.md)
- [Explore OpenLDAP](general-knowledge-base/explore-openldap/readme.md)
- [Go channels](general-knowledge-base/go-channels/readme.md)
- [How tree is stored in database](general-knowledge-base/how-tree-is-stored-in-db/readme.md)
- [Idempotency](general-knowledge-base/idempotency/readme.md)
- [Logging in Cobra](general-knowledge-base/logging-with-cobra/readme.md)
- [Multi-tenancy](general-knowledge-base/multi-tenancy/readme.md)
- [Optimistic vs Pessimistic concurrency control](general-knowledge-base/optimistic-pessimistic-concurrency-control/readme.md)
- [Row vs Column oriented databases](general-knowledge-base/row-vs-column-oriented-database/readme.md)
- [RPC vs REST](general-knowledge-base/rpc-vs-rest/readme.md)
- [Something about gossip-protocol](general-knowledge-base/something-about-gossip-protocols/readme.md)
- [UUID](general-knowledge-base/uuid/readme.md)

### System Design

- Architecture & Patterns
  - [Distributed systems architectural patterns](system-design/distributed-system-architectural-patterns/readme.md)
- Data-Intensive Applications (Book Notes)
  - [Storage and retrieval](system-design/3-storage-and-retrieval/readme.md)
    - [Column oriented storage](system-design/3-storage-and-retrieval/column-oriented-storage.md)
    - [Compare between B-Tree and LSM tree](system-design/3-storage-and-retrieval/compare-b-tree-vs-lsm-tree.md)
    - [Log structured storage](system-design/3-storage-and-retrieval/log-structured-storage.md)
    - [OLTP VS OLAP](system-design/3-storage-and-retrieval/oltp-vs-olap.md)
    - [Other indexing structures](system-design/3-storage-and-retrieval/other-indexing-structures.md)
    - [Page oriented storage](system-design/3-storage-and-retrieval/page-oriented-storage.md)
  - [Encoding and evolutiion](system-design/4-encoding-and-evolution/readme.md)
  - [Replication](system-design/5-replication/readme.md)
    - [How to handle concurrent write](system-design/5-replication/how-to-handle-concurrent-write.md)
    - [Anti-Entropy](system-design/5-replication/manage-anti-entropy-with-merkle-tree.md)
  - [Partitioning](system-design/6-partitioning/readme.md)
  - [Trasactions](system-design/7-transactions/readme.md)
  - [Consistency and consensus](system-design/9-consistency-and-consensus/readme.md)
  - [Stream processing](system-design/11-stream-processing/readme.md)
- System Design Case Studies
  - [Auto complete service](system-design/topics/auto-complete-service/readme.md)
  - [Caching](system-design/topics/caching/readme.md)
  - [Collaborative editing](system-design/topics/how-collaborative-editing-work/readme.md)
  - [Design a distributed delayed job queueing system](system-design/topics/distributed-delayed-job-queueing-system/readme.md)
  - [Design a key-value store](system-design/topics/distributed-key-value-store/readme.md)
  - [Design distributed message broker(RabbitMQ) and message streaming platform(Kafka)](system-design/topics/message-broker-and-event-streaming/readme.md)
  - [Design i18n service](system-design/topics/i18n-service/readme.md)
  - [Design instant messaging system](system-design/topics/instant-messaging-system/readme.md)
  - [Design Netflix or Youtube](system-design/topics/netflix/readme.md)
  - [Design news feeds system](system-design/topics/news-feeds/readme.md)
  - [Design payment system](system-design/topics/payment-system/readme.md)
  - [Design text based search service](system-design/topics/text-based-search/readme.md)
  - [Distributed counter](system-design/topics/distributed-counter/readme.md)
    - [CRDT Distributed Counter](system-design/topics/distributed-counter/crdt-distributed-counter.md)
  - [Distributed file system](system-design/topics/distributed-file-system/readme.md)
  - [Distributed log aggregation](system-design/topics/distributed-log-aggregation/readme.md)
  - [Distributed monitoring system](system-design/topics/monitoring-system/readme.md)
    - [Mimir](system-design/topics/monitoring-system/mimir.md)
    - [Prometheus](system-design/topics/monitoring-system/prometheus.md)
    - [TSDB](system-design/topics/monitoring-system/tsdb.md)
  - [Distributed Unique ID](system-design/topics/distributed-unique-id/readme.md)
  - [Distributed web crawler](system-design/topics/distributed-web-crawler/readme.md)
  - [Geolocation based service](system-design/topics/geolocation-based-service/readme.md)
  - [Rate limiting](system-design/topics/rate-limiting/readme.md)
  - [Real time interactions on live video](system-design/topics/realtime-interactions-on-live-video/readme.md)
  - [Realtime comments on live video](system-design/topics/realtime-comments-on-live-video/readme.md)
    - [Long Polling vs SSE vs WebSocket](system-design/topics/realtime-comments-on-live-video/long-polling-vs-sse-vs-websocket.md)
  - [Realtime gaming leaderboard](system-design/topics/realtime-gaming-leaderboard/readme.md)
  - [Realtime presence platform](system-design/topics/realtime-presence-platform/readme.md)
  - [Stock Exchange](system-design/topics/stock-exchange/readme.md)
  - [What questions to ask at the beginning of a design](system-design/topics/what-to-ask-at-beginning-of-a-design.md)

### How Facebook Builds Systems

- [FOQS: Scaling a distributed priority queue](how-facebook-xxx-series/scale-a-distributed-priority-queue/readme.md)
- [Gorilla: A Fast, Scalable, In-memory Time Series Database](how-facebook-xxx-series/gorilla-in-memory-tsdb/readme.md)
- [Manage datastore locality at scale with Akkio](how-facebook-xxx-series/managing-data-store-locality-at-scale-with-akkio/readme.md)
- [Migrating Messenger storage to optimize performance](how-facebook-xxx-series/migrate-messenger-storage/readme.md)
- [Scribe: Transporting petabytes per hour via a distributed, buffered queueing system](how-facebook-xxx-series/buffered-queueing-system-for-log-transporting/readme.md)
- [TAO: Facebook’s Distributed Data Store for the Social Graph](how-facebook-xxx-series/distribute-datastore-for-social-graph/readme.md)
- [Twine: A Unified Cluster Management System for Shared Infrastructure](how-facebook-xxx-series/cluster-management-system/readme.md)

### How Google Builds Systems

- [MillWheel: Fault-Tolerant Stream Processing at Internet Scale](how-google-xxx-series/millwheel-stream-processing-framework/readme.md)
- [Monarch: Google's Planet-Scale In-Memory Time Series Database](how-google-xxx-series/monarch-planet-scale-in-memory-time-series-database/readme.md)

### How Amazon Builds Systems

- [Dynamo](how-amazon-xxx-series/dynamo/readme.md)

### How Uber Builds Systems

- [Peloton: Uber’s Unified Resource Scheduler for Diverse Cluster Workloads](how-uber-xxx-series/peloton-unified-resource-scheduler/readme.md)

### How Alibaba Builds Systems

- [Virtual Control Plane implementation: A Multi-Tenant Framework for Cloud Container Services](how-alibaba-xxx-series/virtual-control-plane/readme.md)

