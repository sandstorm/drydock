package templateProject

import (
	"context"
	"embed"
	"fmt"
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/gookit/color"
	"github.com/spf13/cobra"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

func BuildSyncCommand() *cobra.Command {
	var (
		templateDir   string
		replace       bool
		compare       bool
		kubeconfig    string
		apiKeyPathK8s string
		apiKey        string
	)

	var command = &cobra.Command{
		Use:   "sync --template [template-dir] [filenames]",
		Short: "Sync the filename from project with a template project (ALPHA FEATURE)",
		Long: color.Sprintf(`
<op=bold;>Usage:  drydock template-project sync --template TEMPLATE [flags] FILES...</>
<gray>(ALPHA FEATURE - might change in further versions without notice)</>

<bold>updates the given files/directories by applying changes from a template project by using Anthropic Claude AI.</>
This is helpful to reduce drift between a kickstarter/template project and the actual project itself.


<op=underscore;>Examples:</>

<op=bold;>Sync a single file from template</>
    drydock template-project sync --template <op=italic;>~/src/neos-on-docker-kickstart Dockerfile</>

<op=bold;>Sync multiple files from template</>
    drydock template-project sync --template <op=italic;>~/src/neos-on-docker-kickstart Dockerfile docker-compose.yml</>

<op=bold;>Sync an entire folder from template</>
    drydock template-project sync --template <op=italic;>~/src/neos-on-docker-kickstart deployment/</>

<op=bold;>Comparing with Template Files</>
    drydock template-project sync <op=bold;>--compare</> --template <op=italic;>~/src/neos-on-docker-kickstart deployment/</>
      <gray># places the file from the template repository as .tmpl next to the original file - for easy comparison</>
    git clean -f <gray># remove the untracked .tmpl files again</>


<op=underscore;>Claude AI Key Configuration</>

The Anthropic API key is loaded from the following locations (1st one wins):

- <op=bold;>CLI Flag:</> Passed directly via --api-key flag
- <op=bold;>Environment Variable:</> ANTHROPIC_API_KEY environment variable
- <op=bold;>Kubernetes Secret:</> Retrieved from a Kubernetes secret, configured via:
  - --api-key-path-k8s (format: namespace/secret-name/secret-value - default: drydock/anthropic-creds/ANTHROPIC_API_KEY)
  - --kubeconfig flag (default: ~/.kube/config)

  <gray>This is tailored to Sandstorm usage, but we are open to ideas to make this easier customizable.</>

		`),
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if templateDir == "" {
				return fmt.Errorf("template directory is required")
			}

			if replace {
				fmt.Println("outputting to replace")
			}

			filesToProcess, err := listFilesRecursively(args)
			if err != nil {
				return fmt.Errorf("expanding directories: %w", err)
			}
			for _, filename := range filesToProcess {
				color.Fprintf(os.Stderr, "<suc>%s</>\n", filename)

				currentProjectFileContents, err := os.ReadFile(filename)
				if err != nil {
					return fmt.Errorf("reading file %s: %w", filename, err)
				}

				currentTemplateFileContents, err := os.ReadFile(filepath.Join(templateDir, filename))
				if err != nil {
					if os.IsNotExist(err) {
						fmt.Println("CONTINUING WITH NEXT FILE")
						continue
					}
					return fmt.Errorf("reading file %s: %w", filepath.Join(templateDir, filename), err)
				}

				apiKey, err := loadAnthropicApiKeyWithFallbackToK8S(apiKey, apiKeyPathK8s, kubeconfig)
				if err != nil {
					return fmt.Errorf("coul: %w", err)
				}

				if compare {
					copyFile(filepath.Join(templateDir, filename), filename+".tpl")
				}

				responseString, err := anthropicApi(apiKey, "anthropicPrompt.tmpl", syncPromptParams{
					ProjectStructure:     "",
					TemplateStructure:    "",
					CurrentFileName:      filename,
					CurrentProjectFile:   string(currentProjectFileContents),
					CurrentTemplateFile:  string(currentTemplateFileContents),
					OriginalTemplateFile: "",
				})
				if err != nil {
					return fmt.Errorf("calling anthropic API: %w", err)
				}
				if replace {
					err = os.WriteFile(filename, []byte(responseString), 0o644)
					if err != nil {
						return fmt.Errorf("writing file %s: %w", filename, err)
					}
				}
			}

			return nil
		},
	}

	// Add flags
	command.Flags().StringVar(&templateDir, "template", "", "Directory containing the template git repository")
	command.Flags().BoolVar(&replace, "replace", true, "Replace the modified file directly")
	command.Flags().BoolVar(&compare, "compare", false, "Place the template file next to the project file as .tmpl, for easy comparison")

	if home := homedir.HomeDir(); home != "" {
		command.Flags().StringVar(&kubeconfig, "kubeconfig", filepath.Join(home, ".kube", "config"), "Kube config file (to retrieve ANTHROPIC_API_KEY from cluster)")
	} else {
		command.Flags().StringVar(&kubeconfig, "kubeconfig", "", "Kube config file (to retrieve ANTHROPIC_API_KEY from cluster)")
	}
	command.Flags().StringVar(&apiKeyPathK8s, "api-key-path-k8s", "drydock/anthropic-creds/ANTHROPIC_API_KEY", "Path to retrieve API key from K8S, in the form namespace/secret-name/secret-value")
	command.Flags().StringVar(&apiKey, "api-key", "", "Anthropic API key to use (defaults to env ANTHROPIC_API_KEY)")

	return command
}

