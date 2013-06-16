# A Go library implementing Minimal Perfect Hashing

This library provides [Minimal Perfect Hashing](http://en.wikipedia.org/wiki/Perfect_hash_function) (MPH) using the [Compress, Hash and Displace](http://cmph.sourceforge.net/papers/esa09.pdf) (CHD) algorithm.

## What is this useful for?

Primarily, extremely efficient access to static datasets, such as geographical data, NLP data sets, etc.

## How would it be used?

Typically, the hash table would be used as a fast index into a (much) larger data set, with values in the hash table being file offsets or similar.

The hash tables can be serialized.
