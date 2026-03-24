---
name: performance-bottleneck-detection
description: Performance bottleneck identification through code pattern analysis-based skill consuming source_code to produce performance_report
---

# Performance Bottleneck Detection

## Guardrails
- timeout: 180s
- max_query_analysis_time: 60s
- performance_threshold_ms: 100

## Context
Consumes: source_code
Produces: performance_report
Memory: short-term

## Strategy
Approach: Performance bottleneck identification through code pattern analysis
Tools: read_file, grep, search, bash

### Steps
1. Analyze database queries for N+1 problems
2. Review memory usage patterns
3. Check caching implementations
4. Assess async/concurrent patterns
5. Evaluate resource management
6. Review load handling strategies

## Security
- Filesystem: read-only
- Network: none
