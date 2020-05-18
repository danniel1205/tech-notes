# Log structured storage

## Naive solution

- On write: Append to log file
- On read: Serach in log file

### Problems on write

- Log file size can be huge

### How to improve on write

- Have log segements
- Async compaction and segements merging
- For deleting records, append a tombstone record and ignore old records on compaction and merging

### Problems on read

### How to improve on read

- Hash Index:
Maintain a **in memory** hash table. Key is the index key (e.g. the key if data is key, value pairs). Value is the offset
of the value in file. So each time we want to read the data, we could get the file offset from hash table and find the
starting offset where the latest data is stored.

### Other problems

## SSTables and LSM-Trees
