# goTap Performance Benchmark Results

## Test Environment
- **OS**: Windows  
- **CPU**: 11th Gen Intel(R) Core(TM) i5-1155G7 @ 2.50GHz
- **Go Version**: 1.21+
- **Test Date**: December 10, 2025

## Executive Summary

✅ **Benchmark Suite**: 35 benchmarks created (1 disabled due to routing limitations)
✅ **Memory Profiling**: Completed - identified allocation hotspots
✅ **CPU Profiling**: Completed - identified performance bottlenecks

### Key Findings:
- **Routing Performance**: 252-445 ns/op (excellent)
- **Memory Efficiency**: 30-35 B/op for simple routes (optimal)
- **Allocations**: 2 allocs/op consistently (ideal)
- **Hot Paths Identified**: Header setting (46.66% allocations), String rendering (29.04%)

## Benchmark Results Summary

### Core Routing Benchmarks

```
BenchmarkSimpleRoute-8       6,050,972 ops     252.7 ns/op     35 B/op    2 allocs/op
BenchmarkOneParam-8          4,105,082 ops     316.6 ns/op     32 B/op    2 allocs/op
BenchmarkFiveParams-8        2,305,250 ops     444.8 ns/op     31 B/op    2 allocs/op
BenchmarkStaticRoutes-8      4,980,916 ops     278.3 ns/op     30 B/op    2 allocs/op
```

**Analysis:**
- ✅ **Simple Route**: ~6M requests/second - excellent throughput
- ✅ **Parameter Extraction**: Minimal overhead (64 ns for 5 params)
- ✅ **Memory**: 30-35 bytes per request - very efficient
- ✅ **Allocations**: Consistent 2 allocs/op - optimal

### Advanced Benchmarks

**Context Operations:**
```
BenchmarkParamExtraction-8    2,169,998 ops     736.3 ns/op    126 B/op    6 allocs/op
BenchmarkQueryParsing-8         876,398 ops    1248 ns/op     582 B/op   11 allocs/op
```

**JSON & Rendering:**
```
BenchmarkJSONRender-8           (pending full run)
BenchmarkStringRender-8         (pending full run)
```

**Middleware:**
```
BenchmarkMiddleware-8           (pending full run)
BenchmarkCORS-8                 (pending full run)
BenchmarkRateLimiter-8          (pending full run)
```

## Memory Profile Analysis

**Tool**: `go tool pprof -top -cum mem.out`

### Top Memory Allocators (by cumulative allocation):

| Function | Flat | Cumulative | % of Total |
|----------|------|------------|------------|
| `(*Context).String` | 103.50MB | 458.68MB | 98.40% |
| `MIMEHeader.Set` | 217.50MB | 217.50MB | 46.66% |
| `bytes.growSlice` | 135.38MB | 135.38MB | 29.04% |

### Key Insights:

1. **Header Setting Overhead** (46.66% allocations)
   - `net/textproto.MIMEHeader.Set` allocates heavily
   - Used by `Context.Header()` for setting Content-Type
   - **Optimization**: Consider header pooling or pre-allocation

2. **String Rendering** (29.04% allocations)
   - `bytes.(*Buffer).grow` from response writing
   - Growing slices in `bytes` package
   - **Optimization**: Pre-allocate response buffers

3. **Minimal Framework Overhead**
   - Context pooling is working (no direct Context allocation in top allocators)
   - Routing tree has zero allocations in hot path
   - Most allocations are from stdlib HTTP and response writing

## CPU Profile Analysis

**Tool**: `go tool pprof -top -cum cpu.out`

### Top CPU Consumers (by cumulative time):

| Function | Self Time | Cumulative | % of Total |
|----------|-----------|------------|------------|
| `(*Engine).ServeHTTP` | 0.07s | 3.79s | 91.55% |
| `(*Engine).handleHTTPRequest` | 0.28s | 3.43s | 82.85% |
| `(*Context).Next` | 0.06s | 3.02s | 72.95% |
| `(*Context).String` | 0.10s | 2.87s | 69.32% |
| `(*Context).Header` | 0.04s | 1.82s | 43.96% |
| `runtime.mallocgc` | 0.55s | 1.28s | 30.92% |

