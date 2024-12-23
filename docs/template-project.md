# `drydock template-project sync` - keep your project in sync with template changes

**This is an ALPHA feature and might change in future versions without notice.**

## Background

When starting a new project from a template or kickstarter, it's common for the template project to receive updates and improvements over time. However, keeping your project in sync with these template changes can be challenging. You need to carefully review and apply changes while preserving your project-specific modifications.

**`drydock template-project sync` uses AI to update your project files with changes from the template project.**

## Prerequisites

You need an Anthropic API key to use this feature. The API key can be provided in one of these ways (in order of precedence):

1. Direct CLI flag: `--api-key`
2. Environment variable: `ANTHROPIC_API_KEY`
3. Kubernetes secret (for Sandstorm internal usage):
    - Configured via `--api-key-path-k8s` (format: `namespace/secret-name/secret-value`)
    - Default path: `drydock/anthropic-creds/ANTHROPIC_API_KEY`
    - Uses kubeconfig from `--kubeconfig` flag (defaults to `~/.kube/config`)

## Usage

Basic syntax:
```bash
drydock template-project sync --template <template-dir> [flags] <files...>
```

### Examples

Sync a single file from template:
```bash
drydock template-project sync --template ~/src/neos-on-docker-kickstart Dockerfile
```

Sync multiple files from template:
```bash
drydock template-project sync --template ~/src/neos-on-docker-kickstart Dockerfile docker-compose.yml
```

Sync an entire folder from template:
```bash
drydock template-project sync --template ~/src/neos-on-docker-kickstart deployment/
```

Compare with template files (useful for reviewing changes before applying):
```bash
# Places template files as .tmpl next to original files for comparison
drydock template-project sync --compare --template ~/src/neos-on-docker-kickstart deployment/

# After reviewing, remove the .tmpl files
git clean -f
```

## Options

- `--template`: (Required) Directory containing the template git repository
- `--replace`: Replace the modified files directly (default: true)
- `--compare`: Place template files next to project files as .tmpl for comparison
- `--api-key`: Anthropic API key (optional)
- `--api-key-path-k8s`: Path to retrieve API key from K8S (format: namespace/secret-name/secret-value)
- `--kubeconfig`: Path to kubeconfig file (default: ~/.kube/config)

## How It Works

The command uses Anthropic Claude AI to:

1. Compare your project files with the corresponding template files
2. Identify meaningful changes and improvements in the template
3. Apply these changes while preserving your project-specific modifications

It handles both single files and entire directory structures.