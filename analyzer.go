package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Analyzer struct {
	clientset *kubernetes.Clientset
}

type ClusterData struct {
	Pods       []corev1.Pod
	Nodes      []corev1.Node
	Events     []corev1.Event
	Namespaces []corev1.Namespace
}

type ResourceGap struct {
	Namespace       string
	PodName         string
	Container       string
	MissingRequests bool
	MissingLimits   bool
}

type NodeIssue struct {
	NodeName          string
	Issue             string
	RequestedCPU      float64
	RequestedMemory   float64
	AllocatableCPU    float64
	AllocatableMemory float64
}

type NamespaceAnalysis struct {
	Namespace           string
	TotalPods           int
	PodsWithoutRequests int
	PodsWithoutLimits   int
	RiskLevel           string
	CriticalPods        []string
	Recommendations     []string
}

type Analysis struct {
	ClusterHealth     string
	CriticalIssues    []CriticalIssue
	ResourceGaps      []ResourceGap
	NodeIssues        []NodeIssue
	OOMEvents         []OOMEvent
	NamespaceAnalysis []NamespaceAnalysis
	RabbitMQFindings  RabbitMQAnalysis
	ShortLivedJobs    JobAnalysis
	AIInsights        *AIInsights
}

type CriticalIssue struct {
	Priority       int
	Title          string
	Description    string
	Impact         string
	Recommendation string
	Examples       []string
}

type OOMEvent struct {
	NodeName  string
	PodName   string
	Namespace string
	Container string
	Timestamp time.Time
	Reason    string
}

type RabbitMQAnalysis struct {
	RabbitMQPods      []string
	HasPriorityClass  bool
	HasResourceLimits bool
	Recommendations   []string
}

type JobAnalysis struct {
	ShortJobs         int
	TotalJobs         int
	ImpactOnStability string
	Recommendations   []string
}

func NewAnalyzer(clientset *kubernetes.Clientset) *Analyzer {
	return &Analyzer{clientset: clientset}
}

func (a *Analyzer) CollectClusterData(ctx context.Context) (*ClusterData, error) {
	data := &ClusterData{}

	// Get all pods
	pods, err := a.clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error listing pods: %w", err)
	}
	data.Pods = pods.Items

	// Get all nodes
	nodes, err := a.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error listing nodes: %w", err)
	}
	data.Nodes = nodes.Items

	// Get recent events (last 24 hours)
	events, err := a.clientset.CoreV1().Events("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error listing events: %w", err)
	}
	data.Events = events.Items

	// Get namespaces
	namespaces, err := a.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error listing namespaces: %w", err)
	}
	data.Namespaces = namespaces.Items

	return data, nil
}

func (a *Analyzer) AnalyzeCluster(data *ClusterData) *Analysis {
	analysis := &Analysis{}

	// Analyze resource gaps
	analysis.ResourceGaps = a.analyzeResourceGaps(data.Pods)

	// Analyze nodes
	analysis.NodeIssues = a.analyzeNodes(data.Nodes, data.Pods)

	// Analyze OOM events
	analysis.OOMEvents = a.analyzeOOMEvents(data.Events)

	// Analyze namespaces
	analysis.NamespaceAnalysis = a.analyzeNamespaces(data.Pods, data.Namespaces)

	// Analyze RabbitMQ
	analysis.RabbitMQFindings = a.analyzeRabbitMQ(data.Pods)

	// Analyze short-lived jobs
	analysis.ShortLivedJobs = a.analyzeJobs(data.Pods)

	// Generate cluster health summary
	analysis.ClusterHealth = a.generateHealthSummary(analysis)

	// Generate critical issues
	analysis.CriticalIssues = a.generateCriticalIssues(analysis)

	return analysis
}

func (a *Analyzer) analyzeResourceGaps(pods []corev1.Pod) []ResourceGap {
	gaps := []ResourceGap{}

	for _, pod := range pods {
		for _, container := range pod.Spec.Containers {
			gap := ResourceGap{
				Namespace: pod.Namespace,
				PodName:   pod.Name,
				Container: container.Name,
			}

			if container.Resources.Requests == nil ||
				(container.Resources.Requests.Cpu().IsZero() &&
					container.Resources.Requests.Memory().IsZero()) {
				gap.MissingRequests = true
			}

			if container.Resources.Limits == nil ||
				(container.Resources.Limits.Cpu().IsZero() &&
					container.Resources.Limits.Memory().IsZero()) {
				gap.MissingLimits = true
			}

			if gap.MissingRequests || gap.MissingLimits {
				gaps = append(gaps, gap)
			}
		}
	}

	return gaps
}

