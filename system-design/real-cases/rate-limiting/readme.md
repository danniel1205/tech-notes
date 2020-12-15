# Rate limiting

## Terminologies

- Fail close system: A system is set to shut down and prevent further operation when failure conditions are detected.
- Fail open system: A system set to fail open does not shut down when failure conditions are present. Instead, the system remains “open” and operations continue as if the system were not even in place.

## Why rate limiting is needed

- Protect shared services from **excessive use** or prevent **resource starvation**(from malicious or non-malicious DoS)
- Managing policies and quotas. When the capacity of a service is shared among many users or consumers, it can apply rate limiting per user to provide fair and reasonable use, without affecting other users.
- Controlling flow. For example, you can distribute work more evenly between workers by limiting the flow into each worker, preventing a single worker from accumulating a queue of unprocessed items while other workers are idle.
- Avoiding excess costs. You can use rate limiting to control costs⁠ in the service which has traffic based auto-scaling feature.

## Strategies

### No rate limiting

No rate limiting is the floor that the design needs to consider in worst-case situation. Using timeouts, deadlines, circuit-breaking pattern helsp your service to be more robust in the absense of rate limiting.

### Pass through

The service calls other service to fulfill requests. `429 Too Many Requests` http response might be used to return to the caller.

- Just forward the rate-limiting response from downstream back to the caller.
- Enforce the rate limits on behalf of the downstream service and block the caller.

### Enforce rate limits

Put the rate limits in place to protect current service or the downstream service.

To enforce rate limiting, first understand why it is being applied in this case, and then determine which attributes of the request are best suited to be used as the limiting key (for example, source IP address, user, API key). After you choose a limiting key, a limiting implementation can use it to track usage. When limits are reached, the service returns a limiting signal (usually a 429 HTTP response).

### Defer response

When under the high traffic responding to the caller's request is also a challenge.

- The service could response a simple job ID which could be used by caller to poll the request status.
- The caller could register a callback, the service will call it once the response is ready.
- The caller could subscribe to a event channel where the service will send the response into.

### Client side strategies

If the backend service does not provide the rate-limiting, the client could apply self-imposed throttling.

- Apply exponential backoff with random offset(jitter) on retries.

## How to implement rate limiting | Techniques for enforcing rate limit

### Token bucket

- Use a counter to indicate the number of available tokens
- Each time a request comes, it needs to "consume a token"(decrease the token counter)
- Refill the token(increase the token counter) at some rate

[Java sample implementation](https://github.com/vladimir-bukhtoyarov/bucket4j/blob/master/doc-pages/token-bucket-brief-overview.md)

### Leaky bucket

![leaky-bucket](./resources/leaky-bucket.png)

- The requests are consumed with a fixed rate
- If more requests come, it would queued up (uber implements this by using sleep, rather than discarding the leaking requests)

This is similar to token bucket, if no tokens are available, we could put the request to sleep until the tokens are refilled. Or we could discard the request and return `429 Too Many Requests` back to client.

[Golang implementation](https://github.com/uber-go/ratelimit/blob/master/ratelimit.go)

#### Pros of Token Bucket and Leaky bucket

- Could smooth out bursts at an approximately average rate
- Easy to implement on single node

#### Cons of Token Bucket and Leaky bucket

- Burst traffic could make so many requests sleep which causes high thread consumption
- There is no guarantee how long the requests could be processed

### Fixed window

![fixed-window](./resources/fixed-window.png)

- Divide the timeline into a fixed window. e.g. 60s or 3600s
- Define the rate for each window. e.g. 10 requests/3600s
- Use a counter to count the number of requests for a particular window
  - For each request, use floor(current_time) to decide which window it belows.
  - e.g. 12:00:03 would be in 12:00:00 window
- If counter exceeds the limit, discard the request (return `429` Http code)
- In a new time window, the counter gets reset

#### Pros

- Make sure the most recent requests could be processed in a new time window, it prevents starvation from using token bucket solution
- Memory efficient since we just use counter

#### Cons

- The bursts would happen near the boundary. e.g. spike at 12:59, and another spike at 1:00
- Requests to be retried could easily fill up next window and cause the spike and another amount of requests to be retried

### Sliding log

### Sliding window

## Rate limiting in K8S

## References

- <https://cloud.google.com/solutions/rate-limiting-strategies-techniques>
- <https://konghq.com/blog/how-to-design-a-scalable-rate-limiting-algorithm/>
- <https://medium.com/nlgn/design-a-scalable-rate-limiting-algorithm-system-design-nlogn-895abba44b77>
- <https://www.figma.com/blog/an-alternative-approach-to-rate-limiting/>
- [Youtube: Leaky Bucket vs Token Bucket](https://www.youtube.com/watch?v=bL0I54Vac9Q&ab_channel=AvinashKokare-CS-ITTutorials)
- [Private repo: Golang rate limiter implementation](https://github.com/danniel1205/rate-limiter)
