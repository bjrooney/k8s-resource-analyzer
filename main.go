package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

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

	ctx := context.Background()

	fmt.Println("🔍 Analyzing Kubernetes cluster...")

	// Initialize analyzer
	analyzer := NewAnalyzer(clientset)

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
		aiClient, err = NewAIClient(apiKey, *aiProvider, *aiEndpoint)
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
	}

	// Generate report
	fmt.Println("📝 Generating report...")
	report := GenerateReport(data, analysis)

	// Write report to file
	err = os.WriteFile(*outputFile, []byte(report), 0644)
	if err != nil {
		log.Fatalf("Error writing report: %v", err)
	}

	fmt.Printf("✨ Analysis complete! Report saved to: %s\n", *outputFile)
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
