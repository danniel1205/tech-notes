# Row vs Column oriented databases

## Row oriented databases

- Common databases: Postgres, MySQL.
- Data is stored row by row. The first column of a row is stored next to the last column of previous row.

  ![](resources/row-oriented-example.png)

- Write: New record is appended. Row oriented data store is commonly used by OLTP style application.

  ![](resources/row-oriented-write.png)

- Read: Can easily read entire row or multiple rows, but slow on selecting columns since it has to load unnecessary columns
  into memory.

## Column oriented databases

- Common databases: Redshift, BigQuery, Snowflake.
- Rows of each column is stored together.

  ![](resources/column-oriented-example.png)

- Write: New record needs to be split out and inserted into proper position.

  |   |   |   |
  |---|---|---|
  |Jane|Vancouver|33|

  - If data is on a single disk, write needs to load all data into memory.

  ![](resources/column-oriented-write.png)

  - If data is split out and distributed, write is more efficient.

  ![](resources/column-oriented-write-data-distributed.png)

- Read: read from single disk or continuous memory address which is very efficient. Column oriented data store is commonly
  used by OLAP style application.

- Has a write store which data could be appended on write, and then read-optimized to a read store which could sort data
  in arbitrary order.

  ![](resources/ws.png)

  ![](resources/rs.png)

## References

- <https://dataschool.com/data-modeling-101/row-vs-column-oriented-databases/>
