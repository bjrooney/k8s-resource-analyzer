package main

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

func GenerateReport(data *ClusterData, analysis *Analysis) string {
	var sb strings.Builder

	// Header
	sb.WriteString("# Kubernetes Cluster Analysis Report\n\n")
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

func generateRabbitMQSection(analysis *Analysis) string {
	var sb strings.Builder

	sb.WriteString("## 5. RabbitMQ Stability Analysis\n\n")

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
	total7d := analysis.PodRestarts.TotalPods7d

	if total24h == 0 && total7d == 0 {
		sb.WriteString("✅ No pod restarts detected in the last 7 days.\n\n")
		return sb.String()
	}

	// Summary
	sb.WriteString("### Summary\n\n")
	sb.WriteString(fmt.Sprintf("- **Last 24 Hours**: %d pods with restarts (%d total restarts)\n",
		total24h, len(analysis.PodRestarts.Last24Hours)))
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

	// Last 7 Days (excluding 24h ones already shown)
	if len(analysis.PodRestarts.Last7Days) > len(analysis.PodRestarts.Last24Hours) {
		sb.WriteString("### Additional Restarts in Last 7 Days (excluding above)\n\n")

		// Create map of 24h restarts to exclude
		recent := make(map[string]bool)
		for _, r := range analysis.PodRestarts.Last24Hours {
			recent[r.Namespace+"/"+r.PodName+"/"+r.ContainerName] = true
		}

		// Filter out recent ones
		older := []PodRestart{}
		for _, r := range analysis.PodRestarts.Last7Days {
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

	sb.WriteString("## 7. Namespace-by-Namespace Analysis\n\n")

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

	sb.WriteString("## 8. AI-Enhanced Insights\n\n")
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

	sb.WriteString("### Data Collection Summary\n\n")
	sb.WriteString(fmt.Sprintf("- **Collection Time**: %s\n", time.Now().Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("- **Total Pods Analyzed**: %d\n", len(data.Pods)))
	sb.WriteString(fmt.Sprintf("- **Total Nodes Analyzed**: %d\n", len(data.Nodes)))
	sb.WriteString(fmt.Sprintf("- **Events Processed**: %d\n\n", len(data.Events)))

	sb.WriteString("### Next Steps\n\n")
	sb.WriteString("1. Review critical issues and prioritize based on business impact\n")
	sb.WriteString("2. Implement resource requests/limits for high-risk namespaces first\n")
	sb.WriteString("3. Set up monitoring for OOM events and resource utilization\n")
	sb.WriteString("4. Establish policies (LimitRange, ResourceQuota) to prevent future issues\n")
	sb.WriteString("5. Schedule follow-up analysis after implementing changes\n\n")

	sb.WriteString("### Resources\n\n")
	sb.WriteString("- [Kubernetes Best Practices - Resource Management](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/)\n")
	sb.WriteString("- [Pod Priority and Preemption](https://kubernetes.io/docs/concepts/scheduling-eviction/pod-priority-preemption/)\n")
	sb.WriteString("- [Pod Disruption Budgets](https://kubernetes.io/docs/tasks/run-application/configure-pdb/)\n")
	sb.WriteString("- [Vertical Pod Autoscaler](https://github.com/kubernetes/autoscaler/tree/master/vertical-pod-autoscaler)\n\n")

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
