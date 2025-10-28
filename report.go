package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
)

func GenerateReport(data *ClusterData, analysis *Analysis) string {
	var sb strings.Builder

	// Header
	sb.WriteString("# Kubernetes Cluster Analysis Report\n\n")
	sb.WriteString(fmt.Sprintf("**Cluster:** `%s`\n\n", data.ClusterName))
	sb.WriteString(fmt.Sprintf("**Generated:** %s\n\n", time.Now().Format(time.RFC3339)))
	sb.WriteString("---\n\n")

	// Cluster Health Summary
	sb.WriteString(generateHealthSection(data, analysis))

	// Critical Issues
	sb.WriteString(generateCriticalIssuesSection(analysis))

	// Resource Management
	sb.WriteString(generateResourceManagementSection(analysis))

	// Node Analysis
	sb.WriteString(generateNodeAnalysisSection(analysis))

	// Pod Restarts Analysis
	sb.WriteString(generatePodRestartsSection(analysis))

	// Flux Events Analysis
	sb.WriteString(generateFluxEventsSection(analysis))

	// Non-Flux Events Analysis
	sb.WriteString(generateNonFluxEventsSection(analysis))

	// Velero Backups Analysis
	sb.WriteString(generateVeleroBackupsSection(analysis))

	// RabbitMQ Stability
	sb.WriteString(generateRabbitMQSection(analysis))

	// Namespace Analysis
	sb.WriteString(generateNamespaceAnalysisSection(analysis))

	// AI Insights (if available)
	if analysis.AIInsights != nil {
		sb.WriteString(generateAIInsightsSection(analysis.AIInsights))
	}

	// Appendix
	sb.WriteString(generateAppendix(data, analysis))

	return sb.String()
}

