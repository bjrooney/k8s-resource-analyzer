package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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

	// Note: Output is now always generated in a directory structure
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

	// Generate output directory and filename based on cluster name and timestamp
	timestamp := time.Now().Format("20060102-150405")
	sanitizedClusterName := strings.ReplaceAll(data.ClusterName, "/", "-")
	sanitizedClusterName = strings.ReplaceAll(sanitizedClusterName, ":", "-")
	baseFilename := fmt.Sprintf("%s-%s", sanitizedClusterName, timestamp)

	// Create output directory
	outputDir := baseFilename
	err = os.MkdirAll(outputDir, 0755)
	if err != nil {
		log.Fatalf("Error creating output directory: %v", err)
	}

	// Generate report
	fmt.Println("📝 Generating report...")
	report := GenerateReport(data, analysis)

	// Write Markdown file
	mdFile := filepath.Join(outputDir, baseFilename+".md")
	err = os.WriteFile(mdFile, []byte(report), 0644)
	if err != nil {
		log.Fatalf("Error writing markdown report: %v", err)
	}
	fmt.Printf("✅ Markdown report saved to: %s\n", mdFile)

	// Generate HTML file
	htmlFile := filepath.Join(outputDir, baseFilename+".html")
	err = generateHTMLReport(report, htmlFile, baseFilename)
	if err != nil {
		fmt.Printf("⚠️  Warning: Could not generate HTML: %v\n", err)
	} else {
		fmt.Printf("✅ HTML report saved to: %s\n", htmlFile)
	}

	// Generate PDF file using pandoc if available
	pdfFile := filepath.Join(outputDir, baseFilename+".pdf")
	err = generatePDFReport(mdFile, pdfFile)
	if err != nil {
		fmt.Printf("⚠️  Warning: Could not generate PDF: %v\n", err)
		fmt.Println("   (Install pandoc and wkhtmltopdf for PDF generation)")
	} else {
		fmt.Printf("✅ PDF report saved to: %s\n", pdfFile)
	}

	fmt.Printf("\n✨ Analysis complete! Reports saved to directory: %s/\n", outputDir)
}

