# E2E Test Structure: Visual Comparison

## Current Structure (Problem)

```
tests/e2e/
│
├── bitbucket_test.go           ⚠️  1850 lines (TOO LARGE!)
│   ├── TestBitBucketScan_HappyPath
│   ├── TestBitBucketScan_Owned_*
│   ├── TestBitBucketScan_Workspace_*
│   ├── TestBitBucketScan_Public_*
│   ├── TestBitBucketScan_Artifacts_* (5 tests)
│   ├── TestBitBucketScan_Pagination
│   ├── TestBitBucketScan_RateLimit
│   └── ... 26 tests total (mixed concerns)
│
├── devops_test.go              ⚠️  Inconsistent naming
│   └── TestAzureDevOpsScan_*
│
├── gitea_test.go               ⚠️  Split across 2 files
│   └── 17 tests
├── gitea_comprehensive_test.go ⚠️  Why separate?
│   └── 5 tests
│
├── github_test.go              ⚠️  Split across 2 files
│   └── 3 tests
├── github_comprehensive_test.go ⚠️  Why separate?
│   └── 6 tests
│
├── gitlab_test.go              ✅  Well organized (813 lines)
│   └── 17 tests
│
├── root_test.go                ✅  CLI tests
│   └── 13 tests
│
└── helpers/
    ├── e2e_helpers_test.go
    └── cli_integration.go
```

**Problems:**
1. ⚠️  1850-line BitBucket file (hard to navigate)
2. ⚠️  Inconsistent naming (devops vs azuredevops)
3. ⚠️  Unclear split (gitea_test.go + gitea_comprehensive_test.go)
4. ⚠️  No logical grouping (all tests mixed together)
5. ⚠️  Hard to find specific test types

---

## Proposed Structure (Solution)

### Option 1: Folder-per-Platform ⭐ RECOMMENDED

```
tests/e2e/
│
├── shared/                      ✨ Reusable test infrastructure
│   ├── helpers_test.go         │   - startMockServer()
│   ├── assertions_test.go      │   - runCLI()
│   └── fixtures.go             │   - Common mock data
│
├── bitbucket/                   ✅ BitBucket tests (organized)
│   ├── scan_basic_test.go      │   - Auth, flags (500 lines)
│   ├── scan_artifacts_test.go  │   - Artifacts, .env (600 lines)
│   ├── scan_advanced_test.go   │   - Pagination, rate-limit (500 lines)
│   └── errors_test.go          │   - Error handling (250 lines)
│
├── azuredevops/                 ✅ Azure DevOps (consistent naming)
│   ├── scan_basic_test.go
│   ├── scan_artifacts_test.go
│   └── errors_test.go
│
├── gitea/                       ✅ Gitea tests (merged & organized)
│   ├── scan_basic_test.go      │   - Basic scanning
│   ├── scan_advanced_test.go   │   - Advanced features
│   ├── enum_test.go            │   - Enumeration
│   └── errors_test.go
│
├── github/                      ✅ GitHub tests (merged & organized)
│   ├── scan_basic_test.go
│   ├── scan_artifacts_test.go
│   ├── scan_advanced_test.go
│   └── errors_test.go
│
├── gitlab/                      ✅ GitLab tests (split by feature)
│   ├── scan_basic_test.go
│   ├── enum_test.go
│   ├── utilities_test.go       │   - Variables, runners, etc.
│   ├── vuln_test.go
│   ├── register_test.go
│   └── errors_test.go
│
├── cli/                         ✅ CLI tests
│   ├── root_test.go
│   └── integration_test.go
│
└── testdata/                    ✅ Test data & docs
    ├── rules.yml
    ├── README.md
    └── QUICK_START.md
```

**Benefits:**
1. ✅ Clear platform separation
2. ✅ Smaller files (200-400 lines each)
3. ✅ Consistent naming
4. ✅ Logical grouping
5. ✅ Easy test discovery

---

## File Size Comparison

### Before
```
bitbucket_test.go:           1850 lines  ⚠️  TOO LARGE
github_comprehensive_test.go: 757 lines  ⚠️  Fragmented
gitlab_test.go:               813 lines  ⚠️  Could be better
gitea_test.go:                698 lines  ⚠️  Fragmented
```

