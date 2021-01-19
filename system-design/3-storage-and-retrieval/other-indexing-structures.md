# Other indexing structures

## Storing values within the index

Usually we will have secondaray index on other keys. And we might have multiple rows could be refered to the same secondary index key.
Like index key is `user_id`, there could be the duplicates of matching rows.

``` text
name: Bill, user_id: 123, city: San Jose
name: Bill, user_id: 123, city: San Francisco
```

We do not really want to store all matching rows in the index table as the value. We could have the reference to the rows
where it stored as the value. The place where rows are stored is known as `heap file`.

Storing only references to the data within index is called `nonclustered index`.

Storing all row data within the index is called `clustered index`.

## Multi-column indexes

- `contatenated index`: combines several fields into one key.
- `space-filling curve` could be used to B-Tree for multi-dimensional indexes. e.g search a restaurant in a two-dimensional
  range which standard B-Tree cannot answer it. If not using `space-filling curve`, we could use **R-trees**.

## Full-text search and fuzzy indexes

Questions: How to build index to support fuzzy queries. Like a misspelled word ?

## Keep everything in memory

In-memory database has higher performance is because they can avoid the overheads of encoding in-memory data structures
in a form that can be written to disk.
