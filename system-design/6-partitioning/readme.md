# Partitioning

If no partitioning, there are several issues:

- The increasing dataset could not fit in one node unless we do vertical scaling.
- In leader based replication, one node(leader) could be the hotspot with large amount of writes.
- In leader-less replication, clients close to one data center might send all requests to one node which causes the hotspot.

The main reason for wanting to partition data is **scalability**. And partition is always along with replication to make sure the availability of data.

![partitioning-with-replication](./resources/partitioning-with-replication.png)

## Partitioning of Key-Value Data

### Partitioning by Key Range

Assign a continuous range of keys as one partition. E.g. userID from 0-10000 to one node.

Pros:

- Keys within a partition is sorted, so range query is easy.

Cons:

- The range boundaries could not be easily determined (usually not evenly spaced).
- Even we have the boundaries well defined, it does not scale for long term.
- A particular access pattern could also lead to hot spots. E.g. clients are more interested in a particular range of data than the other.

### Partitioning by Hash Key

Using hash function to have a relatively evenly distributed hash value, store a range of hash value on a particular node.

![partition-by-hash-key](./resources/partition-by-hash-key.png)

- MongoDB uses MD5
- Cassandra uses Murmur3
- Voldemort uses Fowler-Noll-Vo function

**Important**: Some programming language's built-in hash function is not suitable for partitioning. Java's `Object.hashCode()` could have different hash values in different processes.

Pros:

- Keys are relatively well distributed. In theory, the reqeusts would be spread out to all partitions(nodes) evenly.

Cons:

- For range query, we need to send requests to all partitions. Because the adjacent keys are all distributed to different partitions.
- Choosing the boundaries for the range of hash keys is not easy.
- If there is a hot key, e.g. celebrities on twitter. We could not avoid the hot spot completely.

#### Solutions for range query

Using multiple columns as a compound primary key, only the first part of that key is hashed to determine the partition, the rest of columns are sorted to provide a better range query performance. E.g. `(userID, updateTimestamp)`, we use `userID` as the hash key to find the partition, and `updateTimestamp` is sorted. So that we could easily query the entries within an interval of a particular user.

The only problem is that all the data of a particualr user is stored within one partition which could cause the `celebrity problem`.

#### Solution for celebrity problem

A hot topic or a celebrity could cause the requests for that topic or celebrity to be directed to the same partition. A simple solution could be adding a random decimal number at the beginning or end of the key, e.g. two-digit decimal number could generate 100 unique keys, and allow those keys to be distributed across different partitions. Now reading needs to read from all 100 keys and combine it.

## Partitioning of Secondary Indexes

Sometimes people would add secondary index for a better performance on filtering, e.g. query on the cars which has red color. But how the secondary indexes are partitioned ?

### Document based partitioning or Local index

![document-based-partitioning](./resources/document-based-secondary-index-partitioning.png)

Each partition maintains its own secondary indexes which cover only the data in that partition. On write, only add/update/delete the secondary indexes on the partition where data is located. On read, need to read from all partitions and combine the result.

Pros:

- Write is efficient
  
Cons:

- Read is expensive

### Term based partitioning or Global index

![term-based-partitioning](./resources/term-based-secondary-index-partitioning.png)

Instead of letting each partition has its own secondary indexes, we construct a global one and store it on one of the patition. Which partition should store the secondary indexes ? We could partition by the term or a hash of the term.

Pros:

- Read is efficient, can just read from one partition

Cons:

- Write is slower and complicated, because multiple partitions are involved in a single write. Since the complication of write across multiple partitions, there would have some slowness in terms of data consistency(the global indexes might not be ready on immediately read).

## Rebalancing Partitions

### DO NOT USE HASH MOD N(number of nodes or number of partitions)

Need to say three times that **DO NOT USE HASH MOD N !** **DO NOT USE HASH MOD N !** **DO NOT USE HASH MOD N !** The reason is that when `N` changes, most of the keys will need to be moved from one node to another. One important rule is to move as less as possible. We use consistent hashing !

### Fixed number of patitions

![fix-partition-rebalancing](./resources/fix-partition-rebalancing.png)

We have `10` nodes, and fix the total number of partitions to be `1000`, so we have `100` partitions per node.

- **Add new node**: the new node copy a few patitions from every existing node until all the partitions are fairly distributed again, then the new node starts serving and old partitions could be removed. During the copying, the old partitions are still serving the read and write requests.
- **Drain an existing node**: we copy the partitions from current node evenly to other nodes. Current node still serves the requests until the data move is completed.

This approach is used by Elasticsearch, Couchbase and Voldemort.

Pros:

- No need to split partitions, and reduces the complexity of implementation.

Cons:

- Hard to pick the right number of partitions.

### Dynamic partitioning

Set a max and min ahead. When the data within one partition grows over the max, it is split into two partitions. One of the two halves can be transferred to another node in order to balance the load. When the data shrinks below the min, it can be merged into an adjacent partition. DBs like MongoDB and HBase usually set a `pre-spliting` to make sure an empty database can also have a pre-defined number of partitions. This is to mitigate the issue when empty database with multiple nodes, there is a period of time that all reqests are sent to that single partition until the partition split happens.

- **Add new node**: the new node copy a few patitions from every existing node until all the partitions are fairly distributed again, then the new node starts serving and old partitions could be removed. During the copying, the old partitions are still serving the read and write requests.
- **Drain an existing node**: we copy the partitions from current node evenly to other nodes. Current node still serves the requests until the data move is completed.
- **Data increase over the max**: Partition splits and one of the two halves will be assigned to another node.
- **Data delete below the min**: Partition merges with an adjacent partition on the same node.

### Have fixed number of partitions per node

E.g. `100` partitions per node.

- **Add new node**: It randomly choose existing partitions on all nodes to split.
- **Drain a node**: It copies the partitions randomly to other partitions.

## Request Routing

Client knows the key to query, and send the request to backend service. How does the backend service know which partition and which node the data stays on ?

Using the hash function, we could calculate the hash value of a key. Once we have the hash value, we could know which partition holds the key (This could be easitly done by consistent hashing, which is next clockwise partition greater than current hash value position). The system needs to maintain a mapping between partition and node IP(`Zookeeper`), so we could easily know which node we need redirect the requests to.

Usually there are three approaches:

![service-discovery](./resources/service-discovery.png)

If using `ZooKeeper`, it will track the cluster metadata. When a partition changes ownership, a node is added/removed, `ZooKeeper` notifies the routing tier so that it can keep its routing info up to date.

![zookeeper-service-discovery](./resources/zookeeper-service-discovery.png)

## How consistent hash work

TBA

## How to partition a tree or graph data

TBA
