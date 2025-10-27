package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

type Analyzer struct {
	clientset     *kubernetes.Clientset
	dynamicClient dynamic.Interface
}

type PodMetrics struct {
	Containers map[string]ContainerMetrics // containerName -> metrics
}

type ContainerMetrics struct {
	CPUUsage    string
	MemoryUsage string
}

type ClusterData struct {
	ClusterName   string
	Pods          []corev1.Pod
	Nodes         []corev1.Node
	Events        []corev1.Event
	Namespaces    []corev1.Namespace
	VeleroBackups []unstructured.Unstructured
	PodMetrics    map[string]PodMetrics                    // namespace/podname -> metrics
	AISuggestions map[string]map[string]ResourceSuggestion // namespace -> pod/container -> suggestion
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
	PodRestarts       PodRestartAnalysis
	FluxEvents        FluxEventAnalysis
	NonFluxEvents     NonFluxEventAnalysis
	VeleroBackups     VeleroBackupAnalysis
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

type PodRestart struct {
	Namespace       string
	PodName         string
	ContainerName   string
	RestartCount    int32
	LastRestartTime time.Time
	Reason          string
}

type PodRestartAnalysis struct {
	Last24Hours  []PodRestart
	Last7Days    []PodRestart
	TotalPods24h int
	TotalPods7d  int
}

type PodResourceInfo struct {
	Namespace     string
	PodName       string
	ContainerName string
	Status        string
	CPURequest    string
	CPULimit      string
	MemoryRequest string
	MemoryLimit   string
	CurrentCPU    string
	CurrentMemory string
}

type EventInfo struct {
	Type           string
	Reason         string
	Message        string
	Namespace      string
	InvolvedObject string
	Count          int32
	FirstTime      time.Time
	LastTime       time.Time
}

type FluxEventAnalysis struct {
	Last24Hours []EventInfo
	Last48Hours []EventInfo
	Warnings24h int
	Warnings48h int
	Errors24h   int
	Errors48h   int
}

type NonFluxEventAnalysis struct {
	Last24Hours []EventInfo
	Last48Hours []EventInfo
	Warnings24h int
	Warnings48h int
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
	Last24Hours      []VeleroBackup
	Last48Hours      []VeleroBackup
	TotalBackups24h  int
	TotalBackups48h  int
	FailedBackups24h int
	FailedBackups48h int
}

func NewAnalyzer(clientset *kubernetes.Clientset, dynamicClient dynamic.Interface) *Analyzer {
	return &Analyzer{
		clientset:     clientset,
		dynamicClient: dynamicClient,
	}
}

func (a *Analyzer) CollectClusterData(ctx context.Context) (*ClusterData, error) {
	data := &ClusterData{
		PodMetrics: make(map[string]PodMetrics),
	}

	// Get cluster name from kubeconfig context or server
	data.ClusterName = a.getClusterName(ctx)

	// Get all pods
	pods, err := a.clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error listing pods: %w", err)
	}
	data.Pods = pods.Items

	// Try to get pod metrics (best effort - metrics-server might not be available)
	a.collectPodMetrics(ctx, data)

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

	// Get Velero backups if available
	if a.dynamicClient != nil {
		veleroGVR := schema.GroupVersionResource{
			Group:    "velero.io",
			Version:  "v1",
			Resource: "backups",
		}

		backups, err := a.dynamicClient.Resource(veleroGVR).Namespace("").List(ctx, metav1.ListOptions{})
		if err == nil && backups != nil {
			data.VeleroBackups = backups.Items
		}
		// Ignore errors - Velero might not be installed
	}

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

	// Analyze pod restarts
	analysis.PodRestarts = a.analyzePodRestarts(data.Pods)

	// Analyze Flux events
	analysis.FluxEvents = a.analyzeFluxEvents(data.Events)

	// Analyze non-Flux events
	analysis.NonFluxEvents = a.analyzeNonFluxEvents(data.Events)

	// Analyze Velero backups
	analysis.VeleroBackups = a.analyzeVeleroBackups(data.VeleroBackups)

	// Generate cluster health summary
	analysis.ClusterHealth = a.generateHealthSummary(analysis)

	// Generate critical issues
	analysis.CriticalIssues = a.generateCriticalIssues(analysis)

	return analysis
}