func generateHealthSection(data *ClusterData, analysis *Analysis) string {
	var sb strings.Builder

	sb.WriteString("## 1. Cluster Health Summary\n\n")

	healthEmoji := "🟢"
	if analysis.ClusterHealth == "critical" {
		healthEmoji = "🔴"
	} else if analysis.ClusterHealth == "degraded" {
		healthEmoji = "🟡"
	}

	sb.WriteString(fmt.Sprintf("%s **Overall Health**: %s\n\n", healthEmoji, strings.ToUpper(analysis.ClusterHealth)))

	sb.WriteString("### Key Metrics\n\n")
	sb.WriteString("| Metric | Value |\n")
	sb.WriteString("|--------|-------|\n")
	sb.WriteString(fmt.Sprintf("| Total Pods | %d |\n", len(data.Pods)))
	sb.WriteString(fmt.Sprintf("| Total Nodes | %d |\n", len(data.Nodes)))
	sb.WriteString(fmt.Sprintf("| Pods Missing Resources | %d |\n", len(analysis.ResourceGaps)))
	sb.WriteString(fmt.Sprintf("| OOM Events (Recent) | %d |\n", len(analysis.OOMEvents)))
	sb.WriteString(fmt.Sprintf("| Pods with Restarts (24h) | %d |\n", analysis.PodRestarts.TotalPods24h))
	sb.WriteString(fmt.Sprintf("| Pods with Restarts (7d) | %d |\n", analysis.PodRestarts.TotalPods7d))
	sb.WriteString(fmt.Sprintf("| Node Issues | %d |\n", len(analysis.NodeIssues)))
	sb.WriteString(fmt.Sprintf("| Namespaces at Risk | %d |\n\n", countHighRiskNamespaces(analysis.NamespaceAnalysis)))

	// Potential Issues
	if len(analysis.CriticalIssues) > 0 {
		sb.WriteString("### ⚠️ Potential Issues Identified\n\n")
		for _, issue := range analysis.CriticalIssues {
			sb.WriteString(fmt.Sprintf("- **%s**: %s\n", issue.Title, issue.Description))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func generateCriticalIssuesSection(analysis *Analysis) string {
	var sb strings.Builder

	sb.WriteString("## 2. Critical Issues (Top 5)\n\n")

	if len(analysis.CriticalIssues) == 0 {
		sb.WriteString("✅ No critical issues detected.\n\n")
		return sb.String()
	}

	for i, issue := range analysis.CriticalIssues {
		if i >= 5 {
			break
		}

		sb.WriteString(fmt.Sprintf("### Issue #%d: %s\n\n", i+1, issue.Title))
		sb.WriteString(fmt.Sprintf("**Priority**: %d (1=Highest)\n\n", issue.Priority))
		sb.WriteString(fmt.Sprintf("**Description**: %s\n\n", issue.Description))
		sb.WriteString(fmt.Sprintf("**Impact**: %s\n\n", issue.Impact))

		sb.WriteString("**Recommendation**:\n\n")
		sb.WriteString(fmt.Sprintf("%s\n\n", issue.Recommendation))

		if len(issue.Examples) > 0 {
			sb.WriteString("**Examples**:\n\n")
			for _, example := range issue.Examples {
				sb.WriteString(fmt.Sprintf("- `%s`\n", example))
			}
			sb.WriteString("\n")
		}

		sb.WriteString("**Action Items**:\n\n")
		sb.WriteString(generateActionItems(issue))
		sb.WriteString("\n---\n\n")
	}

	return sb.String()
}

func generateResourceManagementSection(analysis *Analysis) string {
	var sb strings.Builder

	sb.WriteString("## 3. Resource Management Analysis\n\n")

	sb.WriteString("### Missing Resource Requests and Limits\n\n")

	if len(analysis.ResourceGaps) == 0 {
		sb.WriteString("✅ All pods have resource requests and limits configured.\n\n")
	} else {
		missingRequests := 0
		missingLimits := 0
		missingBoth := 0

		for _, gap := range analysis.ResourceGaps {
			if gap.MissingRequests && gap.MissingLimits {
				missingBoth++
			} else if gap.MissingRequests {
				missingRequests++
			} else if gap.MissingLimits {
				missingLimits++
			}
		}

		sb.WriteString(fmt.Sprintf("- **Missing Both**: %d containers\n", missingBoth))
		sb.WriteString(fmt.Sprintf("- **Missing Requests Only**: %d containers\n", missingRequests))
		sb.WriteString(fmt.Sprintf("- **Missing Limits Only**: %d containers\n\n", missingLimits))

		sb.WriteString("### Impact on Cluster Operations\n\n")
		sb.WriteString("**Velero Backups**:\n")
		sb.WriteString("- Pods without resource requests may not be properly backed up\n")
		sb.WriteString("- Restore operations may fail due to resource allocation issues\n")
		sb.WriteString("- Recommendation: Set resource requests to ensure Velero can calculate backup requirements\n\n")

		sb.WriteString("**System Pods**:\n")
		sb.WriteString("- System pods may be evicted when resource-constrained workloads consume all node resources\n")
		sb.WriteString("- Can lead to cluster instability and monitoring gaps\n")
		sb.WriteString("- Recommendation: Implement ResourceQuota and LimitRange policies\n\n")

		sb.WriteString("**Cluster Stability**:\n")
		sb.WriteString("- Without requests, scheduler cannot make informed placement decisions\n")
		sb.WriteString("- Without limits, pods can consume excessive resources and impact neighbors\n")
		sb.WriteString("- May trigger cascading failures during traffic spikes\n\n")
	}

	// Short-lived jobs impact
	if analysis.ShortLivedJobs.TotalJobs > 0 {
		sb.WriteString("### Short-Lived Jobs Impact\n\n")
		sb.WriteString(fmt.Sprintf("- **Total Jobs**: %d\n", analysis.ShortLivedJobs.TotalJobs))
		sb.WriteString(fmt.Sprintf("- **Short Jobs (<2 min)**: %d\n", analysis.ShortLivedJobs.ShortJobs))

		if analysis.ShortLivedJobs.ShortJobs > 0 {
			percentage := float64(analysis.ShortLivedJobs.ShortJobs) / float64(analysis.ShortLivedJobs.TotalJobs) * 100
			sb.WriteString(fmt.Sprintf("- **Percentage**: %.1f%%\n\n", percentage))

			if percentage > 30 {
				sb.WriteString("**Impact**: High frequency of short-lived jobs can:\n")
				sb.WriteString("- Create scheduling churn and API server load\n")
				sb.WriteString("- Complicate resource capacity planning\n")
				sb.WriteString("- Impact cluster autoscaler effectiveness\n\n")
				sb.WriteString("**Recommendations**:\n")
				sb.WriteString("- Consider batching short jobs or using longer-running workers with queue patterns\n")
				sb.WriteString("- Set appropriate resource requests to prevent over-provisioning\n")
				sb.WriteString("- Implement job cleanup policies to prevent accumulation\n\n")
			}
		}

		// Job Execution Details by Namespace
		if len(analysis.ShortLivedJobs.JobExecutions) > 0 {
			sb.WriteString("### Job Execution Details (Last 24 Hours)\n\n")
			sb.WriteString("Analysis of job pod executions per namespace showing frequency and node distribution.\n\n")

			// Sort namespaces for consistent output
			namespaces := make([]string, 0, len(analysis.ShortLivedJobs.JobExecutions))
			for ns := range analysis.ShortLivedJobs.JobExecutions {
				namespaces = append(namespaces, ns)
			}
			sort.Strings(namespaces)

			for _, ns := range namespaces {
				executions := analysis.ShortLivedJobs.JobExecutions[ns]
				if len(executions) == 0 {
					continue
				}

				sb.WriteString(fmt.Sprintf("#### Namespace: `%s`\n\n", ns))
				sb.WriteString(fmt.Sprintf("**Total Job Executions**: %d in last 24 hours\n\n", len(executions)))

				// Count executions per job and track nodes
				jobCounts := make(map[string]int)
				jobNodes := make(map[string]map[string]int) // jobName -> nodeName -> count
				for _, exec := range executions {
					jobCounts[exec.JobName]++
					if jobNodes[exec.JobName] == nil {
						jobNodes[exec.JobName] = make(map[string]int)
					}
					jobNodes[exec.JobName][exec.NodeName]++
				}

				// Create summary table
				sb.WriteString("| Job Name | Executions | Nodes Used | Status Summary |\n")
				sb.WriteString("|----------|------------|------------|----------------|\n")

				// Sort jobs by execution count (descending)
				type jobCount struct {
					name  string
					count int
				}
				sortedJobs := make([]jobCount, 0, len(jobCounts))
				for job, count := range jobCounts {
					sortedJobs = append(sortedJobs, jobCount{job, count})
				}
				sort.Slice(sortedJobs, func(i, j int) bool {
					return sortedJobs[i].count > sortedJobs[j].count
				})

				for _, jc := range sortedJobs {
					nodes := jobNodes[jc.name]
					nodeList := make([]string, 0, len(nodes))
					for node := range nodes {
						nodeList = append(nodeList, node)
					}
					sort.Strings(nodeList)

					// Get status summary for this job
					succeeded := 0
					failed := 0
					running := 0
					for _, exec := range executions {
						if exec.JobName == jc.name {
							switch exec.Status {
							case "Succeeded":
								succeeded++
							case "Failed":
								failed++
							case "Running":
								running++
							}
						}
					}

					statusSummary := fmt.Sprintf("✅ %d", succeeded)
					if failed > 0 {
						statusSummary += fmt.Sprintf(" ❌ %d", failed)
					}
					if running > 0 {
						statusSummary += fmt.Sprintf(" ⏳ %d", running)
					}

					// Format node list
					nodeStr := fmt.Sprintf("%d node(s)", len(nodes))
					if len(nodes) <= 3 {
						nodeStr = strings.Join(nodeList, ", ")
					}

					sb.WriteString(fmt.Sprintf("| `%s` | %d | %s | %s |\n",
						jc.name, jc.count, nodeStr, statusSummary))
				}
				sb.WriteString("\n")

				// Show detailed node distribution for high-frequency jobs
				for _, jc := range sortedJobs {
					if jc.count >= 5 { // Only show details for jobs that ran 5+ times
						nodes := jobNodes[jc.name]
						if len(nodes) > 1 {
							// Recreate node list for this job
							jobNodeList := make([]string, 0, len(nodes))
							for node := range nodes {
								jobNodeList = append(jobNodeList, node)
							}
							sort.Strings(jobNodeList)

							sb.WriteString(fmt.Sprintf("**`%s` node distribution**:\n", jc.name))
							for _, node := range jobNodeList {
								if count := nodes[node]; count > 0 {
									sb.WriteString(fmt.Sprintf("- %s: %d executions\n", node, count))
								}
							}
							sb.WriteString("\n")
						}
					}
				}
			}

			sb.WriteString("**Legend**: ✅ Succeeded | ❌ Failed | ⏳ Running\n\n")
		}
	}

	sb.WriteString("### Recommended Resource Allocation Strategy\n\n")
	sb.WriteString("```yaml\n")
	sb.WriteString("# Example resource configuration\n")
	sb.WriteString("resources:\n")
	sb.WriteString("  requests:\n")
	sb.WriteString("    memory: \"256Mi\"  # Based on observed baseline usage\n")
	sb.WriteString("    cpu: \"100m\"      # 0.1 CPU cores\n")
	sb.WriteString("  limits:\n")
	sb.WriteString("    memory: \"512Mi\"  # 2x requests for burst capacity\n")
	sb.WriteString("    cpu: \"500m\"      # Allow bursting up to 0.5 cores\n")
	sb.WriteString("```\n\n")

	return sb.String()
}

func generateNodeAnalysisSection(analysis *Analysis) string {
	var sb strings.Builder

	sb.WriteString("## 4. Node Analysis\n\n")

	if len(analysis.NodeIssues) == 0 {
		sb.WriteString("✅ All nodes have healthy resource allocation.\n\n")
		return sb.String()
	}

	sb.WriteString("### Nodes with High Resource Utilization\n\n")
	sb.WriteString("| Node Name | Issue | CPU Requested | Memory Requested | CPU Allocatable | Memory Allocatable |\n")
	sb.WriteString("|-----------|-------|---------------|------------------|-----------------|--------------------|\n")

	for _, issue := range analysis.NodeIssues {
		sb.WriteString(fmt.Sprintf("| %s | %s | %.2f cores | %.2f GB | %.2f cores | %.2f GB |\n",
			issue.NodeName, issue.Issue,
			issue.RequestedCPU, issue.RequestedMemory,
			issue.AllocatableCPU, issue.AllocatableMemory))
	}
	sb.WriteString("\n")

	sb.WriteString("### Recommendations\n\n")
	sb.WriteString("1. **Node Pool Balancing**:\n")
	sb.WriteString("   - Review node pool sizing and consider adding nodes\n")
	sb.WriteString("   - Implement pod anti-affinity to spread workloads\n")
	sb.WriteString("   - Consider node taints/tolerations for workload isolation\n\n")

	sb.WriteString("2. **Autoscaling Configuration**:\n")
	sb.WriteString("   ```yaml\n")
	sb.WriteString("   # Cluster Autoscaler Settings\n")
	sb.WriteString("   scaleDownUtilizationThreshold: 0.65\n")
	sb.WriteString("   scaleDownDelayAfterAdd: 10m\n")
	sb.WriteString("   scaleDownUnneededTime: 10m\n")
	sb.WriteString("   maxNodeProvisionTime: 15m\n")
	sb.WriteString("   ```\n\n")

	sb.WriteString("3. **Pod Distribution Strategy**:\n")
	sb.WriteString("   - Use PodTopologySpreadConstraints for even distribution\n")
	sb.WriteString("   - Set appropriate resource requests to enable efficient packing\n")
	sb.WriteString("   - Review and optimize large workloads that may be causing imbalance\n\n")

	// OOM Events
	if len(analysis.OOMEvents) > 0 {
		sb.WriteString("### OOMKilled Events\n\n")
		sb.WriteString(fmt.Sprintf("Found %d OOMKilled events:\n\n", len(analysis.OOMEvents)))

		sb.WriteString("| Timestamp | Namespace | Pod | Container | Node |\n")
		sb.WriteString("|-----------|-----------|-----|-----------|------|\n")

		for i, event := range analysis.OOMEvents {
			if i >= 10 { // Show top 10
				sb.WriteString(fmt.Sprintf("\n_... and %d more events_\n\n", len(analysis.OOMEvents)-10))
				break
			}
			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s |\n",
				event.Timestamp.Format("2006-01-02 15:04:05"),
				event.Namespace, event.PodName, event.Container, event.NodeName))
		}
		sb.WriteString("\n")

		sb.WriteString("**Action Required**:\n")
		sb.WriteString("- Increase memory limits for affected pods\n")
		sb.WriteString("- Investigate application memory leaks\n")
		sb.WriteString("- Consider implementing memory profiling\n\n")
	}

	return sb.String()
}

func generateFluxEventsSection(analysis *Analysis) string {
	var sb strings.Builder

	sb.WriteString("## 6. Flux Events Analysis\n\n")

	total24h := len(analysis.FluxEvents.Last24Hours)
	total48h := len(analysis.FluxEvents.Last48Hours)

	if total24h == 0 && total48h == 0 {
		sb.WriteString("ℹ️ No Flux events detected in the last 48 hours.\n\n")
		return sb.String()
	}

	// Summary
	sb.WriteString("### Summary\n\n")
	sb.WriteString(fmt.Sprintf("- **Last 24 Hours**: %d events (%d warnings, %d errors)\n",
		total24h, analysis.FluxEvents.Warnings24h, analysis.FluxEvents.Errors24h))
	sb.WriteString(fmt.Sprintf("- **Last 48 Hours**: %d events (%d warnings, %d errors)\n\n",
		total48h, analysis.FluxEvents.Warnings48h, analysis.FluxEvents.Errors48h))

	// Warnings in last 24 hours
	if analysis.FluxEvents.Warnings24h > 0 || analysis.FluxEvents.Errors24h > 0 {
		sb.WriteString("### Flux Events in Last 24 Hours\n\n")

		warnings24h := []EventInfo{}
		for _, event := range analysis.FluxEvents.Last24Hours {
			warnings24h = append(warnings24h, event)
		}

		if len(warnings24h) > 20 {
			sb.WriteString(fmt.Sprintf("Showing top 20 of %d events:\n\n", len(warnings24h)))
		}

		sb.WriteString("| Type | Namespace | Object | Reason | Message | Count | Last Seen |\n")
		sb.WriteString("|------|-----------|--------|--------|---------|-------|----------|\n")

		for i, event := range warnings24h {
			if i >= 20 {
				sb.WriteString(fmt.Sprintf("\n_... and %d more events_\n\n", len(warnings24h)-20))
				break
			}

			message := event.Message
			if len(message) > 80 {
				message = message[:77] + "..."
			}

			timeStr := "Unknown"
			if !event.LastTime.IsZero() {
				timeStr = event.LastTime.Format("2006-01-02 15:04")
			}

			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %d | %s |\n",
				event.Type,
				event.Namespace,
				event.InvolvedObject,
				event.Reason,
				message,
				event.Count,
				timeStr))
		}
		sb.WriteString("\n")
	}

	// Additional events in last 48 hours
	if len(analysis.FluxEvents.Last48Hours) > len(analysis.FluxEvents.Last24Hours) {
		sb.WriteString("### Additional Flux Events in Last 48 Hours (excluding above)\n\n")

		// Filter to get only 48h events not in 24h
		recent := make(map[string]bool)
		for _, e := range analysis.FluxEvents.Last24Hours {
			recent[e.Namespace+e.InvolvedObject+e.Reason] = true
		}

		older := []EventInfo{}
		for _, e := range analysis.FluxEvents.Last48Hours {
			key := e.Namespace + e.InvolvedObject + e.Reason
			if !recent[key] {
				older = append(older, e)
			}
		}

		if len(older) > 0 {
			if len(older) > 20 {
				sb.WriteString(fmt.Sprintf("Showing top 20 of %d events:\n\n", len(older)))
			}

			sb.WriteString("| Type | Namespace | Object | Reason | Message | Count | Last Seen |\n")
			sb.WriteString("|------|-----------|--------|--------|---------|-------|----------|\n")

			for i, event := range older {
				if i >= 20 {
					sb.WriteString(fmt.Sprintf("\n_... and %d more events_\n\n", len(older)-20))
					break
				}

				message := event.Message
				if len(message) > 80 {
					message = message[:77] + "..."
				}

				timeStr := "Unknown"
				if !event.LastTime.IsZero() {
					timeStr = event.LastTime.Format("2006-01-02 15:04")
				}

				sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %d | %s |\n",
					event.Type,
					event.Namespace,
					event.InvolvedObject,
					event.Reason,
					message,
					event.Count,
					timeStr))
			}
			sb.WriteString("\n")
		}
	}

	sb.WriteString("### Recommendations\n\n")
	sb.WriteString("1. **Review Warning Events**: Check Flux reconciliation failures\n")
	sb.WriteString("2. **Check Source Repositories**: Verify Git repositories are accessible\n")
	sb.WriteString("3. **Validate Manifests**: Ensure Kustomization and HelmRelease manifests are valid\n")
	sb.WriteString("4. **Monitor Flux Components**: Check flux-system namespace pod health\n\n")

	return sb.String()
}