### After (Option 1)
```
bitbucket/scan_basic_test.go:      ~500 lines  ✅ Manageable
bitbucket/scan_artifacts_test.go:  ~600 lines  ✅ Manageable
bitbucket/scan_advanced_test.go:   ~500 lines  ✅ Manageable
bitbucket/errors_test.go:          ~250 lines  ✅ Focused

github/scan_basic_test.go:         ~300 lines  ✅ Focused
github/scan_artifacts_test.go:     ~400 lines  ✅ Focused
github/scan_advanced_test.go:      ~200 lines  ✅ Focused
```

---

## Test Discovery Comparison

### Before (Current)
```bash
# Want to run BitBucket artifact tests?
# Must know exact test names in huge file
go test -run TestBitBucketScan_Artifacts ./tests/e2e

# Want to add new GitHub artifact test?
# Which file? github_test.go or github_comprehensive_test.go?
# Must read both files to decide
```

### After (Proposed)
```bash
# Run all BitBucket artifact tests
go test ./tests/e2e/bitbucket -run Artifacts

# Run ALL artifact tests across platforms
go test ./tests/e2e/.../scan_artifacts_test.go

# Add new GitHub artifact test?
# Obviously: tests/e2e/github/scan_artifacts_test.go
```

---

## Navigation Comparison

### Before: Finding a Test
```
1. Open tests/e2e/
2. See 10+ test files
3. Guess which file has the test
4. Open bitbucket_test.go (1850 lines)
5. Search for test name
6. Scroll through hundreds of lines
7. Finally find the test at line 978
```

### After: Finding a Test
```
1. Open tests/e2e/
2. Navigate to bitbucket/
3. See clearly labeled files:
   - scan_basic_test.go
   - scan_artifacts_test.go  ← Obviously here!
4. Open scan_artifacts_test.go (600 lines)
5. Test is near the top (clear organization)
```

---

## Test Organization

### Before (Bitbucket Example)
```go
// bitbucket_test.go (1850 lines, 26 tests - NO ORGANIZATION)

func TestBitBucketScan_HappyPath(t *testing.T)
func TestBitBucketScan_MissingCredentials(t *testing.T)
func TestBitBucketScan_Owned_HappyPath(t *testing.T)
func TestBitBucketScan_Owned_Unauthorized(t *testing.T)
func TestBitBucketScan_Workspace_HappyPath(t *testing.T)
func TestBitBucketScan_Artifacts_WithDotEnv(t *testing.T)
func TestBitBucketScan_Artifacts_NestedArchive(t *testing.T)
func TestBitBucketScan_Pagination(t *testing.T)
... all mixed together in one file
```

### After (Bitbucket Example)
```go
// bitbucket/scan_basic_test.go (500 lines)
func TestBitbucketScan_HappyPath(t *testing.T)
func TestBitbucketScan_Owned_HappyPath(t *testing.T)
func TestBitbucketScan_Workspace_HappyPath(t *testing.T)
func TestBitbucketScan_Public_HappyPath(t *testing.T)
func TestBitbucketScan_MaxPipelines(t *testing.T)
func TestBitbucketScan_Confidence(t *testing.T)
func TestBitbucketScan_Threads(t *testing.T)
func TestBitbucketScan_Verbose(t *testing.T)

// bitbucket/scan_artifacts_test.go (600 lines)
func TestBitbucketScan_Artifacts_WithDotEnv(t *testing.T)
func TestBitbucketScan_Artifacts_NestedArchive(t *testing.T)
func TestBitbucketScan_Artifacts_MultipleFiles(t *testing.T)
func TestBitbucketScan_DownloadArtifacts(t *testing.T)

// bitbucket/scan_advanced_test.go (500 lines)
func TestBitbucketScan_Pagination(t *testing.T)
func TestBitbucketScan_RateLimit(t *testing.T)
func TestBitbucketScan_ConfidenceFilter(t *testing.T)

// bitbucket/errors_test.go (250 lines)
func TestBitbucketScan_MissingCredentials(t *testing.T)
func TestBitbucketScan_Owned_Unauthorized(t *testing.T)
func TestBitbucketScan_InvalidCookie(t *testing.T)
```

---

## Running Tests Comparison

### Before
```bash
# Run all tests (forced to run everything)
cd tests/e2e && go test -v

# Run specific platform (must know exact test names)
go test -run TestBitBucket ./tests/e2e

# Run specific feature (hard to filter)
go test -run "Artifacts" ./tests/e2e  # Gets ALL platforms
```