func (a *Analyzer) analyzePodRestarts(pods []corev1.Pod) PodRestartAnalysis {
	analysis := PodRestartAnalysis{
		Last24Hours: []PodRestart{},
		Last7Days:   []PodRestart{},
	}

	now := time.Now()
	threshold24h := now.Add(-24 * time.Hour)
	threshold7d := now.Add(-7 * 24 * time.Hour)

	for _, pod := range pods {
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.RestartCount > 0 {
				var lastRestartTime time.Time
				var reason string

				// Get last restart time and reason
				if containerStatus.LastTerminationState.Terminated != nil {
					lastRestartTime = containerStatus.LastTerminationState.Terminated.FinishedAt.Time
					reason = containerStatus.LastTerminationState.Terminated.Reason
					if reason == "" {
						reason = "Unknown"
					}
				} else if containerStatus.State.Terminated != nil {
					lastRestartTime = containerStatus.State.Terminated.FinishedAt.Time
					reason = containerStatus.State.Terminated.Reason
					if reason == "" {
						reason = "Unknown"
					}
				} else {
					// If we can't determine exact time, use pod start time as estimate
					if pod.Status.StartTime != nil {
						lastRestartTime = pod.Status.StartTime.Time
					}
					reason = "Unknown"
				}

				restart := PodRestart{
					Namespace:       pod.Namespace,
					PodName:         pod.Name,
					ContainerName:   containerStatus.Name,
					RestartCount:    containerStatus.RestartCount,
					LastRestartTime: lastRestartTime,
					Reason:          reason,
				}

				// Add to 24h list if within last 24 hours
				if !lastRestartTime.IsZero() && lastRestartTime.After(threshold24h) {
					analysis.Last24Hours = append(analysis.Last24Hours, restart)
				}

				// Add to 7d list if within last 7 days
				if !lastRestartTime.IsZero() && lastRestartTime.After(threshold7d) {
					analysis.Last7Days = append(analysis.Last7Days, restart)
				}
			}
		}
	}

	// Sort by restart count (highest first)
	sort.Slice(analysis.Last24Hours, func(i, j int) bool {
		return analysis.Last24Hours[i].RestartCount > analysis.Last24Hours[j].RestartCount
	})

	sort.Slice(analysis.Last7Days, func(i, j int) bool {
		return analysis.Last7Days[i].RestartCount > analysis.Last7Days[j].RestartCount
	})

	// Count unique pods
	uniquePods24h := make(map[string]bool)
	for _, r := range analysis.Last24Hours {
		uniquePods24h[r.Namespace+"/"+r.PodName] = true
	}
	analysis.TotalPods24h = len(uniquePods24h)

	uniquePods7d := make(map[string]bool)
	for _, r := range analysis.Last7Days {
		uniquePods7d[r.Namespace+"/"+r.PodName] = true
	}
	analysis.TotalPods7d = len(uniquePods7d)

	return analysis
}

