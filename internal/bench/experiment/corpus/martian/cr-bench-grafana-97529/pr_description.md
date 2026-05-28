## Unified Storage Performance Optimizations

Changes:
- Initializes unified storage when the `ResourceServer` is created, instead of doing it inside the first gRPC call it receives. This was causing context cancelled errors since the index takes too long to build within the context of the gRPC call.
- Fixes trace propagation by passing span contexts down.
- Improves index build speed using finer grain locking when writing the index cache. We were locking the whole `BuildIndex` function, which was slowing things down when building many namespaces with high concurrency.

Notes for reviewer:
- Does anything else depend on initializing US lazily? What could this break?