func generateNonFluxEventsSection(analysis *Analysis) string {
	var sb strings.Builder

	sb.WriteString("## 7. Non-Flux Warning Events\n\n")

	total24h := len(analysis.NonFluxEvents.Last24Hours)
	total48h := len(analysis.NonFluxEvents.Last48Hours)

	if total24h == 0 && total48h == 0 {
		sb.WriteString("✅ No warning events detected in the last 48 hours.\n\n")
		return sb.String()
	}

	// Summary
	sb.WriteString("### Summary\n\n")
	sb.WriteString(fmt.Sprintf("- **Last 24 Hours**: %d warning events\n", analysis.NonFluxEvents.Warnings24h))
	sb.WriteString(fmt.Sprintf("- **Last 48 Hours**: %d warning events\n\n", analysis.NonFluxEvents.Warnings48h))

	// Last 24 hours
	if len(analysis.NonFluxEvents.Last24Hours) > 0 {
		sb.WriteString("### Warning Events in Last 24 Hours\n\n")

		if len(analysis.NonFluxEvents.Last24Hours) > 25 {
			sb.WriteString(fmt.Sprintf("Showing top 25 of %d events:\n\n", len(analysis.NonFluxEvents.Last24Hours)))
		}

		sb.WriteString("| Namespace | Object | Reason | Message | Count | Last Seen |\n")
		sb.WriteString("|-----------|--------|--------|---------|-------|----------|\n")

		for i, event := range analysis.NonFluxEvents.Last24Hours {
			if i >= 25 {
				sb.WriteString(fmt.Sprintf("\n_... and %d more events_\n\n", len(analysis.NonFluxEvents.Last24Hours)-25))
				break
			}

			message := event.Message
			if len(message) > 100 {
				message = message[:97] + "..."
			}

			timeStr := "Unknown"
			if !event.LastTime.IsZero() {
				timeStr = event.LastTime.Format("2006-01-02 15:04")
			}

			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %d | %s |\n",
				event.Namespace,
				event.InvolvedObject,
				event.Reason,
				message,
				event.Count,
				timeStr))
		}
		sb.WriteString("\n")
	}

	// Additional events in last 48 hours
	if len(analysis.NonFluxEvents.Last48Hours) > len(analysis.NonFluxEvents.Last24Hours) {
		sb.WriteString("### Additional Warning Events in Last 48 Hours (excluding above)\n\n")

		// Filter
		recent := make(map[string]bool)
		for _, e := range analysis.NonFluxEvents.Last24Hours {
			recent[e.Namespace+e.InvolvedObject+e.Reason] = true
		}

		older := []EventInfo{}
		for _, e := range analysis.NonFluxEvents.Last48Hours {
			key := e.Namespace + e.InvolvedObject + e.Reason
			if !recent[key] {
				older = append(older, e)
			}
		}

		if len(older) > 0 {
			if len(older) > 25 {
				sb.WriteString(fmt.Sprintf("Showing top 25 of %d events:\n\n", len(older)))
			}

			sb.WriteString("| Namespace | Object | Reason | Message | Count | Last Seen |\n")
			sb.WriteString("|-----------|--------|--------|---------|-------|----------|\n")

			for i, event := range older {
				if i >= 25 {
					sb.WriteString(fmt.Sprintf("\n_... and %d more events_\n\n", len(older)-25))
					break
				}

				message := event.Message
				if len(message) > 100 {
					message = message[:97] + "..."
				}

				timeStr := "Unknown"
				if !event.LastTime.IsZero() {
					timeStr = event.LastTime.Format("2006-01-02 15:04")
				}

				sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %d | %s |\n",
					event.Namespace,
					event.InvolvedObject,
					event.Reason,
					message,
					event.Count,
					timeStr))
			}
			sb.WriteString("\n")
		}
	}

	sb.WriteString("### Recommendations\n\n")
	sb.WriteString("1. **Investigate Frequent Warnings**: Focus on events with high counts\n")
	sb.WriteString("2. **Check Resource Issues**: Look for scheduling, mounting, and resource-related warnings\n")
	sb.WriteString("3. **Review Pod Health**: Investigate liveness/readiness probe failures\n")
	sb.WriteString("4. **Monitor Trends**: Track if warning events are increasing over time\n\n")

	return sb.String()
}