func (a *Analyzer) analyzeFluxEvents(events []corev1.Event) FluxEventAnalysis {
	analysis := FluxEventAnalysis{
		Last24Hours: []EventInfo{},
		Last48Hours: []EventInfo{},
	}

	now := time.Now()
	threshold24h := now.Add(-24 * time.Hour)
	threshold48h := now.Add(-48 * time.Hour)

	for _, event := range events {
		// Check if this is a Flux-related event
		isFlux := strings.Contains(strings.ToLower(event.Source.Component), "flux") ||
			strings.Contains(strings.ToLower(event.InvolvedObject.Kind), "kustomization") ||
			strings.Contains(strings.ToLower(event.InvolvedObject.Kind), "helmrelease") ||
			strings.Contains(strings.ToLower(event.InvolvedObject.APIVersion), "fluxcd") ||
			strings.Contains(strings.ToLower(event.InvolvedObject.APIVersion), "toolkit.fluxcd")

		if !isFlux {
			continue
		}

		eventInfo := EventInfo{
			Type:           event.Type,
			Reason:         event.Reason,
			Message:        event.Message,
			Namespace:      event.Namespace,
			InvolvedObject: fmt.Sprintf("%s/%s", event.InvolvedObject.Kind, event.InvolvedObject.Name),
			Count:          event.Count,
			FirstTime:      event.FirstTimestamp.Time,
			LastTime:       event.LastTimestamp.Time,
		}

		// Count warnings and errors
		if event.LastTimestamp.Time.After(threshold24h) {
			analysis.Last24Hours = append(analysis.Last24Hours, eventInfo)
			if event.Type == "Warning" {
				analysis.Warnings24h++
			} else if event.Type == "Error" {
				analysis.Errors24h++
			}
		}

		if event.LastTimestamp.Time.After(threshold48h) {
			analysis.Last48Hours = append(analysis.Last48Hours, eventInfo)
			if event.Type == "Warning" {
				analysis.Warnings48h++
			} else if event.Type == "Error" {
				analysis.Errors48h++
			}
		}
	}

	// Sort by last time (most recent first)
	sort.Slice(analysis.Last24Hours, func(i, j int) bool {
		return analysis.Last24Hours[i].LastTime.After(analysis.Last24Hours[j].LastTime)
	})

	sort.Slice(analysis.Last48Hours, func(i, j int) bool {
		return analysis.Last48Hours[i].LastTime.After(analysis.Last48Hours[j].LastTime)
	})

	return analysis
}

func (a *Analyzer) analyzeNonFluxEvents(events []corev1.Event) NonFluxEventAnalysis {
	analysis := NonFluxEventAnalysis{
		Last24Hours: []EventInfo{},
		Last48Hours: []EventInfo{},
	}

	now := time.Now()
	threshold24h := now.Add(-24 * time.Hour)
	threshold48h := now.Add(-48 * time.Hour)

	for _, event := range events {
		// Skip Flux events
		isFlux := strings.Contains(strings.ToLower(event.Source.Component), "flux") ||
			strings.Contains(strings.ToLower(event.InvolvedObject.Kind), "kustomization") ||
			strings.Contains(strings.ToLower(event.InvolvedObject.Kind), "helmrelease") ||
			strings.Contains(strings.ToLower(event.InvolvedObject.APIVersion), "fluxcd") ||
			strings.Contains(strings.ToLower(event.InvolvedObject.APIVersion), "toolkit.fluxcd")

		if isFlux {
			continue
		}

		// Only include warnings
		if event.Type != "Warning" {
			continue
		}

		eventInfo := EventInfo{
			Type:           event.Type,
			Reason:         event.Reason,
			Message:        event.Message,
			Namespace:      event.Namespace,
			InvolvedObject: fmt.Sprintf("%s/%s", event.InvolvedObject.Kind, event.InvolvedObject.Name),
			Count:          event.Count,
			FirstTime:      event.FirstTimestamp.Time,
			LastTime:       event.LastTimestamp.Time,
		}

		if event.LastTimestamp.Time.After(threshold24h) {
			analysis.Last24Hours = append(analysis.Last24Hours, eventInfo)
			analysis.Warnings24h++
		}

		if event.LastTimestamp.Time.After(threshold48h) {
			analysis.Last48Hours = append(analysis.Last48Hours, eventInfo)
			analysis.Warnings48h++
		}
	}

	// Sort by last time (most recent first)
	sort.Slice(analysis.Last24Hours, func(i, j int) bool {
		return analysis.Last24Hours[i].LastTime.After(analysis.Last24Hours[j].LastTime)
	})

	sort.Slice(analysis.Last48Hours, func(i, j int) bool {
		return analysis.Last48Hours[i].LastTime.After(analysis.Last48Hours[j].LastTime)
	})

	return analysis
}