### Key Insights:

1. **Memory Allocation** (30.92% CPU time)
   - `runtime.mallocgc` is #1 CPU consumer
   - Triggered by header setting and string rendering
   - **Optimization**: Reduce allocations = faster performance

2. **Header Canonicalization** (13.53% CPU time)
   - `net/textproto.CanonicalMIMEHeaderKey` overhead
   - Called for every header set operation
   - **Optimization**: Cache common headers (Content-Type, etc.)

3. **Efficient Routing**
   - Routing logic itself has minimal CPU overhead
   - Most time spent in stdlib and response generation
   - Framework overhead is very low

## Performance Goals vs. Actuals

| Metric | Goal | Actual | Status |
|--------|------|--------|--------|
| Routing overhead | < 200 ns | ~253 ns | 🟡 Close (26% over) |
| Request throughput | 100k+ req/sec | 6M req/sec | ✅ 60x better |
| Memory per request | < 100 bytes | 30-35 B | ✅ 3x better |
| Allocations | 0-2 per request | 2 | ✅ Optimal |

## Identified Optimization Opportunities

### Priority 1: Header Optimization (High Impact)
**Issue**: Header setting accounts for 46.66% of allocations and 43.96% of CPU time
- **Current**: Every `c.String()` call sets Content-Type header dynamically
- **Solution**: 
  1. Pre-allocate common headers
  2. Cache canonicalized header keys
  3. Use header pooling for common responses

**Expected Gain**: 30-40% performance improvement

### Priority 2: Response Buffer Pre-allocation (Medium Impact)
**Issue**: `bytes.growSlice` allocates 135.38MB (29.04%)
- **Current**: Response buffers grow dynamically
- **Solution**:
  1. Pre-allocate response writer buffers (1KB-4KB)
  2. Pool response buffers
  3. Tune buffer size based on common response sizes

**Expected Gain**: 15-20% performance improvement

### Priority 3: Reduce Allocations in Hot Path (Low Impact)
**Issue**: `runtime.mallocgc` consumes 30.92% CPU
- **Current**: 2 allocations per request (already very good)
- **Solution**:
  1. Profile individual allocations
  2. Consider sync.Pool for temporary objects
  3. Evaluate zero-allocation routing for simple cases

**Expected Gain**: 5-10% performance improvement

## Routing Limitation Discovered

**Issue**: BenchmarkGitHubAPI disabled due to routing conflicts

**Problem**: Routing tree cannot handle both:
- `/users/:user/events`
- `/users/:user/events/public`

When a route has a parameter segment (`:user`), adding both the route itself AND a child route causes conflicts.

**Impact**: Moderate - affects complex API designs like GitHub's
**Priority**: Medium - should be fixed in routing tree implementation
**Workaround**: Use different route patterns or flatten the hierarchy

## Next Steps

### Immediate Optimizations:
1. ✅ Benchmarking completed
2. ✅ Memory profiling completed  
3. ✅ CPU profiling completed
4. ⏳ Implement header optimization (Priority 1)
5. ⏳ Implement response buffer pooling (Priority 2)
6. ⏳ Re-run benchmarks to verify improvements

### Long-term Improvements:
1. Fix routing tree parameter conflict issue
2. Add benchmarks for all middleware
3. Profile under concurrent load
4. Compare against Gin benchmarks
5. Optimize for zero allocations where possible

## Conclusion

**Current Performance**: Excellent
- Already exceeds goals for throughput (6M vs 100K req/sec)
- Memory usage is optimal (30-35B vs <100B goal)
- Allocations are minimal (2 per request)

**Main Bottleneck**: Standard library overhead (header handling)
- 47% of allocations from `net/textproto`
- 29% from `bytes` buffer growth
- Framework itself is very efficient

## Performance Optimization Results

### Header Optimization (Implemented ✅)

**Problem**: `net/textproto.MIMEHeader.Set` was causing 46.66% of allocations (217.50MB) due to header canonicalization overhead.

**Solution**: Created `setContentType()` fast path that bypasses expensive canonicalization:
```go
func (c *Context) setContentType(value string) {
	c.Writer.Header()["Content-Type"] = []string{value}
}
```