func (a *Analyzer) analyzeNodes(nodes []corev1.Node, pods []corev1.Pod) []NodeIssue {
	issues := []NodeIssue{}

	for _, node := range nodes {
		// Calculate resource usage on this node
		var requestedCPU, requestedMemory float64

		for _, pod := range pods {
			if pod.Spec.NodeName == node.Name {
				for _, container := range pod.Spec.Containers {
					requestedCPU += float64(container.Resources.Requests.Cpu().MilliValue())
					requestedMemory += float64(container.Resources.Requests.Memory().Value())
				}
			}
		}

		allocatableCPU := float64(node.Status.Allocatable.Cpu().MilliValue())
		allocatableMemory := float64(node.Status.Allocatable.Memory().Value())

		cpuPercent := (requestedCPU / allocatableCPU) * 100
		memPercent := (requestedMemory / allocatableMemory) * 100

		if cpuPercent > 80 {
			issues = append(issues, NodeIssue{
				NodeName:          node.Name,
				Issue:             "High CPU requests",
				RequestedCPU:      requestedCPU / 1000,
				RequestedMemory:   requestedMemory / (1024 * 1024 * 1024),
				AllocatableCPU:    allocatableCPU / 1000,
				AllocatableMemory: allocatableMemory / (1024 * 1024 * 1024),
			})
		}

		if memPercent > 80 {
			issues = append(issues, NodeIssue{
				NodeName:          node.Name,
				Issue:             "High memory requests",
				RequestedCPU:      requestedCPU / 1000,
				RequestedMemory:   requestedMemory / (1024 * 1024 * 1024),
				AllocatableCPU:    allocatableCPU / 1000,
				AllocatableMemory: allocatableMemory / (1024 * 1024 * 1024),
			})
		}
	}

	return issues
}

func (a *Analyzer) analyzeOOMEvents(events []corev1.Event) []OOMEvent {
	oomEvents := []OOMEvent{}

	for _, event := range events {
		if strings.Contains(event.Reason, "OOMKilled") ||
			strings.Contains(event.Message, "OOMKilled") {
			oomEvents = append(oomEvents, OOMEvent{
				NodeName:  event.Source.Host,
				PodName:   event.InvolvedObject.Name,
				Namespace: event.InvolvedObject.Namespace,
				Container: event.InvolvedObject.FieldPath,
				Timestamp: event.LastTimestamp.Time,
				Reason:    event.Reason,
			})
		}
	}

	// Sort by timestamp, most recent first
	sort.Slice(oomEvents, func(i, j int) bool {
		return oomEvents[i].Timestamp.After(oomEvents[j].Timestamp)
	})

	return oomEvents
}

func (a *Analyzer) analyzeNamespaces(pods []corev1.Pod, namespaces []corev1.Namespace) []NamespaceAnalysis {
	nsMap := make(map[string]*NamespaceAnalysis)

	// Filter application namespaces (3-letter codes)
	appNamespaces := []string{}
	for _, ns := range namespaces {
		if len(ns.Name) == 3 && !strings.HasPrefix(ns.Name, "kube-") {
			appNamespaces = append(appNamespaces, ns.Name)
			nsMap[ns.Name] = &NamespaceAnalysis{
				Namespace:       ns.Name,
				Recommendations: []string{},
				CriticalPods:    []string{},
			}
		}
	}

	// Analyze pods in each namespace
	for _, pod := range pods {
		if nsAnalysis, ok := nsMap[pod.Namespace]; ok {
			nsAnalysis.TotalPods++

			hasRequests := false
			hasLimits := false

			for _, container := range pod.Spec.Containers {
				if container.Resources.Requests != nil &&
					(!container.Resources.Requests.Cpu().IsZero() ||
						!container.Resources.Requests.Memory().IsZero()) {
					hasRequests = true
				}
				if container.Resources.Limits != nil &&
					(!container.Resources.Limits.Cpu().IsZero() ||
						!container.Resources.Limits.Memory().IsZero()) {
					hasLimits = true
				}
			}

			if !hasRequests {
				nsAnalysis.PodsWithoutRequests++
				nsAnalysis.CriticalPods = append(nsAnalysis.CriticalPods, pod.Name)
			}
			if !hasLimits {
				nsAnalysis.PodsWithoutLimits++
			}
		}
	}

	// Calculate risk levels
	result := []NamespaceAnalysis{}
	for _, nsAnalysis := range nsMap {
		if nsAnalysis.TotalPods == 0 {
			continue
		}

		requestGapPercent := float64(nsAnalysis.PodsWithoutRequests) / float64(nsAnalysis.TotalPods) * 100

		if requestGapPercent > 75 {
			nsAnalysis.RiskLevel = "critical"
		} else if requestGapPercent > 50 {
			nsAnalysis.RiskLevel = "high"
		} else if requestGapPercent > 25 {
			nsAnalysis.RiskLevel = "medium"
		} else {
			nsAnalysis.RiskLevel = "low"
		}

		result = append(result, *nsAnalysis)
	}

	// Sort by risk level
	sort.Slice(result, func(i, j int) bool {
		riskOrder := map[string]int{"critical": 0, "high": 1, "medium": 2, "low": 3}
		return riskOrder[result[i].RiskLevel] < riskOrder[result[j].RiskLevel]
	})

	return result
}