func copyFile(source string, destination string) {
	src, _ := os.Open(source)
	dst, _ := os.Create(destination)
	defer src.Close()
	defer dst.Close()
	io.Copy(dst, src)
}

func loadAnthropicApiKeyWithFallbackToK8S(apiKey string, apiKeyPathK8s string, kubeconfig string) (string, error) {
	// ------------------------------------
	// API key given via CLI args - always wins
	// ------------------------------------
	if len(apiKey) > 0 {
		return apiKey, nil
	}

	// ------------------------------------
	// API key given via env variable - wins before K8S
	// ------------------------------------
	val, exists := os.LookupEnv("ANTHROPIC_API_KEY")
	if exists {
		return val, nil
	}

	// ------------------------------------
	// fallback to K8S
	// ------------------------------------
	splitApiKeyPath := strings.SplitN(apiKeyPathK8s, "/", 3)
	if len(splitApiKeyPath) != 3 {
		return "", fmt.Errorf("api-key-path-k8s is not in the form namespace/secret-name/secret-value (given: %s)", apiKeyPathK8s)
	}

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return "", fmt.Errorf("reading kube config %s: %w", kubeconfig, err)
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", fmt.Errorf("reading creating clientset for %s: %w", kubeconfig, err)
	}

	// read the secret
	secret, err := clientset.CoreV1().Secrets(splitApiKeyPath[0]).Get(context.Background(), splitApiKeyPath[1], metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("reading secret on path %s: %w", apiKeyPathK8s, err)
	}
	value, exists := secret.Data[splitApiKeyPath[2]]
	if !exists {
		return "", fmt.Errorf("secret %s %s does not contain element %s", splitApiKeyPath[0], splitApiKeyPath[1], splitApiKeyPath[2])
	}
	return string(value), nil
}

type syncPromptParams struct {
	ProjectStructure     string
	TemplateStructure    string
	CurrentFileName      string
	CurrentProjectFile   string
	CurrentTemplateFile  string
	OriginalTemplateFile string
}

//go:embed *.tmpl
var templateFS embed.FS

func anthropicApi(apiKey, templateFileName string, data any) (string, error) {
	templates, err := template.ParseFS(templateFS, "*.tmpl")
	if err != nil {
		return "", fmt.Errorf("loading templates: %w", err)
	}
	var buf strings.Builder
	err = templates.ExecuteTemplate(&buf, templateFileName, data)

	if err != nil {
		return "", fmt.Errorf("rendering template: %w", err)
	}

	client := anthropic.NewClient(
		option.WithAPIKey(apiKey),
	)
	stream := client.Messages.NewStreaming(context.Background(), anthropic.MessageNewParams{
		Model:     anthropic.F(anthropic.ModelClaude3_5SonnetLatest),
		MaxTokens: anthropic.F(int64(2048)),
		Messages: anthropic.F([]anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(buf.String())),
		}),
	})
	var accumulated strings.Builder
	message := anthropic.Message{}
	for stream.Next() {
		event := stream.Current()
		message.Accumulate(event)

		switch delta := event.Delta.(type) {
		case anthropic.ContentBlockDeltaEventDelta:
			if delta.Text != "" {
				print(delta.Text)
				accumulated.WriteString(delta.Text)
			}
		}
	}

	if stream.Err() != nil {
		return "", fmt.Errorf("calling Anthropic API: %w", stream.Err())
	}
	return accumulated.String(), nil
}

func listFilesRecursively(paths []string) ([]string, error) {
	var allFiles []string

	for _, path := range paths {
		// Get file info to check if it's a directory
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("error accessing path %s: %w", path, err)
		}

		if info.IsDir() {
			// If it's a directory, walk it recursively
			err := filepath.Walk(path, func(filePath string, fileInfo os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				if !fileInfo.IsDir() {
					allFiles = append(allFiles, filePath)
				}
				return nil
			})

			if err != nil {
				return nil, fmt.Errorf("error walking directory %s: %w", path, err)
			}
		} else {
			// If it's a file, add it directly
			allFiles = append(allFiles, path)
		}
	}

	return allFiles, nil
}
