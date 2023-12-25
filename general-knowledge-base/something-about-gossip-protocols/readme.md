# Something about gossip-protocol

## When does gossip-protocol stops

Gossip protocols disseminate information to a subset of peers in each round, not to all peers simultaneously. The
workflow has three major steps:

* Peer selection: Each node randomly selects a small number of peers (often called the "fanout") from its list of known
  neighbors.
* Information exchange.
* Forwarding and Convergence:
  * The receiving peers repeat the process, randomly selecting their own peers and further disseminating the information.
  * This creates a continuous, decentralized wave of information propagation throughout the network.
* Eventual Consistency: Over time, as nodes gossip and merge information, they eventually converge to a consistent state,
  even in the presence of network delays, failures, or partitions.

**But the question is when does gossip-protocol stops?**

Gossip protocols typically don't have an explicit "stop" mechanism. However, several conditions or scenarios can lead to
their effective cessation:

* Convergence:
  * State-based CRDTs: When all nodes in a state-based CRDT system have reached the same state, there's no longer a need
    for extensive gossiping. Protocols might continue with occasional checks for updates or changes, rather than
    continuous full-state exchange.
  * Operation-based CRDTs: If a period of inactivity occurs (no new operations generated), gossiping might naturally
    diminish as nodes exchange fewer updates.
* Timeout Mechanisms:
  * Messages deemed too old or "stale" might be discarded, reducing network traffic and potential for outdated
    information to circulate.
* Resource Constraints:
  * In resource-constrained environments, gossip protocols might be designed to stop or reduce activity when resources
    like bandwidth or processing power are low.
* External Control:
  * Gossip protocols are often integrated with external control mechanisms that can pause or terminate them based on
    specific events or conditions.

## How does gossip protocol avoid duplication

* Unique Identifiers:
  * Each message is assigned a unique identifier (ID) when it's created.
  * Nodes keep track of the IDs of messages they've already received and discard duplicates based on this information.
* Timestamps:
  * Messages can include timestamps indicating their creation time.
* Summary Vectors (Anti-Entropy):
  * Nodes maintain compact representations of the messages they've seen, called summary vectors or digests.
  * When two nodes exchange messages, they compare their summary vectors to identify missing or outdated information and
    only gossip those specific items.
* Probabilistic Forwarding:
  * Nodes don't forward every message they receive to every peer.
  * Instead, they use a probabilistic approach, forwarding messages to a randomly selected subset of neighbors.
  * This controlled randomness helps to prevent excessive flooding and duplication of messages.
* Gossip Fanout Control:
  * The number of peers a node gossips with in each round (fanout) can be adjusted based on network conditions and the
    desired convergence speed.
  * Lower fanout can reduce duplication but might slow down convergence, while higher fanout can speed up convergence
    but increase duplication risk.
* Gossip Protocol Variants:
  * Different gossip protocol variants have been designed to specifically address duplication issues:
    * Push-Pull Gossip: Nodes actively pull missing information from peers based on summary vector comparisons.
    * Rumor Mongering: Nodes track the age of messages and prioritize gossiping newer ones.
* Convergence Detection:
  * Some gossip protocols include mechanisms to detect when all nodes have reached a consistent state.
  * This can help to reduce unnecessary gossiping and prevent further duplication.

## References

* <https://systemdesign.one/gossip-protocol/>



