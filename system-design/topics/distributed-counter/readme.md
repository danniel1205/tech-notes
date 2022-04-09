# Distributed counter

This is applied to various use cases like YouTube view counts, Instagram post likes count, etc.

## User stories

- As a user, I would like to click on the `like` button on a post and see the counter gets incremented by 1.
- As a user, I would like to click on the `dislike` button on a post to unlike and see the counter gets decremented by 1.
- As a user, I would like to always see the post with its likes count from different devices. (HA)

### Cassandra stress test

![img.png](resources/cassandra-stress-test.png)
<https://netflixtechblog.com/benchmarking-cassandra-scalability-on-aws-over-a-million-writes-per-second-39f45f066c9e>

### Millions of websockets

- <https://alexhultman.medium.com/millions-of-active-websockets-with-node-js-7dc575746a01#:~:text=The%20theoretical%20limit%20is%2065k,*%2020k%20%3D%201%20mil).>
- <https://dzone.com/articles/load-balancing-of-websocket-connections>