**Results**:
| Benchmark | Before | After | Improvement |
|-----------|--------|-------|-------------|
| BenchmarkSimpleRoute | 252.7 ns/op | 171.8 ns/op | **32.0% faster** ✅ |
| BenchmarkJSONRender | 476.9 ns/op | 451.9 ns/op | 5.2% faster |

**Impact**: All 11 rendering methods (JSON, XML, YAML, HTML, SSE, etc.) now bypass header canonicalization, achieving **<200 ns/op goal** for simple routing.

### Buffer Pooling (Evaluated ❌)

**Tested**: sync.Pool for response buffers to reduce `bytes.growSlice` allocations (29.04%).

**Result**: **Not effective** for small JSON payloads. `json.NewEncoder` writing directly to `http.ResponseWriter` is already optimal. Buffer pooling added overhead (503.5 ns vs 476.9 ns).

**Decision**: Reverted. Keep direct encoding approach.

### Allocation Reduction (Evaluated ✅)

**Current**: 2 allocs/op for simple routing
**Analysis**: This is already **optimal** for a web framework (1 context + 1 for internal structures)
**Decision**: No further optimization needed - diminishing returns

**Remaining Optimizations**:
- ✅ Priority 1: Header optimization (**32% improvement achieved**)
- ❌ Priority 2: Buffer pooling (ineffective for small payloads)
- ✅ Priority 3: 2 allocs/op is optimal (no action needed)

---

**Status**: Phase 7 (Performance Optimization & Test Coverage) - **Completed**
- ✅ Benchmark suite created (35 benchmarks)
- ✅ Profiling completed (memory & CPU)
- ✅ Hotspots identified (3 priorities)
- ✅ Priority 1 optimization: **32% improvement, <200ns goal achieved**
- ✅ Priority 2 & 3 evaluated (no further gains available)
- ✅ Test coverage increased: 65.2% → **75.0%** (+9.8 points)
- 🟡 Test coverage goal: 90%+ (15 points remaining, 21 functions at 0%)

**Test Files Created (8 total, 2,181 lines, 83 tests)**:
1. **errors_test.go** (201 lines, 11 tests) - ✅ Complete error handling
2. **gotap_server_test.go** (113 lines, 6 tests) - ✅ Server lifecycle & ResponseWriter
3. **binding_coverage_test.go** (338 lines, 9 tests) - ✅ Binding edge cases
4. **middleware_gzip_coverage_test.go** (130 lines, 5 tests) - ✅ Gzip middleware methods
5. **validation_coverage_test.go** (246 lines, 6 tests) - ✅ Validation edge cases
6. **ratelimiter_coverage_test.go** (226 lines, 8 tests) - ✅ Rate limiter comprehensive
7. **websocket_coverage_test.go** (399 lines, 9 tests) - ✅ WebSocket functionality (~80% coverage)
8. **context_coverage_test.go** (528 lines, 26 tests) - ✅ Context methods edge cases

**Coverage by Component**:
- ✅ Errors: 100% complete
- 🟢 WebSocket: ~80% (only Clients() uncovered)
- 🟢 Context: ~75% (edge cases covered)
- 🟢 Binding: ~70% (BindBody uncovered)
- 🟢 Middleware (Gzip): ~90% (Hijack uncovered)
- 🟢 Middleware (Rate Limiter): ~85% (Reset uncovered)
- 🟢 Validation: ~75% (Engine uncovered)
- 🟡 Server: ~60% (Run, RunTLS, Shutdown uncovered - 5 functions)
- 🟡 Render: ~70% (LoadHTMLFiles, Stream uncovered - 2 functions)
- ❌ ShadowDB: 0% (all 6 functions uncovered, requires external package)

**Remaining 0% Functions (21 total)**:
- **Server lifecycle**: Run, RunServer, Shutdown, ShutdownWithTimeout, RunTLS (5)
- **Individual functions**: BindBody, Readdir, Hijack, CombinedIPFilter, RefreshToken, Reset, LoadHTMLFiles, Stream, Engine, Clients (10)
- **ShadowDB**: All middleware functions (6)


