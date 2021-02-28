# Distributed file system

## Requirements

- Be able to store large files, several TBs
- Write operations:
  - Create a new file
  - Append to an existing file
  - Update an existing file
  - Handle concurrent writes
- Read operations:
  - Streaming read
  - Random read at any arbitrary offset
  - Handle concurrent reads
- HA

## Design overview

- Split large files into small chunks with fixed size, e.g. 64 MB.
  - Each chunk is tracked by `chunkID` (called `handle` in GFS)
- Store the chunks on different distributed servers as the Linux files.
- Have a control plane to:
  - Generate the `chunkIDs`.
  - Allocate chunk servers for storing the chunks.
  - Maintain the chunk servers discovery metadata.
    - Master maintains two mappings:
      - file name -> array of chunk ids/handlers
      - id/handler -> [`list of chunk servers`, `version #`, `primary chunk server`, `lease expiration`]
- Have a client lib which is used by application for data IO. No data goes through control plane node. 

### High level architecture 

![gfs-architecture](resources/gfs-architecture.png)

![gfs-architecture-2](resources/gfs-architecture-2.png)

### Write flow

- Application wants to write a new file(`fileOffset=0`) or append data to an existing file(`fileOffset=filesize`).
  - Application uses client lib to calculates the `chunk index` based on offset.
  - Application uses client lib to send the write request(`file name, chunk index`) to control plane.
- Control plane receives the request.
  - From the file name and chunk index to get the `chunk ID`. Generate a new `chunk ID` if no entry found in the map, it
    means that the write operation is writing a new file.
  - From the `chunk ID` to get the `list of chunk servers`.
  - Return the `chunk ID` and `chunk server locations` back to client.(The chunk servers hold the replicas)
- Client writes data directly to **all** chunk servers with the request(`chunk ID`, `chunkOffset`, `data`). See
  more details on [how to handle concurrent writes](#how-to-handle-concurrent-writes) and [how to handle write failure](#how-to-handle-write-failure)

#### how to handle concurrent writes

Multiple applications could write data to the same file at the same time, we want data to be consistent between all
replicas. There are two principles:

- All replicas need to apply the change in same order.
- Version number could be used for optimistic concurrency control.

The problem becomes the distributed consensus problem. We could choose to use leader based replica solution and use [Raft
algorithm](../../../distributed-consensus/raft-distributed-consensus.md) to guarantee the data consistency among all
replicas. So that during write operation, client could write to leader and leader will take care of the date replication.
This helps reduce the network traffic between the client and chunk server, the network bandwidth within chunk servers would
be fast because it is the internal network.

#### how to handle write failure

### Reads

- The client sends file name and offset to the master
- The master sends the handler and list of chunk servers to the client. The client also caches the result for future reads
- The client chooses the closest chunk server to read. 
- The client sends the handler and offset to that chosen chunk server (each chunk of file is stored as a linux file in
  the hard drive) to read te desired range of bytes of the file

### Writes (record appends)

#### No primary chunk server elected

- The master finds up to date replicas (compare the version # of each replica to see if it equals the version # master knows).
  And the version # is persistent on disk, so it does not get lost on crash.
- The master picks one as the primary chunk server with expiration and the rest as secondary.
- The master increments the version # and stores in disk.
- The master tells primary and all the secondaries the data to be written and the version #.
- The primary and all secondaries write the data to a temp location, after they all say YES I have the data, the data
  will be appended to the chunk file.
- Primary returns SUCCESS to client only when the primary and all secondaries have successfully written the data.

## References

- <https://www.youtube.com/watch?v=EpIgvowZr00&ab_channel=MIT6.824%3ADistributedSystems>
