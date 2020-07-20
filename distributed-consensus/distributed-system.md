---
tags: distributed-system
---

# Distributed system

<https://en.wikipedia.org/wiki/Distributed_computing>
A distributed system is a system whose components are located on different networked computers, which communicate and coordinate their actions by passing messages to one another. The components interact with one another in order to achieve a common goal. Three significant characteristics of distributed systems are: concurrency of components, lack of a global clock, and independent failure of components.

## Properties of distributed system

- Concurrency: Each computer executes events independently at the same time.
- Lack of a global clock: There is no single global clock that determines the sequence of events happening across all computers in the network.
- Independent failure of components: It’s impossible to have a system free of faults.
  - Crash-fail: The component stops working without warning (e.g., the computer crashes).
  - Omission: The component sends a message but it is not received by the other nodes (e.g., the message was dropped).
  - Byzantine: The component behaves arbitrarily.
- Message passing
  - Sync
  - Async

## Distributed consensus

Symmetric, leader-less:

- All servers have equal roles
- Client can contact any server

Asymmetric, leader-based:

- Leader
- Follower
- Candidate: Candidate of a leader
- At any given time, one server is in charget, others accept its decision.
- Client communites with leader

---

- Paxos: [wiki](<https://en.wikipedia.org/wiki/Paxos_%28computer_science%29>)
- Raft: [notes](./raft-distributed-consensus.md)