func (a *Analyzer) analyzeRabbitMQ(pods []corev1.Pod) RabbitMQAnalysis {
	analysis := RabbitMQAnalysis{
		RabbitMQPods:    []string{},
		Recommendations: []string{},
	}

	for _, pod := range pods {
		if strings.Contains(pod.Name, "rabbitmq") ||
			strings.Contains(strings.ToLower(pod.Name), "rabbit") {
			analysis.RabbitMQPods = append(analysis.RabbitMQPods,
				fmt.Sprintf("%s/%s", pod.Namespace, pod.Name))

			// Check priority class
			if pod.Spec.PriorityClassName != "" {
				analysis.HasPriorityClass = true
			}

			// Check resource limits
			for _, container := range pod.Spec.Containers {
				if container.Resources.Limits != nil &&
					!container.Resources.Limits.Memory().IsZero() {
					analysis.HasResourceLimits = true
					break
				}
			}
		}
	}

	return analysis
}

func (a *Analyzer) analyzeJobs(pods []corev1.Pod) JobAnalysis {
	analysis := JobAnalysis{}

	for _, pod := range pods {
		if pod.OwnerReferences != nil {
			for _, owner := range pod.OwnerReferences {
				if owner.Kind == "Job" {
					analysis.TotalJobs++

					// Check if short-lived (completed in < 2 minutes)
					if pod.Status.Phase == "Succeeded" &&
						pod.Status.StartTime != nil &&
						pod.Status.ContainerStatuses != nil {
						for _, cs := range pod.Status.ContainerStatuses {
							if cs.State.Terminated != nil {
								duration := cs.State.Terminated.FinishedAt.Sub(pod.Status.StartTime.Time)
								if duration < 2*time.Minute {
									analysis.ShortJobs++
									break
								}
							}
						}
					}
				}
			}
		}
	}

	return analysis
}

func (a *Analyzer) generateHealthSummary(analysis *Analysis) string {
	criticalCount := 0
	for _, issue := range analysis.CriticalIssues {
		if issue.Priority <= 2 {
			criticalCount++
		}
	}

	health := "healthy"
	if criticalCount > 3 || len(analysis.OOMEvents) > 10 {
		health = "critical"
	} else if criticalCount > 0 || len(analysis.OOMEvents) > 0 {
		health = "degraded"
	}

	return health
}

func (a *Analyzer) generateCriticalIssues(analysis *Analysis) []CriticalIssue {
	issues := []CriticalIssue{}

	// Issue 1: Missing resource requests/limits
	if len(analysis.ResourceGaps) > 0 {
		examples := []string{}
		for i, gap := range analysis.ResourceGaps {
			if i >= 3 {
				break
			}
			examples = append(examples, fmt.Sprintf("%s/%s (container: %s)",
				gap.Namespace, gap.PodName, gap.Container))
		}

		issues = append(issues, CriticalIssue{
			Priority:       1,
			Title:          "Missing Resource Requests and Limits",
			Description:    fmt.Sprintf("%d containers are missing resource requests or limits", len(analysis.ResourceGaps)),
			Impact:         "Prevents proper scheduling, impacts Velero backups, and can cause cluster instability",
			Recommendation: "Set resource requests and limits for all containers based on observed usage patterns",
			Examples:       examples,
		})
	}

	// Issue 2: OOM events
	if len(analysis.OOMEvents) > 0 {
		examples := []string{}
		for i, event := range analysis.OOMEvents {
			if i >= 3 {
				break
			}
			examples = append(examples, fmt.Sprintf("%s/%s at %s",
				event.Namespace, event.PodName, event.Timestamp.Format(time.RFC3339)))
		}

		issues = append(issues, CriticalIssue{
			Priority:       2,
			Title:          "OOMKilled Events Detected",
			Description:    fmt.Sprintf("%d OOMKilled events found in recent history", len(analysis.OOMEvents)),
			Impact:         "Workload disruptions, data loss, and degraded application performance",
			Recommendation: "Increase memory limits for affected pods or optimize application memory usage",
			Examples:       examples,
		})
	}

	// Issue 3: Node resource pressure
	if len(analysis.NodeIssues) > 0 {
		examples := []string{}
		for i, issue := range analysis.NodeIssues {
			if i >= 3 {
				break
			}
			examples = append(examples, fmt.Sprintf("%s: %s (%.1f%% utilized)",
				issue.NodeName, issue.Issue,
				(issue.RequestedCPU/issue.AllocatableCPU)*100))
		}

		issues = append(issues, CriticalIssue{
			Priority:       3,
			Title:          "High Node Resource Utilization",
			Description:    fmt.Sprintf("%d nodes showing high resource utilization", len(analysis.NodeIssues)),
			Impact:         "Limited scheduling capacity, potential cascading failures during node issues",
			Recommendation: "Scale node pool or rebalance workloads across nodes",
			Examples:       examples,
		})
	}

	return issues
}