func (a *Analyzer) analyzeVeleroBackups(backups []unstructured.Unstructured) VeleroBackupAnalysis {
	analysis := VeleroBackupAnalysis{
		Last24Hours: []VeleroBackup{},
		Last48Hours: []VeleroBackup{},
	}

	now := time.Now()
	threshold24h := now.Add(-24 * time.Hour)
	threshold48h := now.Add(-48 * time.Hour)

	for _, backup := range backups {
		// Extract backup status
		status, found, err := unstructured.NestedMap(backup.Object, "status")
		if !found || err != nil {
			continue
		}

		// Get start time
		startTimeStr, _, _ := unstructured.NestedString(status, "startTimestamp")
		startTime, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			continue
		}

		// Only include backups from last 48 hours
		if !startTime.After(threshold48h) {
			continue
		}

		// Get completion time
		completionTimeStr, _, _ := unstructured.NestedString(status, "completionTimestamp")
		completionTime, _ := time.Parse(time.RFC3339, completionTimeStr)

		// Calculate duration
		var duration time.Duration
		if !completionTime.IsZero() {
			duration = completionTime.Sub(startTime)
		}

		// Get status phase
		phase, _, _ := unstructured.NestedString(status, "phase")

		// Get errors and warnings
		errors, _, _ := unstructured.NestedInt64(status, "errors")
		warnings, _, _ := unstructured.NestedInt64(status, "warnings")

		veleroBackup := VeleroBackup{
			Name:           backup.GetName(),
			Namespace:      backup.GetNamespace(),
			Status:         phase,
			StartTime:      startTime,
			CompletionTime: completionTime,
			Duration:       duration,
			Errors:         int(errors),
			Warnings:       int(warnings),
		}

		if startTime.After(threshold24h) {
			analysis.Last24Hours = append(analysis.Last24Hours, veleroBackup)
			analysis.TotalBackups24h++
			if phase == "Failed" || phase == "PartiallyFailed" {
				analysis.FailedBackups24h++
			}
		}

		if startTime.After(threshold48h) {
			analysis.Last48Hours = append(analysis.Last48Hours, veleroBackup)
			analysis.TotalBackups48h++
			if phase == "Failed" || phase == "PartiallyFailed" {
				analysis.FailedBackups48h++
			}
		}
	}

	// Sort by start time (most recent first)
	sort.Slice(analysis.Last24Hours, func(i, j int) bool {
		return analysis.Last24Hours[i].StartTime.After(analysis.Last24Hours[j].StartTime)
	})

	sort.Slice(analysis.Last48Hours, func(i, j int) bool {
		return analysis.Last48Hours[i].StartTime.After(analysis.Last48Hours[j].StartTime)
	})

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

func (a *Analyzer) collectPodResourceInfo(ctx context.Context, pods []corev1.Pod) []PodResourceInfo {
	var podInfos []PodResourceInfo

	// Get pod metrics if available (best effort)
	metricsAvailable := false
	podMetrics := make(map[string]map[string]corev1.ResourceList) // namespace/pod -> container -> metrics

	// Try to get metrics from metrics-server
	// This is a best-effort attempt - if metrics-server is not available, we'll just show configured values
	if a.clientset != nil {
		// Note: This would require importing metrics client, for now we'll just show configured values
		// In production, you'd use: metricsv1beta1 "k8s.io/metrics/pkg/client/clientset/versioned"
		metricsAvailable = false
	}

	for _, pod := range pods {
		// Only include running pods
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

			// Get current usage if metrics are available
			if metricsAvailable {
				if podMetric, ok := podMetrics[pod.Namespace+"/"+pod.Name]; ok {
					if containerMetric, ok := podMetric[container.Name]; ok {
						if cpu, ok := containerMetric[corev1.ResourceCPU]; ok {
							podInfo.CurrentCPU = cpu.String()
						}
						if mem, ok := containerMetric[corev1.ResourceMemory]; ok {
							podInfo.CurrentMemory = mem.String()
						}
					}
				}
			} else {
				podInfo.CurrentCPU = "N/A"
				podInfo.CurrentMemory = "N/A"
			}

			podInfos = append(podInfos, podInfo)
		}
	}

	// Sort by namespace, then pod name, then container name
	sort.Slice(podInfos, func(i, j int) bool {
		if podInfos[i].Namespace != podInfos[j].Namespace {
			return podInfos[i].Namespace < podInfos[j].Namespace
		}
		if podInfos[i].PodName != podInfos[j].PodName {
			return podInfos[i].PodName < podInfos[j].PodName
		}
		return podInfos[i].ContainerName < podInfos[j].ContainerName
	})

	return podInfos
}

