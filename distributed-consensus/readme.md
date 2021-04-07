# Distributed consensus

## Distributed system

<https://en.wikipedia.org/wiki/Distributed_computing>
A distributed system is a system whose components are located on different networked computers, which communicate and
coordinate their actions by passing messages to one another. The components interact with one another in order to achieve
a common goal. Three significant characteristics of distributed systems are: concurrency of components, lack of a global
clock, and independent failure of components.

## Properties of distributed system

- Concurrency: Each computer executes events independently at the same time.
- Lack of a global clock: There is no single global clock that determines the sequence of events happening across all
  computers in the network.
- Independent failure of components: It’s impossible to have a system free of faults.
  - Crash-fail: The component stops working without warning (e.g., the computer crashes).
  - Omission: The component sends a message but it is not received by the other nodes (e.g., the message was dropped).
  - Byzantine: The component behaves arbitrarily.
- Message passing
  - Sync
  - Async

## Details of distributed consensus

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

### Byzantine fault tolerant

Both Paxos and Raft are not Byzantine fault-tolerant. The paper [Byzantine General’s Problem](https://people.eecs.berkeley.edu/~luca/cs174/byzantine.pdf)
by Leslie Lamport, Robert Shostak, and Marshall Pease provided the first proof to solve the Byzantine General’s problem:
it showed that a system with `x` Byzantine nodes must have at least `3x + 1` total nodes in order to reach consensus
`(Only works in synchronous environment)`.

We’re going to take a look at two algorithms (DLS and PBFT) that brought us closer than ever before to breaking the
Byzantine + asynchronous barrier.

- [DLS algorithm](https://groups.csail.mit.edu/tds/papers/Lynch/jacm88.pdf)
- [PBFT algorithm](http://pmg.csail.mit.edu/papers/osdi99.pdf)
- Nakamoto Consensus: Instead of every node agreeing on a value, f(x) works such that all of the nodes agree on the
  probability of the value being correct. Rather than electing a leader and then coordinating with all nodes, consensus
  is decided based on which node can solve the computation puzzle the fastest.

## References

- <https://www.preethikasireddy.com/post/lets-take-a-crack-at-understanding-distributed-consensus>