package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/sashabaranov/go-openai"
)

type AIClient struct {
	client   *openai.Client
	provider string
	model    string
}

type AIInsights struct {
	Summary                 string
	EnhancedRecommendations []string
	RiskAssessment          string
	AutomationSuggestions   []string
}

func NewAIClient(apiKey, provider, endpoint, model string) (*AIClient, error) {
	var client *openai.Client

	if provider == "azure" && endpoint != "" {
		config := openai.DefaultAzureConfig(apiKey, endpoint)
		client = openai.NewClientWithConfig(config)
	} else {
		client = openai.NewClient(apiKey)
	}

	// Default to gpt-4o if no model specified
	if model == "" {
		model = "gpt-4o"
	}

	return &AIClient{
		client:   client,
		provider: provider,
		model:    model,
	}, nil
}

func (ai *AIClient) AnalyzeCluster(ctx context.Context, data *ClusterData, analysis *Analysis) (*AIInsights, error) {
	// Build analysis prompt
	prompt := ai.buildAnalysisPrompt(data, analysis)

	// Call OpenAI API
	resp, err := ai.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: ai.model, // Use configured model
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: getSystemPrompt(),
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			Temperature: 0.7,
			MaxTokens:   2000,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("OpenAI API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenAI")
	}

	// Parse AI response
	insights := ai.parseAIResponse(resp.Choices[0].Message.Content)
	return insights, nil
}

func (ai *AIClient) buildAnalysisPrompt(data *ClusterData, analysis *Analysis) string {
	var sb strings.Builder

	sb.WriteString("# Kubernetes Cluster Analysis Data\n\n")

	sb.WriteString(fmt.Sprintf("## Cluster Overview\n"))
	sb.WriteString(fmt.Sprintf("- Total Pods: %d\n", len(data.Pods)))
	sb.WriteString(fmt.Sprintf("- Total Nodes: %d\n", len(data.Nodes)))
	sb.WriteString(fmt.Sprintf("- Health Status: %s\n", analysis.ClusterHealth))
	sb.WriteString(fmt.Sprintf("- OOM Events: %d\n", len(analysis.OOMEvents)))
	sb.WriteString(fmt.Sprintf("- Pods Missing Resources: %d\n\n", len(analysis.ResourceGaps)))

	sb.WriteString("## Critical Issues Detected\n")
	for i, issue := range analysis.CriticalIssues {
		sb.WriteString(fmt.Sprintf("%d. **%s** (Priority %d)\n", i+1, issue.Title, issue.Priority))
		sb.WriteString(fmt.Sprintf("   - Impact: %s\n", issue.Impact))
		sb.WriteString(fmt.Sprintf("   - Current Recommendation: %s\n", issue.Recommendation))
	}
	sb.WriteString("\n")

	sb.WriteString("## Namespace Risk Analysis\n")
	for _, ns := range analysis.NamespaceAnalysis {
		sb.WriteString(fmt.Sprintf("- %s: %s risk (%d/%d pods missing resources)\n",
			ns.Namespace, ns.RiskLevel, ns.PodsWithoutRequests, ns.TotalPods))
	}
	sb.WriteString("\n")

	sb.WriteString("## RabbitMQ Status\n")
	sb.WriteString(fmt.Sprintf("- RabbitMQ Pods Found: %d\n", len(analysis.RabbitMQFindings.RabbitMQPods)))
	sb.WriteString(fmt.Sprintf("- Has Priority Class: %v\n", analysis.RabbitMQFindings.HasPriorityClass))
	sb.WriteString(fmt.Sprintf("- Has Resource Limits: %v\n\n", analysis.RabbitMQFindings.HasResourceLimits))

	sb.WriteString("## Short-Lived Jobs\n")
	sb.WriteString(fmt.Sprintf("- Short Jobs (<2min): %d\n", analysis.ShortLivedJobs.ShortJobs))
	sb.WriteString(fmt.Sprintf("- Total Jobs: %d\n\n", analysis.ShortLivedJobs.TotalJobs))

	sb.WriteString("Please provide:\n")
	sb.WriteString("1. Enhanced insights and strategic recommendations\n")
	sb.WriteString("2. Risk assessment with specific remediation priorities\n")
	sb.WriteString("3. Suggestions for automation and preventive measures\n")

	return sb.String()
}

func (ai *AIClient) parseAIResponse(response string) *AIInsights {
	// Simple parsing - in production, you might want more sophisticated parsing
	return &AIInsights{
		Summary: response,
		EnhancedRecommendations: []string{
			"AI analysis provided in summary section",
		},
		RiskAssessment: "See AI summary for detailed risk assessment",
		AutomationSuggestions: []string{
			"Implement ResourceQuota policies",
			"Set up LimitRange defaults for namespaces",
			"Configure PodDisruptionBudgets for critical workloads",
		},
	}
}

func getSystemPrompt() string {
	return `You are an expert Kubernetes Site Reliability Engineer (SRE) analyzing production cluster data.

## Analysis Requirements:

1. **Cluster Health Summary**: Provide a concise, high-level overview of the cluster's health and identify potential issues.

2. **Critical Issues**: Identify the top 3-5 most critical issues with specific, actionable recommendations and examples.

3. **Resource Management**:
   - Focus on resource gaps (missing requests/limits)
   - Explain how proper requests/limits will benefit Velero, system pods, and cluster stability
   - Analyze the impact of short-lived jobs (1 minute duration) on overall stability

4. **Node Analysis**:
   - Identify poorly balanced node pools and nodes with suboptimal resource allocation
   - Review OOMKilled events and nodes with high resource requests
   - Assess cluster autoscaling settings and potential bottlenecks

5. **RabbitMQ Stability**:
   - Provide specific recommendations to ensure RabbitMQ remains stable
   - Explain how to make RabbitMQ the last workload to be evicted during OOM situations
   - Include priority class and resource allocation strategies

6. **Namespace Analysis**:
   - Analyze each application namespace (typically 3-letter codes like asu, cgh, etc.)
   - For each namespace, identify which pods are missing resource requests/limits
   - Prioritize which pods in each namespace most critically need resource constraints
   - Provide namespace-specific recommendations with suggested resource values based on observed usage patterns
   - Group namespaces by risk level (critical, high, medium, low) based on missing resources

Format your response in clear, well-structured Markdown with sections and code examples where appropriate.`
}
