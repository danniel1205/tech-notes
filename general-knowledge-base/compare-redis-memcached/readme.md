# Compare Redis and Memcached

- `redis` provides a superset of features comparing to `memcached`
- To implement `memcached` cluster, client needs to solve traffic rounting, e.g. using consistent caching
- They have different implementations on memory management

More details could be found from the links below in references section.

## References

- [Redis vs Memcached (by AWS)](https://aws.amazon.com/elasticache/redis-vs-memcached/)
- [Redis vs Memcached (by Alibaba Cloud)](https://alibaba-cloud.medium.com/redis-vs-memcached-in-memory-data-storage-systems-3395279b0941)
- [Cache Consistency: Memcached at Facebook (MIT Lecture)](https://www.youtube.com/watch?v=Myp8z0ybdzM&ab_channel=MIT6.824%3ADistributedSystems)
- [Paper: Scaling memcache at Facebook](resources/memcache-fb.pdf)