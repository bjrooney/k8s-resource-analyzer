package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}

	outputFile := flag.String("output", "cluster-analysis-report.md", "output file path for the analysis report")
	aiProvider := flag.String("ai-provider", "openai", "AI provider (openai or azure)")
	aiEndpoint := flag.String("ai-endpoint", "", "AI endpoint URL (for Azure OpenAI)")
	aiModel := flag.String("ai-model", "gpt-4o", "AI model to use (gpt-4o, gpt-4o-mini, gpt-4-turbo, etc.)")
	flag.Parse()

	// Build kubernetes client
	config, err := buildConfig(*kubeconfig)
	if err != nil {
		log.Fatalf("Error building kubeconfig: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error creating kubernetes client: %v", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error creating dynamic client: %v", err)
	}

	ctx := context.Background()

	fmt.Println("🔍 Analyzing Kubernetes cluster...")

	// Initialize analyzer
	analyzer := NewAnalyzer(clientset, dynamicClient)

	// Collect cluster data
	fmt.Println("📊 Collecting cluster data...")
	data, err := analyzer.CollectClusterData(ctx)
	if err != nil {
		log.Fatalf("Error collecting cluster data: %v", err)
	}

	fmt.Printf("✅ Collected data: %d pods, %d nodes, %d events\n",
		len(data.Pods), len(data.Nodes), len(data.Events))

	// Analyze cluster data
	fmt.Println("🔬 Analyzing cluster resources...")
	analysis := analyzer.AnalyzeCluster(data)

	// Get AI API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("AZURE_OPENAI_API_KEY")
	}

	// Initialize AI client
	var aiClient *AIClient
	if apiKey != "" {
		fmt.Println("🤖 Initializing AI analysis...")
		aiClient, err = NewAIClient(apiKey, *aiProvider, *aiEndpoint, *aiModel)
		if err != nil {
			log.Printf("Warning: Could not initialize AI client: %v", err)
		}
	} else {
		log.Println("⚠️  No AI API key found. Skipping AI-enhanced analysis.")
		log.Println("   Set OPENAI_API_KEY or AZURE_OPENAI_API_KEY environment variable to enable AI analysis.")
	}

	// Generate AI insights
	if aiClient != nil {
		fmt.Println("💡 Generating AI insights...")
		aiInsights, err := aiClient.AnalyzeCluster(ctx, data, analysis)
		if err != nil {
			log.Printf("Warning: AI analysis failed: %v", err)
		} else {
			analysis.AIInsights = aiInsights
		}

		// Generate AI resource suggestions for ALL namespaces with missing resources
		fmt.Println("🎯 Generating AI resource suggestions for all namespaces with missing resources...")
		data.AISuggestions = make(map[string]map[string]ResourceSuggestion)

		// Find all namespaces that have pods with missing resources
		namespacesWithMissingResources := make(map[string]bool)
		for _, pod := range data.Pods {
			if pod.Status.Phase != corev1.PodRunning {
				continue
			}
			for _, container := range pod.Spec.Containers {
				hasMissingResources := false
				if container.Resources.Requests == nil {
					hasMissingResources = true
				}
				if container.Resources.Limits == nil {
					hasMissingResources = true
				} else if container.Resources.Requests != nil {
					if _, ok := container.Resources.Requests[corev1.ResourceCPU]; !ok {
						hasMissingResources = true
					}
					if _, ok := container.Resources.Requests[corev1.ResourceMemory]; !ok {
						hasMissingResources = true
					}
					if _, ok := container.Resources.Limits[corev1.ResourceCPU]; !ok {
						hasMissingResources = true
					}
					if _, ok := container.Resources.Limits[corev1.ResourceMemory]; !ok {
						hasMissingResources = true
					}
				}
				if hasMissingResources {
					namespacesWithMissingResources[pod.Namespace] = true
					break
				}
			}
		}

		fmt.Printf("   Found %d namespaces with missing resource configurations\n", len(namespacesWithMissingResources))

		// Generate suggestions for each namespace
		for ns := range namespacesWithMissingResources {
			suggestions, err := aiClient.SuggestResourceLimits(ctx, collectPodResourceInfoForNamespace(data.Pods, data.PodMetrics, ns), ns)
			if err != nil {
				log.Printf("Warning: AI resource suggestion failed for namespace %s: %v", ns, err)
			} else if len(suggestions) > 0 {
				data.AISuggestions[ns] = suggestions
				fmt.Printf("   ✅ Generated suggestions for %d pods in namespace '%s'\n", len(suggestions), ns)
			}
		}
	}

	// Generate output filename based on cluster name and timestamp
	timestamp := time.Now().Format("20060102")
	sanitizedClusterName := strings.ReplaceAll(data.ClusterName, "/", "-")
	sanitizedClusterName = strings.ReplaceAll(sanitizedClusterName, ":", "-")
	autoOutputFile := fmt.Sprintf("%s-%s.md", sanitizedClusterName, timestamp)

	// Use auto-generated filename unless user specified a custom one
	finalOutputFile := *outputFile
	if *outputFile == "cluster-analysis-report.md" {
		finalOutputFile = autoOutputFile
	}

	// Generate report
	fmt.Println("📝 Generating report...")
	report := GenerateReport(data, analysis)

	// Write report to file
	err = os.WriteFile(finalOutputFile, []byte(report), 0644)
	if err != nil {
		log.Fatalf("Error writing report: %v", err)
	}

	fmt.Printf("✨ Analysis complete! Report saved to: %s\n", finalOutputFile)
}

func collectPodResourceInfoForNamespace(pods []corev1.Pod, podMetrics map[string]PodMetrics, namespace string) []PodResourceInfo {
	var podInfos []PodResourceInfo

	for _, pod := range pods {
		if pod.Namespace != namespace || pod.Status.Phase != corev1.PodRunning {
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
			if podMetric, ok := podMetrics[podKey]; ok {
				if containerMetric, ok := podMetric.Containers[container.Name]; ok {
					podInfo.CurrentCPU = containerMetric.CPUUsage
					podInfo.CurrentMemory = containerMetric.MemoryUsage
				}
			}

			if podInfo.CurrentCPU == "" {
				podInfo.CurrentCPU = "N/A"
			}
			if podInfo.CurrentMemory == "" {
				podInfo.CurrentMemory = "N/A"
			}

			podInfos = append(podInfos, podInfo)
		}
	}

	return podInfos
}

func buildConfig(kubeconfig string) (*rest.Config, error) {
	// Try in-cluster config first
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}

	// Fall back to kubeconfig
	if kubeconfig == "" {
		return nil, fmt.Errorf("unable to load in-cluster config and no kubeconfig provided")
	}

	return clientcmd.BuildConfigFromFlags("", kubeconfig)
}
