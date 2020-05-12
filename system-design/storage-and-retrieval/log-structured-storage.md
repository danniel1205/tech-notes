## Hash Index
Maintain a **in memory** hash table. Key is the index key (e.g. the key if data is key, value pairs). Value is the offset 
of the value in file. So each time we want to read the data, we could get the file offset from hash table and find the 
starting offset where the latest data is stored.

### Log segments
### SSTables and LSM-Trees
