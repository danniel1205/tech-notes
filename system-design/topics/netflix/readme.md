# Design Netflix or Youtube

## Requirements

### Functional requirements

- Stream videos to user
  - One video could be watched by multiple users simultaneously
- Allow users to watch video previews for x seconds
- Allow users to search video
- Video recommendations

### Non-functional requirements

- HA: If one of the CDN node down, user could still watch the video. If a backend server crashes, user could still login
  and search videos

## Assumptions

- DAU
- Number of users are allowed to watch the same video at the same time ?
- What is the average size of video ?

## Data models

### User related

```go
type Account struct {
	CustomerID string // unique identifier for the account 
	Email string
	Password string // hashed string which stores the user logging password 
	Name string
	PlanID string // indicates which plan the account is subscribed to 
	Locale string // indicates the language used for i18n 
	...
}

type PaymentMethod struct {
	FullName string
	BillingAddress string
	CardNumber string // unique identifier of a payment method
	CVS string
	CustomerID string // linked to the account
}

type Billing struct {
	CustomerID string // which account the billing applies to
	Date Date // when the bill is filed
	Amount float32 // the amount has been charged 
	CardNumber string // which card used of the payment
}
```

### Video related

```go
type Video struct {
	VideoID string // unique identifier of a video
	Title string
	Subtitle string
	Description string
	SeriesName string // only applies to series
	Episode int // only applies to series
	Likes int
	MediaURL string // the endpoint to access the media ?
	...
}

type Comment struct {
	CustomerID string // the account who puts the comment 
	VideoID string // to which video the comment is applied to
	Content string
}

type ViewHistorys struct {
	Email string // the unique id of account
	History [] struct {
		Video Video
		Data string
	}
}
```

## Storage 

Netflix uses mySQL cluster and Cassandra

### Member Viewing History

Netflix uses Cassandra to store the viewing history.

#### First solution

RowKey: customerID
Column: each column is the viewing history

---
- On write: create a new column for each view history
- On read: read entire row or slice(range query)

Problems:

- read does not scale with the growth of user viewing history

Improvements:

- use EVCache to cache the result
  - customerID as the key
  - compressed viewing history as the value
  
#### Second solution

Above improvements still have problems when user data is huge, either the Cassandra row data is huge or the large data
could not fit into a single EVCache entry.

So the idea of LiveVH and CompressedVH is raised.

- Live or Recent Viewing History (LiveVH): Small number of recent viewing records with frequent updates. The data is
  stored in uncompressed form as in the simple design detailed above.
- Compressed or Archival Viewing History (CompressedVH): Large number of older viewing records with rare updates.
  The data is compressed to reduce storage footprint. **Compressed viewing history is stored in a single column per row
  key**.
  
---
- On write: New view history is written to LiveVH
- On read:
  - Recent Viewing History: For most cases this results in reading from LiveVH only, which limits the data size
    resulting in much lower latencies. 
  - Full Viewing History: Implemented as parallel reads of LiveVH and CompressedVH. Due to data compression and
    CompressedVH having fewer columns, less data is read thereby significantly speeding up reads.
- On LiveVH rollup: When the LiveVH hits the limit, the older viewing history is rolling up into the compressedVH. New
  row(rowID: customerID) will be created if there was no such row exists, or viewing history will be merged into existing
  compressedVH row.
  
![vh-rollup](resources/second-solution-vh-rollup.png)

Problems:

- With the growth of the data, the compressedVH could has a very large row size

Improvements:

- Break the compressedVH into chunks and store them in different row, so that it could be processed in parallel
  - rowID: CustomerId$Version$ChunkNumber
  - On write: rolled up compressed data is split into multiple chunks based on a configurable chunk size. All chunks are
    written in parallel to different rows with row key:CustomerId$Version$ChunkNumber. Metadata is written to its own
    row with row key:CustomerId after successful write of the chunked data. This bounds the write latency to two writes for
    rollups of very large viewing data. In this case the metadata row has an empty data column to enable fast read of metadata.
  - On read: The metadata row is first read using CustomerId as the key. For the common case, the chunk count is 1 and the
    metadata row also has the most recent version of rolled up compressed viewing data. For the rare case, there are
    multiple chunks of compressed viewing data. Using the metadata information like version and chunk count, different
    row keys for the chunks are generated and all chunks are read in parallel. This bounds the read latency to two reads.

![compressed-vh-chunks](resources/compressed-vh-chunks.png)

There are more improvements have been done, please refer to this [blog](https://netflixtechblog.com/scaling-time-series-data-storage-part-ii-d67939655586)
for more details.

![viewing-datastore-rearchitecture](resources/viewing-datastore-rearchitecture.png)

### Comments

It might be a good idea to use NoSQL database to store comments as well since we have already been using wide column
database for viewing history. So it would be easier for dev team to develop and maintain the same tech stack. If using
wide column, then the `rowKey` could be the `videoID`, the `columnKey` could be `customerID`. 

#### First solution

RowKey: videoID
Column: timestamp + customerID

---
- On write: create a new column for the new comment
- On read: read entire row or slice(range query)

The same ideas with viewing history on the improvements.

### Videos

Netflix uses S3 to store the initially uploaded videos and then transcode them to be pushed to remote CDNs. Object storage
could be a good choice to store media files, and it also has REST API support.

## How admin upload video to Netflix or users upload video to Youtube

## How to upload large video files to thousands of CDN servers

## How to build personalized home page per user profile

## How to auto switch between different resolutions while streaming

<https://en.wikipedia.org/wiki/Adaptive_bitrate_streaming>

## How to remember last played time

<https://netflixtechblog.com/netflixs-viewing-data-how-we-know-where-you-are-in-house-of-cards-608dd61077da>
<https://netflixtechblog.com/to-be-continued-helping-you-find-shows-to-continue-watching-on-7c0d8ee4dab6>

## How user search the video

## How video is recommended

Spark, collaborative filtering and content based filtering

## References

- [Recommendation system for Netflix](https://beta.vu.nl/nl/Images/werkstuk-fernandez_tcm235-874624.pdf)
- [Youtube: Netflix system design](https://www.youtube.com/watch?v=psQzyFfsUGU&ab_channel=TechDummiesNarendraL)
- <https://about.netflix.com/en/news/how-netflix-works-with-isps-around-the-globe-to-deliver-a-great-viewing-experience>
- <https://netflixtechblog.com/scaling-time-series-data-storage-part-i-ec2b6d44ba39>
- <https://netflixtechblog.com/scaling-time-series-data-storage-part-ii-d67939655586>
- <https://netflixtechblog.com/to-be-continued-helping-you-find-shows-to-continue-watching-on-7c0d8ee4dab6>
- <https://netflixtechblog.com/netflixs-viewing-data-how-we-know-where-you-are-in-house-of-cards-608dd61077da>
- <https://en.wikipedia.org/wiki/Adaptive_bitrate_streaming>