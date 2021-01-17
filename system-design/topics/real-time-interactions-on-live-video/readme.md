# Real time interactions on live video

This note is based on a InfoQ talk about [Streaming a Million Likes/Second: Real-Time Interactions on Live
Video](https://www.youtube.com/watch?v=yqc3PPmHvrA&ab_channel=InfoQ) from Linkedin.

![workflow-1](resources/workflow-1.png)
![workflow-2](resources/workflow-2.png)
![workflow-3](resources/workflow-3.png)

## How to stream

The persistent connection is using `HTTP Long Poll` with Server Sent Events.

- Client sends `GET` request with `Accept: text/event-stream`
- Server respons `200 OK` with `Content-Type: text/event-stream`
- Connection is established without closing
- Server sends `data:{"like", object}` or `data:{"comment", object}` to client

## Challenges

### Connection management with tons of connections

![multi-devices-challenge](resources/multi-devices-challenge.png)

Linkedin uses Akka and Play framework for connection management.

![connection-mgmt-1](resources/connection-mgmt-1.png)
![connection-mgmt-2](resources/connection-mgmt-2.png)
![connection-mgmt-3](resources/connection-mgmt-3.png)
![connection-mgmt-4](resources/connection-mgmt-4.png)
![client-sees-likes](resources/client-sees-likes.png)

### Subscriptions

We could not blindly broadcast the `likes` to all clients, because different users are watching different live videos.

![viewers-sub](resources/viewers-sub.png)
![state-after-sub](resources/state-after-sub.png)

### 10K or more viewers

Add an abstraction between clients and backend, known as `frontend server`

![scale-dispatch-1](resources/scale-dispatch-1.png)
![scale-dispatch-2](resources/scale-dispatch-2.png)
![scale-dispatch-3](resources/scale-dispatch-3.png)
![scale-dispatch-4](resources/scale-dispatch-4.png)

### Dispatcher is the bottleneck

How to handle the 1000 likes persecond? We could have multiple dispatcher nodes, and allow a balanced number of clients
to be connected to dispatcher nodes. All `likes` could be sent to any dispatcher nodes and render to clients. But this
requires to pull out the `in-memory` mapping table out to its own key-value store.

![dispatcher-bottleneck-1](resources/dispatcher-bottleneck-1.png)

### Multi data centers

![multi-datacenter-1](resources/multi-datacenter-1.png)

#### Cross data center subscriptions

#### Publish likes to all data centers

![publish-to-all-dc-1](resources/publish-to-all-dc-1.png)
![publish-to-all-dc-2](resources/publish-to-all-dc-2.png)
![publish-to-all-dc-3](resources/publish-to-all-dc-3.png)

## References

- <https://www.youtube.com/watch?v=yqc3PPmHvrA&ab_channel=InfoQ>
