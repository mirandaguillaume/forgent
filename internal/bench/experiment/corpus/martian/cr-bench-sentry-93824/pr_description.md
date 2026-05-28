## Span Buffer Multiprocess Enhancement with Health Monitoring

Refs STREAM-266

Compression helped a lot bring the time on the main thread down. We now think we can scale down the consumers. However, we also need to ensure the flusher can keep up. So let's make the flusher spawn a process per redis shard. 

We think this is easier to do than making the process_spans/insert_spans multiprocessed. Over time we can offload mroe things like statistics/metrics to the flusher, and keep the main thread minimal.