func generateVeleroBackupsSection(analysis *Analysis) string {
	var sb strings.Builder

	sb.WriteString("## 8. Velero Backup Analysis\n\n")

	total24h := len(analysis.VeleroBackups.Last24Hours)
	total48h := len(analysis.VeleroBackups.Last48Hours)

	if total24h == 0 && total48h == 0 {
		sb.WriteString("ℹ️ No Velero backups detected in the last 48 hours.\n\n")
		sb.WriteString("This may indicate:\n")
		sb.WriteString("- Velero is not installed in this cluster\n")
		sb.WriteString("- No backups have been scheduled recently\n")
		sb.WriteString("- Backup schedule needs to be reviewed\n\n")
		return sb.String()
	}

	// Summary
	sb.WriteString("### Summary\n\n")
	sb.WriteString(fmt.Sprintf("- **Last 24 Hours**: %d backups (%d failed)\n",
		analysis.VeleroBackups.TotalBackups24h, analysis.VeleroBackups.FailedBackups24h))
	sb.WriteString(fmt.Sprintf("- **Last 48 Hours**: %d backups (%d failed)\n\n",
		analysis.VeleroBackups.TotalBackups48h, analysis.VeleroBackups.FailedBackups48h))

	// Last 24 hours
	if len(analysis.VeleroBackups.Last24Hours) > 0 {
		sb.WriteString("### Backups in Last 24 Hours\n\n")

		sb.WriteString("| Backup Name | Status | Start Time | Duration | Errors | Warnings |\n")
		sb.WriteString("|-------------|--------|------------|----------|--------|----------|\n")

		for _, backup := range analysis.VeleroBackups.Last24Hours {
			statusEmoji := "✅"
			if backup.Status == "Failed" {
				statusEmoji = "❌"
			} else if backup.Status == "PartiallyFailed" {
				statusEmoji = "⚠️"
			} else if backup.Status == "InProgress" {
				statusEmoji = "⏳"
			}

			durationStr := "In Progress"
			if backup.Duration > 0 {
				durationStr = fmt.Sprintf("%.1fm", backup.Duration.Minutes())
			}

			startTimeStr := backup.StartTime.Format("2006-01-02 15:04")

			sb.WriteString(fmt.Sprintf("| %s | %s %s | %s | %s | %d | %d |\n",
				backup.Name,
				statusEmoji,
				backup.Status,
				startTimeStr,
				durationStr,
				backup.Errors,
				backup.Warnings))
		}
		sb.WriteString("\n")
	}

	// Last 48 hours (excluding 24h)
	if len(analysis.VeleroBackups.Last48Hours) > len(analysis.VeleroBackups.Last24Hours) {
		sb.WriteString("### Additional Backups in Last 48 Hours (excluding above)\n\n")

		// Filter
		recent := make(map[string]bool)
		for _, b := range analysis.VeleroBackups.Last24Hours {
			recent[b.Name] = true
		}

		older := []VeleroBackup{}
		for _, b := range analysis.VeleroBackups.Last48Hours {
			if !recent[b.Name] {
				older = append(older, b)
			}
		}

		if len(older) > 0 {
			sb.WriteString("| Backup Name | Status | Start Time | Duration | Errors | Warnings |\n")
			sb.WriteString("|-------------|--------|------------|----------|--------|----------|\n")

			for _, backup := range older {
				statusEmoji := "✅"
				if backup.Status == "Failed" {
					statusEmoji = "❌"
				} else if backup.Status == "PartiallyFailed" {
					statusEmoji = "⚠️"
				}

				durationStr := "In Progress"
				if backup.Duration > 0 {
					durationStr = fmt.Sprintf("%.1fm", backup.Duration.Minutes())
				}

				startTimeStr := backup.StartTime.Format("2006-01-02 15:04")

				sb.WriteString(fmt.Sprintf("| %s | %s %s | %s | %s | %d | %d |\n",
					backup.Name,
					statusEmoji,
					backup.Status,
					startTimeStr,
					durationStr,
					backup.Errors,
					backup.Warnings))
			}
			sb.WriteString("\n")
		}
	}

	// Analysis
	sb.WriteString("### Analysis\n\n")

	if analysis.VeleroBackups.FailedBackups24h > 0 {
		sb.WriteString(fmt.Sprintf("⚠️ **Failed Backups**: %d backup(s) failed in the last 24 hours\n\n",
			analysis.VeleroBackups.FailedBackups24h))
	}

	// Calculate average duration
	if len(analysis.VeleroBackups.Last24Hours) > 0 {
		totalDuration := time.Duration(0)
		completedBackups := 0
		for _, b := range analysis.VeleroBackups.Last24Hours {
			if b.Duration > 0 {
				totalDuration += b.Duration
				completedBackups++
			}
		}
		if completedBackups > 0 {
			avgDuration := totalDuration / time.Duration(completedBackups)
			sb.WriteString(fmt.Sprintf("**Average Backup Duration**: %.1f minutes\n\n", avgDuration.Minutes()))
		}
	}

	sb.WriteString("### Recommendations\n\n")
	sb.WriteString("1. **Review Failed Backups**:\n")
	sb.WriteString("   - Check logs: `velero backup describe <backup-name>`\n")
	sb.WriteString("   - Review errors: `velero backup logs <backup-name>`\n\n")

	sb.WriteString("2. **Validate Backup Schedule**:\n")
	sb.WriteString("   - Ensure backups are running at expected intervals\n")
	sb.WriteString("   - Check backup retention policies\n\n")

	sb.WriteString("3. **Monitor Backup Storage**:\n")
	sb.WriteString("   - Verify backup storage location is accessible\n")
	sb.WriteString("   - Check available storage space\n\n")

	sb.WriteString("4. **Test Restore Process**:\n")
	sb.WriteString("   - Regularly test backup restoration\n")
	sb.WriteString("   - Validate backup integrity\n\n")

	return sb.String()
}

