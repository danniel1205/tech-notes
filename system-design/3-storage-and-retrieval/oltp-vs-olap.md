# OLTP VS OLAP

## What is OLTP

In Online transaction processing (OLTP), information systems typically facilitate and manage transaction-oriented
applications. An automated teller machine (ATM) for a bank is an example of a commercial transaction processing application.

An OLTP system is an accessible data processing system in today's enterprises. Some examples of OLTP systems include order
entry, retail sales, and financial transaction systems.

Online transaction process concerns about concurrency and atomicity.

## What is OLAP

Online analytical processing(OLAP) is an approach to answer multi-dimensional analytical (MDA) queries swiftly in computing.
Typical applications of OLAP include business reporting for sales, marketing, management reporting, business process management
(BPM), budgeting and forecasting, financial reporting and similar areas, with new applications emerging, such as agriculture.

OLTP is typically contrasted to OLAP (online analytical processing), which is generally characterized by much more complex
queries, in a smaller volume, for the purpose of business intelligence or reporting rather than to process transactions.
Whereas OLTP systems process all kinds of queries (read, insert, update and delete), OLAP is generally optimized for read
only and might not even support other kinds of queries.

![oltp-vs-olap](./resources/oltp-vs-olap.jpg)

The telemetry data is usually sent to data warehouse for business analysis. One of the popular products is
[tableau](https://www.tableau.com/).

## Star schema (dimensional modeling)

When customers want to dump data into data warehouse, they usually have to define the data schemas. There usually have
a centralized table named `fact_table`, and `dimentional_tables` which the central table has references to.

## Pain from query against row oriented table

One of the pain BA has is the slowness of the query. Because BA usually queries from large set of data, and each query
has to load the entire row if the table is `row-oriented`. That is why we have introduced the
[column-oriented storage](./column-oriented-storage.md)

## References

- <https://en.wikipedia.org/wiki/Online_transaction_processing>
- <https://en.wikipedia.org/wiki/Online_analytical_processing>