### After
```bash
# Run all tests
cd tests/e2e && go test ./... -v

# Run specific platform
go test ./tests/e2e/bitbucket/... -v

# Run specific feature across platforms
go test ./tests/e2e/.../scan_artifacts_test.go -v

# Run specific feature for one platform
go test ./tests/e2e/bitbucket/scan_artifacts_test.go -v

# Run only basic tests across all platforms
go test ./tests/e2e/.../scan_basic_test.go -v
```

---

## Code Review Comparison

### Before: Adding BitBucket Artifact Test
```diff
# Single PR affecting bitbucket_test.go
  bitbucket_test.go | 150 lines changed
  
⚠️  Reviewer must:
   - Review 150 new lines
   - In context of 1850-line file
   - Scroll to find where change is
   - Understand if placement makes sense
   - Check for conflicts with other tests
```

### After: Adding BitBucket Artifact Test
```diff
# Single PR affecting bitbucket/scan_artifacts_test.go
  bitbucket/scan_artifacts_test.go | 80 lines changed
  
✅  Reviewer can:
   - Focus on 80 new lines
   - In context of 600-line focused file
   - Immediately see it's in the right place
   - Understand purpose from file name
   - Less likely to conflict
```

---

## Maintenance Comparison

### Before: Fixing BitBucket Bug
```
1. Open bitbucket_test.go
2. Search for relevant test (1850 lines)
3. Find test at line 1120
4. Fix test
5. Run all 26 BitBucket tests (slow)
6. Commit changes to massive file
7. Potential conflicts with other PRs
```

### After: Fixing BitBucket Bug
```
1. Navigate to bitbucket/scan_artifacts_test.go
2. Test is obviously in this 600-line file
3. Find test quickly (good organization)
4. Fix test
5. Run only artifact tests: go test ./tests/e2e/bitbucket/scan_artifacts_test.go
6. Commit changes to focused file
7. Less likely to conflict
```

---

## Summary: Why Folder-per-Platform?

| Aspect | Before | After | Improvement |
|--------|--------|-------|-------------|
| **File Size** | 1850 lines | 200-600 lines | ✅ 3-9x smaller |
| **Navigation** | Search 10 files | Navigate folders | ✅ Intuitive |
| **Test Discovery** | Guess & search | Obvious location | ✅ Clear |
| **Organization** | Mixed concerns | Separated by type | ✅ Logical |
| **Test Running** | Run all or filter | Run specific folders | ✅ Targeted |
| **Code Review** | Large diffs | Focused changes | ✅ Easier |
| **Maintenance** | Hard to find tests | Easy to locate | ✅ Faster |
| **Scalability** | Gets worse over time | Stays organized | ✅ Sustainable |

---

## Next Steps

1. **Review Proposal**: Team reviews `REORGANIZATION_PROPOSAL.md`
2. **Approve Approach**: Decide on Option 1 (folder-per-platform)
3. **Pilot Migration**: Start with BitBucket (most problematic)
4. **Validate**: Ensure all tests still pass
5. **Document**: Update README with new structure
6. **Roll Out**: Migrate other platforms incrementally
7. **Clean Up**: Remove old files once migration is complete

---

## Quick Reference

### Platform Folder Structure (Standard)
```
<platform>/
├── scan_basic_test.go      # Auth, flags, basic functionality
├── scan_artifacts_test.go  # Artifact scanning (if supported)
├── scan_advanced_test.go   # Pagination, rate-limits, filters
├── enum_test.go           # Enumeration (if supported)
├── utilities_test.go      # Platform-specific utilities
└── errors_test.go         # Error handling & edge cases
```

### Test Naming Convention
```go
// Pattern: Test<Platform><Command>_<Feature>
func TestBitbucketScan_HappyPath(t *testing.T)
func TestBitbucketScan_Artifacts_WithDotEnv(t *testing.T)
func TestGithubScan_Pagination(t *testing.T)
func TestGitlabEnum_Unauthenticated(t *testing.T)
```

### Running Tests
```bash
# All tests
go test ./tests/e2e/... -v

# One platform
go test ./tests/e2e/bitbucket/... -v

# One feature across platforms
go test ./tests/e2e/.../scan_artifacts_test.go -v

# One specific test file
go test ./tests/e2e/bitbucket/scan_basic_test.go -v
```
