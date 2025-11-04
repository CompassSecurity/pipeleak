# E2E Test Structure Reorganization Proposal

## Current State Analysis

### File Structure (Current)
```
tests/e2e/
├── bitbucket_test.go           (1850 lines, 26 tests)
├── devops_test.go              (391 lines, 7 tests)
├── gitea_test.go               (698 lines, 17 tests)
├── gitea_comprehensive_test.go (440 lines, 5 tests)
├── github_test.go              (141 lines, 3 tests)
├── github_comprehensive_test.go(757 lines, 6 tests)
├── gitlab_test.go              (813 lines, 17 tests)
├── root_test.go                (392 lines, 13 tests)
├── e2e_helpers_test.go         (491 lines, helpers)
├── cli_integration.go          (94 lines, helpers)
├── rules.yml                   (test data)
├── README.md
├── QUICK_START.md
└── IMPLEMENTATION_SUMMARY.md
```

### Issues with Current Structure

1. **Naming Inconsistency**
   - Mix of platform names: `bitbucket`, `devops` (not `azuredevops`), `gitea`, `github`, `gitlab`
   - Inconsistent naming: `gitea_test.go` + `gitea_comprehensive_test.go` vs single files for others
   - No clear distinction between basic and comprehensive tests in file names

2. **File Size Imbalance**
   - BitBucket: 1850 lines (too large, hard to navigate)
   - GitHub: Split across 2 files (141 + 757 lines) with unclear separation
   - Gitea: Split across 2 files (698 + 440 lines) with unclear separation
   - DevOps: Small (391 lines) but could grow

3. **Test Organization**
   - Tests are grouped by platform but not by functionality
   - No clear separation between:
     - Scan tests (logs, artifacts, flags)
     - Enum/discovery tests
     - Utility command tests (variables, runners, etc.)
     - Error handling tests

4. **Test Naming Patterns**
   - BitBucket uses `TestBitBucketScan_*` (camelCase "BitBucket")
   - DevOps uses `TestAzureDevOpsScan_*` (full name)
   - Gitea uses `TestGiteaScan_*` + `TestGiteaEnum`
   - GitHub uses `TestGitHubScan_*` (camelCase "GitHub")
   - GitLab uses `TestGitLabScan_*` + `TestGitLabEnum` + utilities

---

## Proposed Reorganization

### Option 1: Folder-per-Platform (Recommended)

Organize tests into platform-specific folders with consistent file naming:

```
tests/e2e/
├── shared/
│   ├── helpers_test.go         (test helpers, mock server setup)
│   ├── assertions_test.go      (common assertion helpers)
│   └── fixtures.go             (shared test data, mock responses)
│
├── bitbucket/
│   ├── scan_basic_test.go      (auth, flags, basic functionality)
│   ├── scan_artifacts_test.go  (artifact scanning, .env, nested archives)
│   ├── scan_advanced_test.go   (pagination, rate-limit, confidence)
│   └── errors_test.go          (error handling, edge cases)
│
├── azuredevops/
│   ├── scan_basic_test.go
│   ├── scan_artifacts_test.go
│   ├── scan_advanced_test.go
│   └── errors_test.go
│
├── gitea/
│   ├── scan_basic_test.go      (auth, flags, owned, org, repo)
│   ├── scan_advanced_test.go   (pagination, rate-limit, confidence)
│   ├── enum_test.go            (enumeration functionality)
│   └── errors_test.go
│
├── github/
│   ├── scan_basic_test.go      (auth, flags, owned, org)
│   ├── scan_artifacts_test.go  (logs, artifacts, nested)
│   ├── scan_advanced_test.go   (pagination, rate-limit, maxWorkflows)
│   └── errors_test.go
│
├── gitlab/
│   ├── scan_basic_test.go      (auth, flags, scan variations)
│   ├── enum_test.go
│   ├── utilities_test.go       (variables, runners, cicd, schedule, securefiles)
│   ├── vuln_test.go            (vulnerability scanning)
│   ├── register_test.go        (unauthenticated registration)
│   └── errors_test.go
│
├── cli/
│   ├── root_test.go            (root command, flags, version)
│   └── integration_test.go     (cross-platform CLI tests)
│
└── testdata/
    ├── rules.yml
    ├── README.md
    ├── QUICK_START.md
    └── IMPLEMENTATION_SUMMARY.md
```

**Pros:**
- Clear separation by platform
- Each platform folder is self-contained
- Easier to find tests for specific platforms
- Better file size distribution (200-400 lines per file)
- Can add platform-specific helpers in each folder