func (a *Analyzer) getClusterName(ctx context.Context) string {
	// Try to get cluster name from kube-system namespace configmap
	cm, err := a.clientset.CoreV1().ConfigMaps("kube-system").Get(ctx, "cluster-info", metav1.GetOptions{})
	if err == nil && cm.Data != nil {
		if clusterName, ok := cm.Data["cluster-name"]; ok {
			return clusterName
		}
	}

	// Try to get from kube-public namespace
	cm, err = a.clientset.CoreV1().ConfigMaps("kube-public").Get(ctx, "cluster-info", metav1.GetOptions{})
	if err == nil && cm.Data != nil {
		if clusterName, ok := cm.Data["cluster-name"]; ok {
			return clusterName
		}
	}

	// Fallback: try to extract from server URL or use "Unknown"
	// Get the first node and check for labels
	nodes, err := a.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{Limit: 1})
	if err == nil && len(nodes.Items) > 0 {
		node := nodes.Items[0]
		// Check common labels for cluster name
		if name, ok := node.Labels["cluster-name"]; ok {
			return name
		}
		if name, ok := node.Labels["alpha.eksctl.io/cluster-name"]; ok {
			return name
		}
		if name, ok := node.Labels["kubernetes.azure.com/cluster"]; ok {
			return name
		}
		// AKS cluster name from node resource group
		if rg, ok := node.Labels["kubernetes.azure.com/node-pool-name"]; ok {
			return "AKS-" + rg
		}
	}

	return "Unknown Cluster"
}

func (a *Analyzer) collectPodMetrics(ctx context.Context, data *ClusterData) {
	// Try to get metrics from metrics-server using dynamic client
	metricsGVR := schema.GroupVersionResource{
		Group:    "metrics.k8s.io",
		Version:  "v1beta1",
		Resource: "pods",
	}

	metricsList, err := a.dynamicClient.Resource(metricsGVR).Namespace("").List(ctx, metav1.ListOptions{})
	if err != nil {
		// Metrics-server not available or error fetching metrics
		return
	}

	// Parse metrics
	for _, item := range metricsList.Items {
		namespace, _, _ := unstructured.NestedString(item.Object, "metadata", "namespace")
		name, _, _ := unstructured.NestedString(item.Object, "metadata", "name")

		containers, found, _ := unstructured.NestedSlice(item.Object, "containers")
		if !found {
			continue
		}

		podMetrics := PodMetrics{
			Containers: make(map[string]ContainerMetrics),
		}

		for _, c := range containers {
			container, ok := c.(map[string]interface{})
			if !ok {
				continue
			}

			containerName, _, _ := unstructured.NestedString(container, "name")
			usage, found, _ := unstructured.NestedMap(container, "usage")
			if !found {
				continue
			}

			metrics := ContainerMetrics{}
			if cpu, ok := usage["cpu"].(string); ok {
				metrics.CPUUsage = cpu
			}
			if mem, ok := usage["memory"].(string); ok {
				metrics.MemoryUsage = mem
			}

			podMetrics.Containers[containerName] = metrics
		}

		key := namespace + "/" + name
		data.PodMetrics[key] = podMetrics
	}
}
