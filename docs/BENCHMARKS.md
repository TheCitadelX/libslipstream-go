# Benchmarks

Run transport benchmarks with:

```sh
go test -bench 'BenchmarkFragmenter' -benchmem ./...
```

The current benchmark group measures DNS transport overhead around a 1200-byte
QUIC packet:

- `BenchmarkFragmenterSplit`: packet to DNS-safe fragments.
- `BenchmarkFragmenterSplitEncodeDNSQueries`: fragments plus DNS TXT query
  encoding.
- `BenchmarkFragmenterSplitReassemble`: fragments back to a packet.

Use these numbers to compare allocator pressure and throughput when tuning
fragment size, resolver strategy, or DNS encoding.