**Cons:**
- More directory structure to navigate
- Requires updating import paths
- Slightly more boilerplate

---

### Option 2: Single File per Platform with Clear Sections

Keep single file per platform but with clear internal organization:

```
tests/e2e/
├── helpers_test.go              (shared helpers)
├── bitbucket_test.go            (all BitBucket tests, organized by sections)
├── azuredevops_test.go          (renamed from devops_test.go)
├── gitea_test.go                (merge gitea_test.go + gitea_comprehensive_test.go)
├── github_test.go               (merge github_test.go + github_comprehensive_test.go)
├── gitlab_test.go               (keep as-is, well organized)
├── root_test.go                 (CLI tests)
└── testdata/
    └── rules.yml
```

With clear section markers in each file:
```go
// ========================================
// Basic Scan Tests
// ========================================

// ========================================
// Artifact Scan Tests
// ========================================

// ========================================
// Advanced Features (Pagination, Rate-Limit, etc.)
// ========================================

// ========================================
// Error Handling & Edge Cases
// ========================================
```

**Pros:**
- Minimal structural changes
- Easy to navigate if files are well-organized internally
- No import path changes needed
- Simple to understand for new contributors

**Cons:**
- Large files remain large (bitbucket would still be 1850 lines)
- Harder to isolate and run specific test categories
- Merge conflicts more likely with large files

---

### Option 3: Hybrid Approach (Test Type + Platform)

Organize by test type first, then platform:

```
tests/e2e/
├── shared/
│   └── helpers_test.go
│
├── scan/
│   ├── bitbucket_basic_test.go
│   ├── bitbucket_artifacts_test.go
│   ├── bitbucket_advanced_test.go
│   ├── azuredevops_test.go
│   ├── gitea_test.go
│   ├── github_basic_test.go
│   ├── github_artifacts_test.go
│   ├── github_advanced_test.go
│   └── gitlab_test.go
│
├── enum/
│   ├── gitea_test.go
│   └── gitlab_test.go
│
├── utilities/
│   ├── gitlab_variables_test.go
│   ├── gitlab_runners_test.go
│   ├── gitlab_cicd_test.go
│   ├── gitlab_schedule_test.go
│   └── gitlab_securefiles_test.go
│
├── vuln/
│   └── gitlab_test.go
│
├── register/
│   └── gitlab_test.go
│
├── cli/
│   └── root_test.go
│
└── testdata/
    └── rules.yml
```

**Pros:**
- Groups related functionality across platforms
- Easy to compare similar tests across platforms
- Clear categorization of test types

**Cons:**
- Less intuitive for platform-specific development
- Harder to find all tests for a single platform
- May be confusing when platforms have different features

---

## Recommended Approach: **Option 1 (Folder-per-Platform)**

### Reasoning

1. **Platform-Centric Development**: Most contributions focus on a single platform
2. **Clear Boundaries**: Each platform has different features and APIs
3. **Scalability**: Easy to add new platforms without affecting existing tests
4. **Maintainability**: Smaller files (200-400 lines) are easier to review and modify
5. **Test Discovery**: Clear directory structure makes it obvious where to add new tests

### Migration Path

#### Phase 1: Create New Structure (Non-Breaking)
1. Create new folder structure
2. Copy tests into new locations
3. Update imports
4. Ensure all tests pass in new structure

#### Phase 2: Update Documentation
1. Update README with new structure
2. Add CONTRIBUTING guide for test organization
3. Document naming conventions

#### Phase 3: Deprecate Old Files
1. Mark old files as deprecated
2. Add redirects/notices in old files
3. Eventually remove old files

### Naming Conventions (Standardized)

#### Platform Names
- `bitbucket/` (lowercase, one word)
- `azuredevops/` (lowercase, no separator)
- `gitea/` (lowercase)
- `github/` (lowercase)
- `gitlab/` (lowercase)

#### Test File Names
- `scan_basic_test.go` - Authentication, flags, basic scanning
- `scan_artifacts_test.go` - Artifact scanning (logs, .env, nested)
- `scan_advanced_test.go` - Pagination, rate-limits, filters
- `enum_test.go` - Enumeration/discovery features
- `utilities_test.go` - Platform-specific utilities (variables, runners, etc.)
- `errors_test.go` - Error handling and edge cases

#### Test Function Names
Pattern: `Test<Platform><Command>_<Feature>`

Examples:
- `TestBitbucketScan_HappyPath`
- `TestBitbucketScan_Artifacts_WithDotEnv`
- `TestGitlabEnum_Unauthenticated`
- `TestGithubScan_RateLimit`