func generateRabbitMQSection(analysis *Analysis) string {
	var sb strings.Builder

	sb.WriteString("## 9. RabbitMQ Stability Analysis\n\n")

	if len(analysis.RabbitMQFindings.RabbitMQPods) == 0 {
		sb.WriteString("ℹ️ No RabbitMQ pods detected in the cluster.\n\n")
		return sb.String()
	}

	sb.WriteString(fmt.Sprintf("**RabbitMQ Pods Found**: %d\n\n", len(analysis.RabbitMQFindings.RabbitMQPods)))

	for _, pod := range analysis.RabbitMQFindings.RabbitMQPods {
		sb.WriteString(fmt.Sprintf("- `%s`\n", pod))
	}
	sb.WriteString("\n")

	sb.WriteString("### Current Configuration\n\n")
	sb.WriteString(fmt.Sprintf("- ✓ Priority Class Configured: %v\n", analysis.RabbitMQFindings.HasPriorityClass))
	sb.WriteString(fmt.Sprintf("- ✓ Resource Limits Set: %v\n\n", analysis.RabbitMQFindings.HasResourceLimits))

	// OOM Events in Last 7 Days
	if len(analysis.RabbitMQFindings.OOMEventsLast7d) > 0 {
		sb.WriteString("### ⚠️ OOM Events in Last 7 Days\n\n")
		sb.WriteString(fmt.Sprintf("**Total OOM Events**: %d\n\n", len(analysis.RabbitMQFindings.OOMEventsLast7d)))

		sb.WriteString("| Timestamp | Pod | Namespace | Node | Container |\n")
		sb.WriteString("|-----------|-----|-----------|------|--------|\n")

		for _, oom := range analysis.RabbitMQFindings.OOMEventsLast7d {
			timeStr := oom.Timestamp.Format("2006-01-02 15:04:05")
			container := oom.Container
			if container == "" {
				container = "N/A"
			}
			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s |\n",
				timeStr,
				oom.PodName,
				oom.Namespace,
				oom.NodeName,
				container))
		}
		sb.WriteString("\n")
		sb.WriteString("⚠️ **Action Required**: RabbitMQ pods are experiencing OOM kills. Review and increase memory limits immediately.\n\n")
	} else {
		sb.WriteString("### ✅ OOM Events in Last 7 Days\n\n")
		sb.WriteString("No OOM events detected for RabbitMQ pods in the last 7 days.\n\n")
	}

	sb.WriteString("### Recommendations for Maximum Stability\n\n")

	sb.WriteString("#### 1. Create High-Priority PriorityClass\n\n")
	sb.WriteString("```yaml\n")
	sb.WriteString("apiVersion: scheduling.k8s.io/v1\n")
	sb.WriteString("kind: PriorityClass\n")
	sb.WriteString("metadata:\n")
	sb.WriteString("  name: rabbitmq-critical\n")
	sb.WriteString("value: 1000000  # Higher than system-cluster-critical (2000000000 reserved for system)\n")
	sb.WriteString("globalDefault: false\n")
	sb.WriteString("description: \"Priority class for RabbitMQ to prevent eviction\"\n")
	sb.WriteString("```\n\n")

	sb.WriteString("#### 2. Configure RabbitMQ Pod Resources\n\n")
	sb.WriteString("```yaml\n")
	sb.WriteString("apiVersion: v1\n")
	sb.WriteString("kind: Pod\n")
	sb.WriteString("metadata:\n")
	sb.WriteString("  name: rabbitmq\n")
	sb.WriteString("spec:\n")
	sb.WriteString("  priorityClassName: rabbitmq-critical\n")
	sb.WriteString("  containers:\n")
	sb.WriteString("  - name: rabbitmq\n")
	sb.WriteString("    resources:\n")
	sb.WriteString("      requests:\n")
	sb.WriteString("        memory: \"2Gi\"    # Set based on your observed usage\n")
	sb.WriteString("        cpu: \"1000m\"     # 1 full CPU core\n")
	sb.WriteString("      limits:\n")
	sb.WriteString("        memory: \"4Gi\"    # Allow headroom for spikes\n")
	sb.WriteString("        cpu: \"2000m\"     # Allow burst capacity\n")
	sb.WriteString("```\n\n")

	sb.WriteString("#### 3. Add PodDisruptionBudget\n\n")
	sb.WriteString("```yaml\n")
	sb.WriteString("apiVersion: policy/v1\n")
	sb.WriteString("kind: PodDisruptionBudget\n")
	sb.WriteString("metadata:\n")
	sb.WriteString("  name: rabbitmq-pdb\n")
	sb.WriteString("spec:\n")
	sb.WriteString("  minAvailable: 2  # For clustered RabbitMQ\n")
	sb.WriteString("  selector:\n")
	sb.WriteString("    matchLabels:\n")
	sb.WriteString("      app: rabbitmq\n")
	sb.WriteString("```\n\n")

	sb.WriteString("#### 4. Node Affinity (Optional)\n\n")
	sb.WriteString("Consider dedicating specific nodes for RabbitMQ:\n\n")
	sb.WriteString("```yaml\n")
	sb.WriteString("affinity:\n")
	sb.WriteString("  nodeAffinity:\n")
	sb.WriteString("    preferredDuringSchedulingIgnoredDuringExecution:\n")
	sb.WriteString("    - weight: 100\n")
	sb.WriteString("      preference:\n")
	sb.WriteString("        matchExpressions:\n")
	sb.WriteString("        - key: workload-type\n")
	sb.WriteString("          operator: In\n")
	sb.WriteString("          values:\n")
	sb.WriteString("          - messaging\n")
	sb.WriteString("```\n\n")

	sb.WriteString("### How This Ensures RabbitMQ is Last to be Evicted\n\n")
	sb.WriteString("1. **PriorityClass**: Kubernetes evicts lower-priority pods first during resource pressure\n")
	sb.WriteString("2. **Resource Requests**: Guarantees RabbitMQ gets its requested resources\n")
	sb.WriteString("3. **Resource Limits**: Prevents RabbitMQ from being OOMKilled unnecessarily\n")
	sb.WriteString("4. **PodDisruptionBudget**: Prevents voluntary disruptions during maintenance\n\n")

	return sb.String()
}

func generatePodRestartsSection(analysis *Analysis) string {
	var sb strings.Builder

	sb.WriteString("## 5. Pod Restart Analysis\n\n")

	total24h := analysis.PodRestarts.TotalPods24h
	total48h := analysis.PodRestarts.TotalPods48h
	total7d := analysis.PodRestarts.TotalPods7d

	if total24h == 0 && total48h == 0 && total7d == 0 {
		sb.WriteString("✅ No pod restarts detected in the last 7 days.\n\n")
		return sb.String()
	}

	// Summary
	sb.WriteString("### Summary\n\n")
	sb.WriteString(fmt.Sprintf("- **Last 24 Hours**: %d pods with restarts (%d total restarts)\n",
		total24h, len(analysis.PodRestarts.Last24Hours)))
	sb.WriteString(fmt.Sprintf("- **Last 48 Hours**: %d pods with restarts (%d total restarts)\n",
		total48h, len(analysis.PodRestarts.Last48Hours)))
	sb.WriteString(fmt.Sprintf("- **Last 7 Days**: %d pods with restarts (%d total restarts)\n\n",
		total7d, len(analysis.PodRestarts.Last7Days)))

	// Last 24 Hours
	if len(analysis.PodRestarts.Last24Hours) > 0 {
		sb.WriteString("### Restarts in Last 24 Hours\n\n")

		if len(analysis.PodRestarts.Last24Hours) > 20 {
			sb.WriteString(fmt.Sprintf("Showing top 20 of %d containers with restarts:\n\n",
				len(analysis.PodRestarts.Last24Hours)))
		}

		sb.WriteString("| Namespace | Pod | Container | Restart Count | Last Restart | Reason |\n")
		sb.WriteString("|-----------|-----|-----------|---------------|--------------|--------|\n")

		for i, restart := range analysis.PodRestarts.Last24Hours {
			if i >= 20 {
				sb.WriteString(fmt.Sprintf("\n_... and %d more containers with restarts_\n\n",
					len(analysis.PodRestarts.Last24Hours)-20))
				break
			}

			timeStr := "Unknown"
			if !restart.LastRestartTime.IsZero() {
				timeStr = restart.LastRestartTime.Format("2006-01-02 15:04")
			}

			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %d | %s | %s |\n",
				restart.Namespace,
				restart.PodName,
				restart.ContainerName,
				restart.RestartCount,
				timeStr,
				restart.Reason))
		}
		sb.WriteString("\n")
	}

	// Last 48 Hours (excluding 24h ones already shown)
	if len(analysis.PodRestarts.Last48Hours) > len(analysis.PodRestarts.Last24Hours) {
		sb.WriteString("### Restarts in Last 48 Hours (excluding above)\n\n")

		// Create map of 24h restarts to exclude
		recent := make(map[string]bool)
		for _, r := range analysis.PodRestarts.Last24Hours {
			recent[r.Namespace+"/"+r.PodName+"/"+r.ContainerName] = true
		}

		// Filter out recent ones
		older := []PodRestart{}
		for _, r := range analysis.PodRestarts.Last48Hours {
			key := r.Namespace + "/" + r.PodName + "/" + r.ContainerName
			if !recent[key] {
				older = append(older, r)
			}
		}

		if len(older) > 0 {
			if len(older) > 20 {
				sb.WriteString(fmt.Sprintf("Showing top 20 of %d containers:\n\n", len(older)))
			}

			sb.WriteString("| Namespace | Pod | Container | Restart Count | Last Restart | Reason |\n")
			sb.WriteString("|-----------|-----|-----------|---------------|--------------|--------|\n")

			for i, restart := range older {
				if i >= 20 {
					sb.WriteString(fmt.Sprintf("\n_... and %d more containers_\n\n", len(older)-20))
					break
				}

				timeStr := "Unknown"
				if !restart.LastRestartTime.IsZero() {
					timeStr = restart.LastRestartTime.Format("2006-01-02 15:04")
				}

				sb.WriteString(fmt.Sprintf("| %s | %s | %s | %d | %s | %s |\n",
					restart.Namespace,
					restart.PodName,
					restart.ContainerName,
					restart.RestartCount,
					timeStr,
					restart.Reason))
			}
			sb.WriteString("\n")
		}
	}

	// All Restarts in Last 7 Days
	if len(analysis.PodRestarts.Last7Days) > 0 {
		sb.WriteString("### All Restarts in the Last 7 Days\n\n")

		if len(analysis.PodRestarts.Last7Days) > 20 {
			sb.WriteString(fmt.Sprintf("Showing top 20 of %d containers:\n\n", len(analysis.PodRestarts.Last7Days)))
		}

		sb.WriteString("| Namespace | Pod | Container | Restart Count | Last Restart | Reason |\n")
		sb.WriteString("|-----------|-----|-----------|---------------|--------------|--------|\n")

		for i, restart := range analysis.PodRestarts.Last7Days {
			if i >= 20 {
				sb.WriteString(fmt.Sprintf("\n_... and %d more containers_\n\n", len(analysis.PodRestarts.Last7Days)-20))
				break
			}

			timeStr := "Unknown"
			if !restart.LastRestartTime.IsZero() {
				timeStr = restart.LastRestartTime.Format("2006-01-02 15:04")
			}

			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %d | %s | %s |\n",
				restart.Namespace,
				restart.PodName,
				restart.ContainerName,
				restart.RestartCount,
				timeStr,
				restart.Reason))
		}
		sb.WriteString("\n")
	}

	// Analysis and recommendations
	sb.WriteString("### Analysis\n\n")

	if total24h > 10 {
		sb.WriteString("⚠️ **High Restart Rate**: Significant number of pod restarts in the last 24 hours.\n\n")
	}

	sb.WriteString("**Common Restart Reasons**:\n")

	// Count reasons
	reasonCount := make(map[string]int)
	for _, r := range analysis.PodRestarts.Last7Days {
		if r.Reason != "" && r.Reason != "Unknown" {
			reasonCount[r.Reason]++
		}
	}

	if len(reasonCount) > 0 {
		type reasonPair struct {
			reason string
			count  int
		}
		reasons := []reasonPair{}
		for r, c := range reasonCount {
			reasons = append(reasons, reasonPair{r, c})
		}
		sort.Slice(reasons, func(i, j int) bool {
			return reasons[i].count > reasons[j].count
		})

		for i, rp := range reasons {
			if i >= 5 {
				break
			}
			sb.WriteString(fmt.Sprintf("- **%s**: %d occurrences\n", rp.reason, rp.count))
		}
		sb.WriteString("\n")
	} else {
		sb.WriteString("- Most restarts have unknown/unspecified reasons\n\n")
	}

	sb.WriteString("### Recommendations\n\n")
	sb.WriteString("1. **Investigate High Restart Pods**:\n")
	sb.WriteString("   - Review logs: `kubectl logs <pod-name> --previous -n <namespace>`\n")
	sb.WriteString("   - Check events: `kubectl describe pod <pod-name> -n <namespace>`\n\n")

	sb.WriteString("2. **Address Common Issues**:\n")
	sb.WriteString("   - **OOMKilled**: Increase memory limits\n")
	sb.WriteString("   - **Error**: Check application logs for errors\n")
	sb.WriteString("   - **CrashLoopBackOff**: Fix application startup issues\n")
	sb.WriteString("   - **Liveness probe failures**: Adjust probe timing or fix health checks\n\n")

	sb.WriteString("3. **Set Proper Resource Limits**:\n")
	sb.WriteString("   - Ensure memory and CPU limits are appropriate\n")
	sb.WriteString("   - Use VPA to get right-sizing recommendations\n\n")

	sb.WriteString("4. **Implement Monitoring**:\n")
	sb.WriteString("   - Set up alerts for high restart rates\n")
	sb.WriteString("   - Track restart trends over time\n")
	sb.WriteString("   - Monitor resource usage patterns\n\n")

	return sb.String()
}

