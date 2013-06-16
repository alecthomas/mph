# A Go library implementing Minimal Perfect Hashing

This library provides [Minimal Perfect Hashing](http://en.wikipedia.org/wiki/Perfect_hash_function) (MPH) using the [Compress, Hash and Displace](http://cmph.sourceforge.net/papers/esa09.pdf) (CHD) algorithm.

## What is this useful for?

Primarily, extremely efficient access to static datasets, such as geographical data, NLP data sets, etc.

## How would it be used?

Typically, the hash table would be used as a fast index into a (much) larger data set, with values in the hash table being file offsets or similar.

The hash tables can be serialized.

## Example code

Building and serializing an MPH hash table (error checking omitted for clarity):

```go
b := mph.NewCHDBuilder()
for k, v := range data {
    b.Add([]byte(k), []byte(v))
}
h, _ := b.Build()
b, _ := h.Marshal()
ioutil.WriteFile("data.idx", b, 0666)
```

Deserializing the hash table and performing lookups:

```go
b, _ := ioutil.ReadFile("data.idx")
h, _ := h.Unmarshal(b)

v := h.Get([]byte("some key"))
if v == nil {
    // Key not found
}
```
