# Changelog

## Latest Updates

### New Analysis Sections Added

#### 1. Flux Events Analysis (Section 6)
- **24-hour and 48-hour event tracking**
- Filters for Flux-related components (HelmRelease, Kustomization, GitRepository, etc.)
- Separate tracking for warnings and errors
- Top 20 most frequent events with counts
- Actionable recommendations for Flux issues

**Key Features:**
- Detects reconciliation failures
- Tracks GitRepository sync issues
- Identifies Helm chart problems
- Monitors Kustomization errors

#### 2. Non-Flux Warning Events (Section 7)
- **24-hour and 48-hour warning event tracking**
- Filters out Flux-specific events to show general cluster issues
- Top 25 most frequent warnings
- Pattern detection for common problems

**Key Features:**
- Identifies pod scheduling issues
- Tracks resource allocation warnings
- Monitors volume mounting problems
- Detects container runtime warnings

#### 3. Velero Backup Analysis (Section 8)
- **24-hour and 48-hour backup tracking**
- Backup status monitoring (Completed/Failed/PartiallyFailed)
- Duration calculation and analysis
- Error and warning counts

**Key Features:**
- Tracks backup success rates
- Identifies slow backups
- Monitors backup failures
- Provides retention recommendations
- Tracks warnings that may indicate future issues

### Technical Implementation

#### New Data Structures
```go
type EventInfo struct {
    Type           string    // Warning, Normal, Error
    Reason         string
    Message        string
    Namespace      string
    InvolvedObject string
    Count          int32
    FirstTime      time.Time
    LastTime       time.Time
}

type FluxEventAnalysis struct {
    Events24h        []EventInfo
    Events48h        []EventInfo
    Warnings24h      int
    Warnings48h      int
    Errors24h        int
    Errors48h        int
}

type NonFluxEventAnalysis struct {
    Events24h   []EventInfo
    Events48h   []EventInfo
}

type VeleroBackup struct {
    Name           string
    Namespace      string
    Status         string
    StartTime      time.Time
    CompletionTime time.Time
    Duration       time.Duration
    Errors         int
    Warnings       int
}

type VeleroBackupAnalysis struct {
    Backups24h []VeleroBackup
    Backups48h []VeleroBackup
}
```

#### New Dependencies
- Added `k8s.io/client-go/dynamic` for CRD access (Velero backups)
- Added `k8s.io/apimachinery/pkg/apis/meta/v1/unstructured` for dynamic object parsing

#### Report Section Updates
- **Section 5**: Pod Restart Analysis (previously added)
- **Section 6**: Flux Events Analysis (NEW)
- **Section 7**: Non-Flux Warning Events (NEW)
- **Section 8**: Velero Backup Analysis (NEW)
- **Section 9**: RabbitMQ Stability (renumbered from 6)
- **Section 10**: Namespace Analysis (renumbered from 7)
- **Section 11**: AI Insights (renumbered from 8)

### Usage

The tool automatically collects these new metrics when run:

```bash
./k8s-analyzer
```

No additional flags required - all new sections are automatically included in the generated report.

### Benefits

1. **Operational Visibility**: Track GitOps reconciliation health in real-time
2. **Proactive Monitoring**: Identify backup issues before data loss occurs
3. **Faster Troubleshooting**: Quickly identify patterns in cluster events
4. **Trend Analysis**: Compare 24h vs 48h metrics to spot deteriorating conditions
5. **Comprehensive Coverage**: Combines Flux-specific and general cluster monitoring

### Example Output

Each section includes:
- Summary statistics (counts, trends)
- Detailed tables with top events/backups
- Filtering and sorting for relevance
- Actionable recommendations
- Time-based analysis (recent vs historical)

### Known Limitations

- Velero CRD must be installed for backup analysis
- Event history limited by Kubernetes TTL (typically 1 hour)
- Flux detection based on component/kind patterns
- Requires cluster-wide read permissions for all resources