func generateNamespaceAnalysisSection(analysis *Analysis) string {
	var sb strings.Builder

	sb.WriteString("## 10. Namespace-by-Namespace Analysis\n\n")

	if len(analysis.NamespaceAnalysis) == 0 {
		sb.WriteString("ℹ️ No application namespaces found (looking for 3-letter codes).\n\n")
		return sb.String()
	}

	// Group by risk level
	critical := []NamespaceAnalysis{}
	high := []NamespaceAnalysis{}
	medium := []NamespaceAnalysis{}
	low := []NamespaceAnalysis{}

	for _, ns := range analysis.NamespaceAnalysis {
		switch ns.RiskLevel {
		case "critical":
			critical = append(critical, ns)
		case "high":
			high = append(high, ns)
		case "medium":
			medium = append(medium, ns)
		case "low":
			low = append(low, ns)
		}
	}

	// Critical Risk Namespaces
	if len(critical) > 0 {
		sb.WriteString("### 🔴 Critical Risk Namespaces\n\n")
		for _, ns := range critical {
			sb.WriteString(generateNamespaceDetail(ns))
		}
	}

	// High Risk Namespaces
	if len(high) > 0 {
		sb.WriteString("### 🟠 High Risk Namespaces\n\n")
		for _, ns := range high {
			sb.WriteString(generateNamespaceDetail(ns))
		}
	}

	// Medium Risk Namespaces
	if len(medium) > 0 {
		sb.WriteString("### 🟡 Medium Risk Namespaces\n\n")
		for _, ns := range medium {
			sb.WriteString(generateNamespaceDetail(ns))
		}
	}

	// Low Risk Namespaces
	if len(low) > 0 {
		sb.WriteString("### 🟢 Low Risk Namespaces\n\n")
		for _, ns := range low {
			sb.WriteString(generateNamespaceDetail(ns))
		}
	}

	sb.WriteString("### Namespace-Level Recommendations\n\n")
	sb.WriteString("1. **Implement LimitRange defaults**:\n")
	sb.WriteString("```yaml\n")
	sb.WriteString("apiVersion: v1\n")
	sb.WriteString("kind: LimitRange\n")
	sb.WriteString("metadata:\n")
	sb.WriteString("  name: default-limits\n")
	sb.WriteString("  namespace: <namespace>\n")
	sb.WriteString("spec:\n")
	sb.WriteString("  limits:\n")
	sb.WriteString("  - default:\n")
	sb.WriteString("      memory: 512Mi\n")
	sb.WriteString("      cpu: 500m\n")
	sb.WriteString("    defaultRequest:\n")
	sb.WriteString("      memory: 256Mi\n")
	sb.WriteString("      cpu: 100m\n")
	sb.WriteString("    type: Container\n")
	sb.WriteString("```\n\n")

	sb.WriteString("2. **Set ResourceQuota per namespace**:\n")
	sb.WriteString("```yaml\n")
	sb.WriteString("apiVersion: v1\n")
	sb.WriteString("kind: ResourceQuota\n")
	sb.WriteString("metadata:\n")
	sb.WriteString("  name: namespace-quota\n")
	sb.WriteString("  namespace: <namespace>\n")
	sb.WriteString("spec:\n")
	sb.WriteString("  hard:\n")
	sb.WriteString("    requests.cpu: \"10\"\n")
	sb.WriteString("    requests.memory: 20Gi\n")
	sb.WriteString("    limits.cpu: \"20\"\n")
	sb.WriteString("    limits.memory: 40Gi\n")
	sb.WriteString("```\n\n")

	return sb.String()
}

