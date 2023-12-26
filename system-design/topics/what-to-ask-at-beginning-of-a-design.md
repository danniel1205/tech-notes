# What questions to ask at the beginning of a design

## Questions around functional requirements

* What are the main use cases as of MVP
* Questions around load(Data load, Network load, etc). E.g.:
  * What is the total number of users?
  * What is the average amount of concurrent online users?
  * What is the expected peak traffic?
  * Read heavy or Write heavy, and what is the ratio ?
  * What are the limitations on storage ?

## Questions around non-functional requirements

* Should the distributed counter be strongly consistent ?
* How much latency we can tolerate ?
* Can I assume we want to the service to be high available ?