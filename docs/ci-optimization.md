# CI Pipeline Optimization

## Overview

This document describes the optimizations applied to the GitHub Actions CI pipeline to significantly reduce build times and improve developer experience.

## Implemented Optimizations

### Phase 1: Parallel Job Execution

**What Changed:**
- Backend and frontend jobs now run in parallel instead of sequentially
- Added a `status-check` job that depends on both jobs for branch protection

**Benefits:**
- Up to 50% reduction in total CI time when both jobs need to run
- Single merge gate for branch protection rules via the status-check job

**Implementation Details:**
- Jobs no longer have dependencies on each other (removed `needs` between backend/frontend)
- Status check job handles skipped jobs correctly (when path filters skip a job)

### Phase 2: Go Backend Caching Optimization

**What Changed:**
- Migrated to `actions/setup-go@v5`'s built-in cache parameter (more efficient)
- Added caching for Go build artifacts (`~/.cache/go-build`)
- Added caching for golangci-lint cache (`~/.cache/golangci-lint`)

**Benefits:**
- Faster dependency resolution and compilation
- Reduced linting time on cache hits
- Expected 20-40% reduction in backend job time on cache hits

**Cache Keys:**
- Go modules: Based on `backend/go.sum` hash
- Build cache: Based on all Go files in `backend/**/*.go`
- Lint cache: Based on `backend/go.sum` hash

**Expected Cache Hit Rates:**
- Go modules: ~90% (only changes when dependencies change)
- Build cache: ~70% (changes when any Go code changes)
- Lint cache: ~90% (only changes when dependencies change)

### Phase 3: Frontend Build Performance

**What Changed:**
- Ensured `pnpm install --frozen-lockfile` for faster, more reliable installs
- Node.js caching via `setup-node@v4` with pnpm cache enabled

**Benefits:**
- Faster dependency installation
- More reliable builds (frozen lockfile prevents unexpected version changes)
- Expected 15-25% reduction in frontend job time on cache hits

**Cache Keys:**
- pnpm store: Based on `frontend/pnpm-lock.yaml` hash

**Expected Cache Hit Rates:**
- pnpm cache: ~85% (only changes when dependencies change)

### Phase 4: Smart Job Triggering (Path Filters)

**What Changed:**
- Added `changes` job using `dorny/paths-filter@v3` action
- Backend job only runs when backend code, Go modules, or workflow changes
- Frontend job only runs when frontend code, package files, or workflow changes
- Both jobs always run on main branch pushes for complete validation

**Benefits:**
- Reduced CI resource usage (don't run frontend tests when only backend changes)
- Faster feedback for focused changes
- Expected 30-50% reduction in CI minutes usage

**Path Filters:**
```yaml
backend:
  - 'backend/**'
  - 'go.mod'
  - 'go.sum'
  - '.github/workflows/ci.yml'

frontend:
  - 'frontend/**'
  - 'package.json'
  - 'pnpm-lock.yaml'
  - '.github/workflows/ci.yml'
```

### Phase 5: Concurrency Controls

**What Changed:**
- Added concurrency group configuration at workflow level
- Cancels in-progress runs when new commits are pushed to the same PR
- Main branch runs always complete (never cancelled)

**Benefits:**
- Saves CI resources by canceling redundant runs
- Faster feedback (don't wait for old runs to complete)
- Reduced queue times during active development

**Configuration:**
```yaml
concurrency:
  group: ${{ github.workflow }}-${{ github.event.pull_request.number || github.ref }}
  cancel-in-progress: ${{ github.ref != 'refs/heads/main' }}
```

### Phase 6: Test Execution Optimization

**What Changed:**
- Go tests now run with `-p=4` flag for parallel package execution
- Tests run up to 4 packages simultaneously

**Benefits:**
- Faster test execution on multi-core runners
- Expected 15-30% reduction in test execution time

**Implementation:**
- Modified `backend/Makefile.testing` to add `-p=4` to test command

### Phase 7: Documentation and Metrics

**What Changed:**
- Added comprehensive inline comments in workflow file
- Created this documentation file
- Documented cache strategies and expected improvements

## Expected Overall Improvements

### Time Savings
- **Baseline scenario** (both jobs run, cache hit): 40-60% faster
- **Backend-only changes**: 50-70% faster (frontend job skipped)
- **Frontend-only changes**: 50-70% faster (backend job skipped)
- **Cache miss**: 20-30% faster (parallelization + concurrency benefits)

### Resource Efficiency
- **30-50% reduction** in GitHub Actions minutes usage due to:
  - Path filtering (skip unnecessary jobs)
  - Concurrency controls (cancel redundant runs)
  - Smart caching (faster execution)

### Developer Experience
- **Faster PR feedback**: Results in 2-5 minutes instead of 5-10 minutes
- **Cancelled redundant runs**: No need to wait for old runs when pushing updates
- **Clear status checks**: Single status-check job for branch protection

### Cost Optimization
- Lower CI costs through efficient resource usage
- Reduced GitHub Actions minutes consumption
- Better utilization of runner capacity

## Monitoring and Metrics

### How to Check Cache Performance

1. **Check cache hit rates** in workflow run logs:
   - Look for "Cache restored from key:" (cache hit)
   - Look for "Cache not found" (cache miss)

2. **Compare run times**:
   - Before optimization: ~8-12 minutes (both jobs sequential)
   - After optimization (cache hit): ~3-5 minutes (parallel with cache)
   - After optimization (cache miss): ~5-7 minutes (parallel, no cache)

3. **Monitor path filter effectiveness**:
   - Check how often jobs are skipped
   - Look at the "Detect Changes" job output

### Troubleshooting

#### Cache not working?
- Check if cache keys are correctly computed
- Verify file paths in cache configuration
- Look for "Post Cache" steps in logs for cache save errors

#### Jobs not skipping when expected?
- Verify path filters in the `changes` job
- Check if `.github/workflows/ci.yml` was modified (triggers all jobs)
- Ensure you're on a PR (path filters don't apply to main branch)

#### Tests failing with parallel execution?
- Some tests might not be thread-safe
- Reduce `-p` value in `Makefile.testing` (try `-p=2`)
- Add `-parallel=1` for specific test packages if needed

## Future Optimization Opportunities

### Short-term (Next 3-6 months)
1. **Matrix builds**: Split tests across multiple runners for larger test suites
2. **Docker layer caching**: If Docker builds are added
3. **Dependency caching**: Cache external tools (protobuf, etc.)

### Long-term (6-12 months)
1. **Self-hosted runners**: For faster execution and lower costs at scale
2. **Test result caching**: Skip tests for unchanged code
3. **Remote caching**: Distributed cache for monorepo setups
4. **Build artifacts caching**: Cache compiled binaries across workflows

## References

- [GitHub Actions Caching](https://docs.github.com/en/actions/using-workflows/caching-dependencies-to-speed-up-workflows)
- [Go Test Parallelization](https://golang.org/cmd/go/#hdr-Testing_flags)
- [Path Filtering Action](https://github.com/dorny/paths-filter)
- [Concurrency in Workflows](https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions#concurrency)