func generateNamespaceDetail(ns NamespaceAnalysis) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("#### Namespace: `%s`\n\n", ns.Namespace))
	sb.WriteString("| Metric | Value |\n")
	sb.WriteString("|--------|-------|\n")
	sb.WriteString(fmt.Sprintf("| Total Pods | %d |\n", ns.TotalPods))
	sb.WriteString(fmt.Sprintf("| Pods Missing Requests | %d |\n", ns.PodsWithoutRequests))
	sb.WriteString(fmt.Sprintf("| Pods Missing Limits | %d |\n", ns.PodsWithoutLimits))
	sb.WriteString(fmt.Sprintf("| Risk Level | %s |\n\n", strings.ToUpper(ns.RiskLevel)))

	if len(ns.CriticalPods) > 0 {
		sb.WriteString("**Critical Pods Missing Resources**:\n\n")
		for i, pod := range ns.CriticalPods {
			if i >= 5 {
				sb.WriteString(fmt.Sprintf("\n_... and %d more pods_\n\n", len(ns.CriticalPods)-5))
				break
			}
			sb.WriteString(fmt.Sprintf("- `%s`\n", pod))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("**Recommended Actions**:\n")
	percentage := float64(ns.PodsWithoutRequests) / float64(ns.TotalPods) * 100
	sb.WriteString(fmt.Sprintf("- Priority: %s (%.1f%% pods affected)\n", strings.ToUpper(ns.RiskLevel), percentage))
	sb.WriteString("- Implement LimitRange to set defaults for new pods\n")
	sb.WriteString("- Update existing deployments with appropriate resource requests/limits\n")
	sb.WriteString("- Monitor resource usage patterns for 1-2 weeks before setting permanent values\n\n")

	return sb.String()
}

func generateAIInsightsSection(insights *AIInsights) string {
	var sb strings.Builder

	sb.WriteString("## 11. AI-Enhanced Insights\n\n")
	sb.WriteString(insights.Summary)
	sb.WriteString("\n\n")

	if len(insights.EnhancedRecommendations) > 0 {
		sb.WriteString("### Enhanced Recommendations\n\n")
		for _, rec := range insights.EnhancedRecommendations {
			sb.WriteString(fmt.Sprintf("- %s\n", rec))
		}
		sb.WriteString("\n")
	}

	if insights.RiskAssessment != "" {
		sb.WriteString("### Risk Assessment\n\n")
		sb.WriteString(insights.RiskAssessment)
		sb.WriteString("\n\n")
	}

	if len(insights.AutomationSuggestions) > 0 {
		sb.WriteString("### Automation Suggestions\n\n")
		for _, suggestion := range insights.AutomationSuggestions {
			sb.WriteString(fmt.Sprintf("- %s\n", suggestion))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func generateAppendix(data *ClusterData, analysis *Analysis) string {
	var sb strings.Builder

	sb.WriteString("## Appendix\n\n")

	sb.WriteString("### A. Data Collection Summary\n\n")
	sb.WriteString(fmt.Sprintf("- **Collection Time**: %s\n", time.Now().Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("- **Total Pods Analyzed**: %d\n", len(data.Pods)))
	sb.WriteString(fmt.Sprintf("- **Total Nodes Analyzed**: %d\n", len(data.Nodes)))
	sb.WriteString(fmt.Sprintf("- **Events Processed**: %d\n", len(data.Events)))

	metricsAvailable := len(data.PodMetrics) > 0
	if metricsAvailable {
		sb.WriteString(fmt.Sprintf("- **Metrics Available**: ✅ Yes (%d pods with metrics)\n\n", len(data.PodMetrics)))
	} else {
		sb.WriteString("- **Metrics Available**: ⚠️ No (metrics-server not found or unavailable)\n\n")
	}

	sb.WriteString("### B. All Active Pods - Resource Configuration\n\n")
	sb.WriteString("Complete inventory of all running pods with their resource requests, limits, and current usage.\n")
	if !metricsAvailable {
		sb.WriteString("\n⚠️ **Note**: Current CPU/Memory usage shows 'N/A' because metrics-server is not available. Install metrics-server to see real-time usage data.\n")
	}
	sb.WriteString("\n")

	// Collect pod resource information
	podInfos := []PodResourceInfo{}
	for _, pod := range data.Pods {
		if pod.Status.Phase != corev1.PodRunning {
			continue
		}
		for _, container := range pod.Spec.Containers {
			podInfo := PodResourceInfo{
				Namespace:     pod.Namespace,
				PodName:       pod.Name,
				ContainerName: container.Name,
				Status:        string(pod.Status.Phase),
			}

			// Get configured requests and limits
			if container.Resources.Requests != nil {
				if cpu, ok := container.Resources.Requests[corev1.ResourceCPU]; ok {
					podInfo.CPURequest = cpu.String()
				} else {
					podInfo.CPURequest = "Not Set"
				}
				if mem, ok := container.Resources.Requests[corev1.ResourceMemory]; ok {
					podInfo.MemoryRequest = mem.String()
				} else {
					podInfo.MemoryRequest = "Not Set"
				}
			} else {
				podInfo.CPURequest = "Not Set"
				podInfo.MemoryRequest = "Not Set"
			}

			if container.Resources.Limits != nil {
				if cpu, ok := container.Resources.Limits[corev1.ResourceCPU]; ok {
					podInfo.CPULimit = cpu.String()
				} else {
					podInfo.CPULimit = "Not Set"
				}
				if mem, ok := container.Resources.Limits[corev1.ResourceMemory]; ok {
					podInfo.MemoryLimit = mem.String()
				} else {
					podInfo.MemoryLimit = "Not Set"
				}
			} else {
				podInfo.CPULimit = "Not Set"
				podInfo.MemoryLimit = "Not Set"
			}

			// Get actual usage from metrics if available
			podKey := pod.Namespace + "/" + pod.Name
			if podMetrics, ok := data.PodMetrics[podKey]; ok {
				if containerMetrics, ok := podMetrics.Containers[container.Name]; ok {
					if containerMetrics.CPUUsage != "" {
						podInfo.CurrentCPU = convertCPUToMillicores(containerMetrics.CPUUsage)
					} else {
						podInfo.CurrentCPU = "N/A"
					}
					if containerMetrics.MemoryUsage != "" {
						podInfo.CurrentMemory = convertMemoryToMi(containerMetrics.MemoryUsage)
					} else {
						podInfo.CurrentMemory = "N/A"
					}
				} else {
					podInfo.CurrentCPU = "N/A"
					podInfo.CurrentMemory = "N/A"
				}
			} else {
				podInfo.CurrentCPU = "N/A"
				podInfo.CurrentMemory = "N/A"
			}

			podInfos = append(podInfos, podInfo)
		}
	} // Sort by namespace, then pod name, then container name
	sort.Slice(podInfos, func(i, j int) bool {
		if podInfos[i].Namespace != podInfos[j].Namespace {
			return podInfos[i].Namespace < podInfos[j].Namespace
		}
		if podInfos[i].PodName != podInfos[j].PodName {
			return podInfos[i].PodName < podInfos[j].PodName
		}
		return podInfos[i].ContainerName < podInfos[j].ContainerName
	})

	// Group by namespace
	namespaceGroups := make(map[string][]PodResourceInfo)
	for _, podInfo := range podInfos {
		namespaceGroups[podInfo.Namespace] = append(namespaceGroups[podInfo.Namespace], podInfo)
	}

	// Get sorted namespace list
	namespaces := []string{}
	for ns := range namespaceGroups {
		namespaces = append(namespaces, ns)
	}
	sort.Strings(namespaces)

	// Generate tables by namespace
	for _, ns := range namespaces {
		pods := namespaceGroups[ns]
		sb.WriteString(fmt.Sprintf("#### Namespace: `%s` (%d containers)\n\n", ns, len(pods)))

		// Check if we have AI suggestions for this namespace
		nsSuggestions, hasAISuggestions := data.AISuggestions[ns]
		if hasAISuggestions && len(nsSuggestions) > 0 {
			sb.WriteString(fmt.Sprintf("🤖 **AI Resource Suggestions Available** - %d pods with missing resources analyzed\n\n", len(nsSuggestions)))
		}

		sb.WriteString("| Pod | Container | CPU Req | CPU Limit | CPU Usage | Mem Req | Mem Limit | Mem Usage | Status |\n")
		sb.WriteString("|-----|-----------|---------|-----------|-----------|---------|-----------|-----------|--------|\n")

		for _, pod := range pods {
			// Check if we have AI suggestion for this pod/container
			suggestionKey := pod.PodName + "/" + pod.ContainerName
			suggestion, hasSuggestion := nsSuggestions[suggestionKey]

			cpuReq := pod.CPURequest
			cpuLim := pod.CPULimit
			memReq := pod.MemoryRequest
			memLim := pod.MemoryLimit

			// Apply AI suggestions with green highlighting (using HTML/Markdown color)
			if hasSuggestion {
				if suggestion.CPURequest != "KEEP" && suggestion.CPURequest != "" && pod.CPURequest == "Not Set" {
					cpuReq = fmt.Sprintf("🟢 **`%s`**", suggestion.CPURequest)
				}
				if suggestion.CPULimit != "KEEP" && suggestion.CPULimit != "" && pod.CPULimit == "Not Set" {
					cpuLim = fmt.Sprintf("🟢 **`%s`**", suggestion.CPULimit)
				}
				if suggestion.MemoryRequest != "KEEP" && suggestion.MemoryRequest != "" && pod.MemoryRequest == "Not Set" {
					memReq = fmt.Sprintf("🟢 **`%s`**", suggestion.MemoryRequest)
				}
				if suggestion.MemoryLimit != "KEEP" && suggestion.MemoryLimit != "" && pod.MemoryLimit == "Not Set" {
					memLim = fmt.Sprintf("🟢 **`%s`**", suggestion.MemoryLimit)
				}
			}

			sb.WriteString(fmt.Sprintf("| `%s` | `%s` | %s | %s | %s | %s | %s | %s | %s |\n",
				pod.PodName,
				pod.ContainerName,
				cpuReq,
				cpuLim,
				pod.CurrentCPU,
				memReq,
				memLim,
				pod.CurrentMemory,
				pod.Status,
			))
		}
		sb.WriteString("\n")

		// Add AI suggestion note if applicable
		if hasAISuggestions && len(nsSuggestions) > 0 {
			sb.WriteString("**Legend:** 🟢 = AI-suggested values (apply these to pods missing resource configurations)\n\n")
		}
	}

	// Summary statistics
	totalContainers := len(podInfos)
	containersWithoutRequests := 0
	containersWithoutLimits := 0
	containersFullyConfigured := 0

	for _, pod := range podInfos {
		hasRequests := pod.CPURequest != "Not Set" && pod.MemoryRequest != "Not Set"
		hasLimits := pod.CPULimit != "Not Set" && pod.MemoryLimit != "Not Set"

		if !hasRequests {
			containersWithoutRequests++
		}
		if !hasLimits {
			containersWithoutLimits++
		}
		if hasRequests && hasLimits {
			containersFullyConfigured++
		}
	}

	sb.WriteString("#### Resource Configuration Summary\n\n")
	sb.WriteString(fmt.Sprintf("- **Total Running Containers**: %d\n", totalContainers))
	sb.WriteString(fmt.Sprintf("- **Fully Configured** (requests + limits): %d (%.1f%%)\n",
		containersFullyConfigured, float64(containersFullyConfigured)/float64(totalContainers)*100))
	sb.WriteString(fmt.Sprintf("- **Missing Requests**: %d (%.1f%%)\n",
		containersWithoutRequests, float64(containersWithoutRequests)/float64(totalContainers)*100))
	sb.WriteString(fmt.Sprintf("- **Missing Limits**: %d (%.1f%%)\n\n",
		containersWithoutLimits, float64(containersWithoutLimits)/float64(totalContainers)*100))

	sb.WriteString("### C. Next Steps\n\n")
	sb.WriteString("1. Review critical issues and prioritize based on business impact\n")
	sb.WriteString("2. Implement resource requests/limits for high-risk namespaces first\n")
	sb.WriteString("3. Set up monitoring for OOM events and resource utilization\n")
	sb.WriteString("4. Establish policies (LimitRange, ResourceQuota) to prevent future issues\n")
	sb.WriteString("5. Schedule follow-up analysis after implementing changes\n\n")

	sb.WriteString("### D. Useful Commands\n\n")
	sb.WriteString("**Get pods without resource requests:**\n")
	sb.WriteString("```bash\n")
	sb.WriteString("kubectl get pods -A -o json | jq -r '.items[] | select(.spec.containers[].resources.requests == null) | \"\\(.metadata.namespace)/\\(.metadata.name)\"'\n")
	sb.WriteString("```\n\n")

	sb.WriteString("**Get resource usage for a namespace:**\n")
	sb.WriteString("```bash\n")
	sb.WriteString("kubectl top pods -n <namespace>\n")
	sb.WriteString("```\n\n")

	sb.WriteString("**View pod resource configuration:**\n")
	sb.WriteString("```bash\n")
	sb.WriteString("kubectl get pod <pod-name> -n <namespace> -o jsonpath='{.spec.containers[*].resources}'\n")
	sb.WriteString("```\n\n")

	sb.WriteString("### E. Resources\n\n")
	sb.WriteString("- [Kubernetes Best Practices - Resource Management](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/)\n")
	sb.WriteString("- [Pod Priority and Preemption](https://kubernetes.io/docs/concepts/scheduling-eviction/pod-priority-preemption/)\n")
	sb.WriteString("- [Pod Disruption Budgets](https://kubernetes.io/docs/tasks/run-application/configure-pdb/)\n")
	sb.WriteString("- [Vertical Pod Autoscaler](https://github.com/kubernetes/autoscaler/tree/master/vertical-pod-autoscaler)\n")
	sb.WriteString("- [LimitRange Documentation](https://kubernetes.io/docs/concepts/policy/limit-range/)\n")
	sb.WriteString("- [ResourceQuota Documentation](https://kubernetes.io/docs/concepts/policy/resource-quotas/)\n\n")

	return sb.String()
}

func generateActionItems(issue CriticalIssue) string {
	var sb strings.Builder

	switch issue.Title {
	case "Missing Resource Requests and Limits":
		sb.WriteString("1. Audit all pods using: `kubectl get pods --all-namespaces -o json | jq '.items[] | select(.spec.containers[].resources.requests == null)'`\n")
		sb.WriteString("2. Implement LimitRange in each namespace\n")
		sb.WriteString("3. Update deployment manifests with appropriate resource values\n")
		sb.WriteString("4. Use Vertical Pod Autoscaler to recommend resource values\n")

	case "OOMKilled Events Detected":
		sb.WriteString("1. Identify affected pods from the events list\n")
		sb.WriteString("2. Increase memory limits by 50-100% initially\n")
		sb.WriteString("3. Monitor memory usage patterns using metrics server or Prometheus\n")
		sb.WriteString("4. Investigate potential memory leaks in applications\n")

	case "High Node Resource Utilization":
		sb.WriteString("1. Review cluster autoscaler configuration\n")
		sb.WriteString("2. Add nodes to the cluster or scale up node pools\n")
		sb.WriteString("3. Implement pod affinity/anti-affinity for better distribution\n")
		sb.WriteString("4. Consider using multiple node pools for workload isolation\n")

	default:
		sb.WriteString("1. Review the issue description and impact\n")
		sb.WriteString("2. Follow the specific recommendations provided\n")
		sb.WriteString("3. Test changes in a non-production environment first\n")
		sb.WriteString("4. Monitor the cluster after implementing changes\n")
	}

	return sb.String()
}

func countHighRiskNamespaces(namespaces []NamespaceAnalysis) int {
	count := 0
	for _, ns := range namespaces {
		if ns.RiskLevel == "critical" || ns.RiskLevel == "high" {
			count++
		}
	}
	return count
}

// convertCPUToMillicores converts CPU from various formats to millicores (m)
func convertCPUToMillicores(cpu string) string {
	if cpu == "" || cpu == "N/A" || cpu == "Not Set" {
		return cpu
	}

	// Parse the quantity
	if strings.HasSuffix(cpu, "n") {
		// Nanocores to millicores: divide by 1,000,000
		valueStr := strings.TrimSuffix(cpu, "n")
		if value, err := parseFloat(valueStr); err == nil {
			millicores := value / 1000000.0
			return fmt.Sprintf("%.0fm", millicores)
		}
	} else if strings.HasSuffix(cpu, "u") {
		// Microcores to millicores: divide by 1,000
		valueStr := strings.TrimSuffix(cpu, "u")
		if value, err := parseFloat(valueStr); err == nil {
			millicores := value / 1000.0
			return fmt.Sprintf("%.0fm", millicores)
		}
	}

	// Already in a good format (m, or whole number)
	return cpu
}

// convertMemoryToMi converts memory from various formats to Mi
func convertMemoryToMi(mem string) string {
	if mem == "" || mem == "N/A" || mem == "Not Set" {
		return mem
	}

	// Parse the quantity
	if strings.HasSuffix(mem, "Ki") {
		// Kibibytes to Mebibytes: divide by 1024
		valueStr := strings.TrimSuffix(mem, "Ki")
		if value, err := parseFloat(valueStr); err == nil {
			mebibytes := value / 1024.0
			return fmt.Sprintf("%.0fMi", mebibytes)
		}
	} else if strings.HasSuffix(mem, "Gi") {
		// Gibibytes to Mebibytes: multiply by 1024
		valueStr := strings.TrimSuffix(mem, "Gi")
		if value, err := parseFloat(valueStr); err == nil {
			mebibytes := value * 1024.0
			return fmt.Sprintf("%.0fMi", mebibytes)
		}
	} else if strings.HasSuffix(mem, "Ti") {
		// Tebibytes to Mebibytes: multiply by 1024*1024
		valueStr := strings.TrimSuffix(mem, "Ti")
		if value, err := parseFloat(valueStr); err == nil {
			mebibytes := value * 1024.0 * 1024.0
			return fmt.Sprintf("%.0fMi", mebibytes)
		}
	}

	// Already in Mi or other format
	return mem
}

// parseFloat is a helper to parse float values
func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}