#### Package Names
All test files in same package: `package e2e_test` (external test package)

---

## Implementation Checklist

### For Option 1 (Folder-per-Platform)

- [ ] Create folder structure
  - [ ] `shared/`
  - [ ] `bitbucket/`
  - [ ] `azuredevops/`
  - [ ] `gitea/`
  - [ ] `github/`
  - [ ] `gitlab/`
  - [ ] `cli/`
  - [ ] `testdata/`

- [ ] Move and split existing tests
  - [ ] BitBucket: Split 1850 lines into 4 files
    - [ ] `scan_basic_test.go` (~500 lines)
    - [ ] `scan_artifacts_test.go` (~600 lines)
    - [ ] `scan_advanced_test.go` (~500 lines)
    - [ ] `errors_test.go` (~250 lines)
  
  - [ ] GitHub: Merge and organize
    - [ ] Merge `github_test.go` + `github_comprehensive_test.go`
    - [ ] Split into basic/artifacts/advanced
  
  - [ ] Gitea: Merge and organize
    - [ ] Merge `gitea_test.go` + `gitea_comprehensive_test.go`
    - [ ] Split into basic/advanced/enum/errors
  
  - [ ] Azure DevOps: Organize
    - [ ] Split `devops_test.go` into logical files
  
  - [ ] GitLab: Organize
    - [ ] Split `gitlab_test.go` by functionality
  
  - [ ] CLI: Organize
    - [ ] Move `root_test.go` to `cli/`

- [ ] Extract shared helpers
  - [ ] Move `e2e_helpers_test.go` → `shared/helpers_test.go`
  - [ ] Create `shared/assertions_test.go` for common assertions
  - [ ] Create `shared/fixtures.go` for shared test data

- [ ] Update imports and references
  - [ ] Update all import paths
  - [ ] Ensure package names are consistent
  - [ ] Update test discovery patterns

- [ ] Verify all tests pass
  - [ ] Run `go test ./tests/e2e/...`
  - [ ] Check coverage hasn't decreased
  - [ ] Verify no tests were lost

- [ ] Update documentation
  - [ ] Update README with new structure
  - [ ] Add navigation guide
  - [ ] Document conventions

- [ ] Clean up
  - [ ] Remove old files
  - [ ] Update CI/CD if needed
  - [ ] Add CODEOWNERS for test folders

---

## Benefits Summary

### Maintainability
- **Before**: 1850-line BitBucket file, hard to navigate
- **After**: 4 focused files of 250-600 lines each

### Discoverability
- **Before**: "Where do I add a GitHub artifact test?"
- **After**: Clearly in `tests/e2e/github/scan_artifacts_test.go`

### Testability
- **Before**: Must run all platform tests together
- **After**: Can run specific test categories: `go test ./tests/e2e/github/...`

### Scalability
- **Before**: Adding new platform requires careful file placement
- **After**: Create new folder with standard structure

### Code Review
- **Before**: Large diffs in big files
- **After**: Focused changes in small, categorized files

---

## Examples

### Before (Current)
```bash
# Running BitBucket tests (mixed concerns)
go test -run TestBitBucket ./tests/e2e

# Finding artifact test for BitBucket
# Must search through 1850 lines
grep -n "Artifacts" bitbucket_test.go
```

### After (Proposed)
```bash
# Running only BitBucket artifact tests
go test ./tests/e2e/bitbucket -run Artifacts

# Finding artifact test for BitBucket
# Clearly in bitbucket/scan_artifacts_test.go
```

---

## Alternative: Keep Current Structure

If reorganization is deemed too disruptive, minimum improvements:

1. **Rename Files for Consistency**
   - `devops_test.go` → `azuredevops_test.go`
   - Merge split files: `gitea_test.go` ← `gitea_comprehensive_test.go`
   - Merge split files: `github_test.go` ← `github_comprehensive_test.go`

2. **Add Section Markers**
   ```go
   // ========================================
   // BASIC SCAN TESTS
   // ========================================
   
   // ========================================
   // ARTIFACT SCAN TESTS
   // ========================================
   ```

3. **Split Large Files (BitBucket)**
   - Keep in same directory but split by concern
   - `bitbucket_scan_test.go`
   - `bitbucket_artifacts_test.go`
   - `bitbucket_advanced_test.go`

---

## Recommendation

Implement **Option 1: Folder-per-Platform** for the best long-term maintainability, discoverability, and scalability. The migration can be done gradually and won't break existing functionality.

Start with the most problematic platform (BitBucket with 1850 lines) as a proof of concept, then migrate others incrementally.
