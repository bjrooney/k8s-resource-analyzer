# Pod Resource Appendix Feature

## Overview

Added a comprehensive **Appendix B** section to the cluster analysis report that provides a complete inventory of all running pods with their resource configurations and **actual usage metrics**.

## Latest Updates (v2)

### Cluster Name in Report Header
- Report now displays the cluster name at the top
- Extracted from kubeconfig context or cluster server URL
- Makes it easy to identify which cluster was analyzed

### Live Metrics Integration
- Integrated with Kubernetes metrics-server
- Shows **actual CPU and Memory usage** for each container
- Enables comparison of usage vs configured limits
- Falls back gracefully if metrics-server is not available

## Feature Details

### What's Included

The appendix now contains a detailed table for every running container in the cluster, organized by namespace:

**For Each Container:**
- Namespace
- Pod Name
- Container Name
- CPU Request (or "Not Set")
- CPU Limit (or "Not Set")
- Memory Request (or "Not Set")
- Memory Limit (or "Not Set")
- **Current CPU Usage** (from metrics-server)
- **Current Memory Usage** (from metrics-server)
- Status

### Summary Statistics

At the end of the appendix, summary statistics show:
- **Total Running Containers**: Total count
- **Fully Configured**: Count and percentage with both requests and limits
- **Missing Requests**: Count and percentage
- **Missing Limits**: Count and percentage

### Organization

Pods are grouped by namespace with:
- Alphabetical sorting of namespaces
- Within each namespace: sorted by pod name, then container name
- Container count per namespace in section header

## Implementation

### New Data Structure

```go
type PodResourceInfo struct {
    Namespace       string
    PodName         string
    ContainerName   string
    Status          string
    CPURequest      string
    CPULimit        string
    MemoryRequest   string
    MemoryLimit     string
    CurrentCPU      string    // Actual usage from metrics-server
    CurrentMemory   string    // Actual usage from metrics-server
}
```

### Changes Made

**analyzer.go:**
- Added `PodResourceInfo` struct
- Added `getClusterName()` helper function
- Added `getPodMetrics()` function for metrics-server integration
- Added metrics collection to `CollectClusterData()`
- Updated `ClusterData` struct to include cluster name and pod metrics

**report.go:**
- Added `corev1` import for Kubernetes types
- Updated report header to show cluster name
- Updated `generateAppendix()` function to:
  - Collect all running pod resource information
  - Group by namespace
  - Generate detailed tables
  - Calculate summary statistics
  - Renamed existing sections (A, B, C, D, E)

**Appendix Structure:**
- **Section A**: Data Collection Summary
- **Section B**: All Active Pods - Resource Configuration (NEW)
- **Section C**: Next Steps
- **Section D**: Useful Commands (NEW)
- **Section E**: Resources (expanded)

## Use Cases

1. **Compliance Auditing**: Quickly verify which pods are missing resource configurations
2. **Capacity Planning**: Export complete resource allocation data
3. **Cost Optimization**: Identify over-provisioned containers
4. **Troubleshooting**: Cross-reference resource settings with performance issues
5. **Documentation**: Maintain records of cluster resource allocation

## Example Output

```markdown
# Kubernetes Cluster Analysis Report

**Cluster:** production-aks-eastus  
**Generated:** 2025-10-27T14:55:00Z

---

#### Namespace: `production` (42 containers)

| Pod | Container | CPU Req | CPU Limit | Mem Req | Mem Limit | Current CPU | Current Mem | Status |
|-----|-----------|---------|-----------|---------|-----------|-------------|-------------|--------|
| `api-server-abc` | `api` | 500m | 1000m | 512Mi | 1Gi | 234m (46.8%) | 678Mi (66.2%) | Running |
| `api-server-abc` | `sidecar` | 100m | 200m | 128Mi | 256Mi | 45m (45.0%) | 89Mi (34.8%) | Running |
| `cache-redis-0` | `redis` | Not Set | Not Set | Not Set | Not Set | 123m | 456Mi | Running |
| `worker-xyz-123` | `worker` | 1000m | 2000m | 2Gi | 4Gi | 891m (89.1%) | 3.2Gi (80.0%) | Running |

**Note:** Metrics-server is available. Current usage reflects live data.

#### Resource Configuration Summary

- **Total Running Containers**: 387
- **Fully Configured** (requests + limits): 245 (63.3%)
- **Missing Requests**: 142 (36.7%)
- **Missing Limits**: 198 (51.2%)
```

## Benefits of Metrics Integration
- **Usage vs Allocation**: Percentage of requested/limited resources being used

## Benefits of Metrics Integration

1. **Right-Sizing Analysis**: See actual usage vs configured limits to identify over/under-provisioned pods
2. **Cost Optimization**: Identify containers using far less than their allocated resources
3. **Performance Tuning**: Spot containers hitting their limits that may need more resources
4. **Capacity Planning**: Use actual usage data for accurate future capacity estimates
5. **Trend Analysis**: Compare reports over time to track usage patterns

### Metrics-Server Detection

- Tool automatically detects if metrics-server is available
- Gracefully falls back to "N/A" if metrics unavailable
- Shows note in report indicating metrics availability status

## Benefits

1. **Complete Visibility**: Single source of truth for all pod resource configurations and usage
2. **Easy Auditing**: Quickly scan for misconfigured workloads
3. **Export Ready**: Tables can be easily converted to CSV/Excel for reporting
4. **Namespace Isolation**: Clear view of resource allocation per tenant/team
5. **Trend Analysis**: Compare reports over time to track configuration drift
6. **Cluster Identification**: Cluster name in header makes multi-cluster analysis easier

## Testing

Build and run:
```bash
cd /home/brendan/projects/sita/scripts/k8s-resource-analyzer
go build -o bin/k8s-analyzer .
./bin/k8s-analyzer
```

The generated report will include:
- Cluster name in the header
- Complete pod inventory with live metrics (if available)
- Usage percentages when limits are configured

## Documentation Updates

- ✅ README.md: Added Appendix to report sections list
- ✅ README.md: Added "Appendix Features" section with detailed explanation
- ✅ FEATURE_APPENDIX.md: This document
- ✅ Code comments in analyzer.go and report.go