func generateHTMLReport(markdown string, outputFile string, title string) error {
	// Convert markdown to HTML with custom styling
	htmlContent := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s - Kubernetes Cluster Analysis</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
            background: #f5f5f5;
        }
        .container {
            background: white;
            padding: 40px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        h1 { color: #2c3e50; border-bottom: 3px solid #3498db; padding-bottom: 10px; }
        h2 { color: #34495e; border-bottom: 2px solid #95a5a6; padding-bottom: 8px; margin-top: 30px; }
        h3 { color: #555; margin-top: 20px; }
        pre {
            background: #f8f9fa;
            padding: 15px;
            border-radius: 5px;
            overflow-x: auto;
            border-left: 4px solid #3498db;
        }
        code {
            background: #f8f9fa;
            padding: 2px 6px;
            border-radius: 3px;
            font-family: "Courier New", monospace;
        }
        table {
            border-collapse: collapse;
            width: 100%%;
            margin: 20px 0;
            font-size: 14px;
        }
        th {
            background: #3498db;
            color: white;
            padding: 12px;
            text-align: left;
            font-weight: 600;
        }
        td {
            padding: 10px 12px;
            border-bottom: 1px solid #ddd;
        }
        tr:hover {
            background: #f8f9fa;
        }
        .emoji {
            font-size: 1.2em;
        }
        ul, ol {
            margin: 10px 0;
            padding-left: 30px;
        }
        li {
            margin: 5px 0;
        }
    </style>
</head>
<body>
    <div class="container">
%s
    </div>
</body>
</html>`, title, convertMarkdownToHTML(markdown))

	return os.WriteFile(outputFile, []byte(htmlContent), 0644)
}

func convertMarkdownToHTML(markdown string) string {
	// Basic markdown to HTML conversion
	html := markdown

	// Convert headers
	html = regexp.MustCompile(`(?m)^# (.+)$`).ReplaceAllString(html, "<h1>$1</h1>")
	html = regexp.MustCompile(`(?m)^## (.+)$`).ReplaceAllString(html, "<h2>$1</h2>")
	html = regexp.MustCompile(`(?m)^### (.+)$`).ReplaceAllString(html, "<h3>$1</h3>")
	html = regexp.MustCompile(`(?m)^#### (.+)$`).ReplaceAllString(html, "<h4>$1</h4>")

	// Convert code blocks
	html = regexp.MustCompile("(?s)```([^`]+)```").ReplaceAllString(html, "<pre><code>$1</code></pre>")

	// Convert inline code
	html = regexp.MustCompile("`([^`]+)`").ReplaceAllString(html, "<code>$1</code>")

	// Convert bold
	html = regexp.MustCompile(`\*\*([^*]+)\*\*`).ReplaceAllString(html, "<strong>$1</strong>")

	// Convert italic (be careful not to match **text**)
	html = regexp.MustCompile(`([^*])\*([^*]+)\*([^*])`).ReplaceAllString(html, "$1<em>$2</em>$3")

	// Convert unordered lists - process line by line
	lines := strings.Split(html, "\n")
	var processedLines []string
	inList := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- ") {
			if !inList {
				processedLines = append(processedLines, "<ul>")
				inList = true
			}
			content := strings.TrimPrefix(trimmed, "- ")
			processedLines = append(processedLines, "<li>"+content+"</li>")

			// Check if next line is not a list item
			if i+1 >= len(lines) || !strings.HasPrefix(strings.TrimSpace(lines[i+1]), "- ") {
				processedLines = append(processedLines, "</ul>")
				inList = false
			}
		} else {
			processedLines = append(processedLines, line)
		}
	}
	html = strings.Join(processedLines, "\n")

	// Convert tables (basic support)
	lines = strings.Split(html, "\n")
	var result strings.Builder
	inTable := false

	for i, line := range lines {
		if strings.HasPrefix(line, "|") {
			if !inTable {
				result.WriteString("<table>\n")
				inTable = true
			}

			// Check if it's a separator line
			if strings.Contains(line, "---") {
				continue
			}

			cells := strings.Split(strings.Trim(line, "|"), "|")
			if i > 0 && strings.Contains(lines[i-1], "---") {
				// Regular row
				result.WriteString("<tr>")
				for _, cell := range cells {
					result.WriteString("<td>" + strings.TrimSpace(cell) + "</td>")
				}
				result.WriteString("</tr>\n")
			} else if !strings.Contains(line, "---") {
				// Header row
				result.WriteString("<tr>")
				for _, cell := range cells {
					result.WriteString("<th>" + strings.TrimSpace(cell) + "</th>")
				}
				result.WriteString("</tr>\n")
			}
		} else {
			if inTable {
				result.WriteString("</table>\n")
				inTable = false
			}
			result.WriteString(line + "\n")
		}
	}

	if inTable {
		result.WriteString("</table>\n")
	}

	html = result.String()

	// Convert line breaks to paragraphs
	html = regexp.MustCompile(`\n\n+`).ReplaceAllString(html, "</p><p>")
	html = "<p>" + html + "</p>"

	// Clean up empty paragraphs
	html = regexp.MustCompile(`<p>\s*</p>`).ReplaceAllString(html, "")

	return html
}

func generatePDFReport(mdFile string, pdfFile string) error {
	// Check if pandoc is available
	_, err := exec.LookPath("pandoc")
	if err != nil {
		return fmt.Errorf("pandoc not found in PATH")
	}

	// Use pandoc to convert markdown to PDF
	cmd := exec.Command("pandoc", mdFile, "-o", pdfFile,
		"--pdf-engine=wkhtmltopdf",
		"--variable", "geometry:margin=1in",
		"--variable", "fontsize=10pt")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pandoc conversion failed: %v\nOutput: %s", err, string(output))
	}

	return nil
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
