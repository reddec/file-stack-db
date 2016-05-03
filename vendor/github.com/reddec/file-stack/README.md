# About

This is extended stack implementation on Go-lang with the file as storage

For API see http://godoc.org/github.com/reddec/file-stack

# Introduction

## Use-case

When a lot of messages (events from IoT for example) with simple structure
have to be saved on disk for later usage with instant (or near that) access to
last message (I mean something like `get last state`). Processing of
historical events may be non-realtime operation.

## Requirements

Expected load:

* Rate: about 40'000+ messages/seconds
* Message size:
 * 90% - less then 1KB (means status updates or notifications)
 * 10% - about 5-50MB (means camera images or batch updates)
* Message contains:
 * Small (can easily fits into one memory block - about 4KB) header
 * Big body

Expected environment:

* Fast, big and reliable file storage - like SSD, enterprise HDD or NAS
* Low-cost multi-cores computing unit - like virtual machine in Digital Ocean, Supermicro or Raspberry Pi 2

Expected usage:

* State query (aka `get last message`): about 5'000+ requests/seconds
* Full historical dump: about 1 times/week
* Full historical headers scan: about 0.5 time/second
* Almost messages have not to be lost

Formal minimal operations descriptions:

* Put full message
* Get last message
* Iterate over headers

## Possible decisions

There are several implementations: SQL databases, no-SQL, KV databases,
in-memory storage and others.

Tested:

### PostgreSQL

* Tested version: *9.4*

#### Pros

* Stability
* Clustering
* Low resource usage

#### Cons

* Not enough speed (7000-8000 inserts/second on target hardware)

### Cassandra DB

#### Pros

* Fastest solution in theory
* Stability

#### Cons

* Awful memory usage - about 10 GB in idle state

### LevelDB

#### Pros

* Stable
* Fast

#### Cons

* Not enough fast (about 20000 inserts/second)
* Too simple structure: can't iterate over keys without loading full item

# Current implementation

12-cores I7 5-gen, SSD 840 PRO

* Push: 200'000 messages/second
* Pop: 200'000 messages/second
* Last message: 1'000'000 requests/second
