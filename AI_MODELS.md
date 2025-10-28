# AI Model Guide

## Available Models

The k8s-analyzer now supports configurable AI models. By default, it uses **GPT-4o**.

### Model Options

| Model | Description | Speed | Cost | Best For |
|-------|-------------|-------|------|----------|
| `gpt-4o` | Latest GPT-4 Optimized | Fast | $$ | Production analysis (default) |
| `gpt-4o-mini` | Smaller, faster GPT-4o | Very Fast | $ | Quick checks, dev clusters |
| `gpt-4-turbo` | GPT-4 Turbo | Medium | $$$ | Detailed analysis |
| `gpt-3.5-turbo` | GPT-3.5 | Very Fast | $ | Basic analysis, testing |

## Usage

### Default (GPT-4o)
```bash
export OPENAI_API_KEY="sk-..."
./bin/k8s-analyzer
```

### Use GPT-4o-mini (Faster & Cheaper)
```bash
export OPENAI_API_KEY="sk-..."
./bin/k8s-analyzer -ai-model=gpt-4o-mini
```

### Use GPT-4 Turbo
```bash
export OPENAI_API_KEY="sk-..."
./bin/k8s-analyzer -ai-model=gpt-4-turbo
```

### Use GPT-3.5 Turbo (Budget Option)
```bash
export OPENAI_API_KEY="sk-..."
./bin/k8s-analyzer -ai-model=gpt-3.5-turbo
```

## Azure OpenAI

For Azure OpenAI, specify your deployment name:

```bash
export AZURE_OPENAI_API_KEY="your-key"
./bin/k8s-analyzer \
  -ai-provider=azure \
  -ai-endpoint=https://your-resource.openai.azure.com/ \
  -ai-model=gpt-4o
```

## Troubleshooting

### "Model does not exist" Error

If you see:
```
The model `gpt-4` does not exist or you do not have access to it.
```

**Solutions:**

1. **Use GPT-4o (recommended)**:
   ```bash
   ./bin/k8s-analyzer -ai-model=gpt-4o
   ```

2. **Use GPT-4o-mini** (if you don't have GPT-4o access):
   ```bash
   ./bin/k8s-analyzer -ai-model=gpt-4o-mini
   ```

3. **Use GPT-3.5-turbo** (available to all OpenAI users):
   ```bash
   ./bin/k8s-analyzer -ai-model=gpt-3.5-turbo
   ```

4. **Check your OpenAI account**: Visit https://platform.openai.com/account/limits to see which models you have access to.

### Model Access by API Tier

| Tier | Available Models |
|------|------------------|
| Free Trial | `gpt-3.5-turbo`, `gpt-4o-mini` |
| Tier 1 ($5+ spent) | Above + `gpt-4o` |
| Tier 2+ | Above + `gpt-4-turbo`, etc. |

## Recommendations

### For Production Clusters
Use `gpt-4o` for best quality insights:
```bash
./bin/k8s-analyzer -ai-model=gpt-4o
```

### For Quick Analysis
Use `gpt-4o-mini` for faster, cheaper analysis:
```bash
./bin/k8s-analyzer -ai-model=gpt-4o-mini
```

### For Development/Testing
Use `gpt-3.5-turbo` to save costs:
```bash
./bin/k8s-analyzer -ai-model=gpt-3.5-turbo
```

### Without AI
Skip AI entirely (still get full analysis):
```bash
# Just don't set OPENAI_API_KEY
unset OPENAI_API_KEY
./bin/k8s-analyzer
```

## Environment Variable

You can also set a default model via environment:

```bash
# In your ~/.zshrc or ~/.bashrc
export K8S_ANALYZER_MODEL="gpt-4o-mini"

# Then just run
./bin/k8s-analyzer
```

(Note: This would require a small code change to read from env var)

## Cost Comparison

Approximate costs for analyzing a 500-pod cluster:

| Model | Tokens (est) | Cost per run |
|-------|--------------|--------------|
| `gpt-3.5-turbo` | ~5,000 | ~$0.01 |
| `gpt-4o-mini` | ~5,000 | ~$0.02 |
| `gpt-4o` | ~5,000 | ~$0.10 |
| `gpt-4-turbo` | ~5,000 | ~$0.20 |

*Costs are approximate and vary based on cluster size and complexity*
