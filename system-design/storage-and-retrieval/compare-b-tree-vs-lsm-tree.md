# Compare between B-Tree and LSM tree

## Terminologies

- `write amplification`: one write to database resutls in multiple writes on disk

## Comparision

- LSM Tree is **faster** on write than B-Tree
  - B-Tree writes twice: once is to write-ahead-log, another is to tree page
  - B-Tree writes the entire page even there are only few bytes in that page changed
  - LSM as lower `write amplification`

- LSM Tree can be compressed bettern than B-Tree, so that LSM Tree usually produce smaller files on disk
  - B-Tree usually has some spaces unsued due to fragmentation

- LSM Tree is **slower** on read than B-Tree
  - LSM Tree has to check `in-memory` LSM Tree first, and then latest segments

- LSM Tree might cause high write throughput due to compaction

- LSM Tree might have multiple copies of a key in different segments, B-Tree has each key exists exactly in one place 