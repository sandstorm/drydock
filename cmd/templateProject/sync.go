package templateProject

import (
	"context"
	"embed"
	"fmt"
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/gookit/color"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

func BuildSyncCommand() *cobra.Command {
	var (
		templateDir string
		stdout      bool
	)

	var command = &cobra.Command{
		Use:   "sync [filename]",
		Short: "Sync the filename from project with a template project",
		Long:  `Sync updates the given filename by applying changes from a template project.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filename := args[0]

			if templateDir == "" {
				return fmt.Errorf("template directory is required")
			}

			if stdout {
				fmt.Println("Dry run mode - no changes will be made")
			}

			currentProjectFileContents, err := os.ReadFile(filename)
			if err != nil {
				return fmt.Errorf("reading file %s: %w", filename, err)
			}

			currentTemplateFileContents, err := os.ReadFile(filepath.Join(templateDir, filename))
			if err != nil {
				return fmt.Errorf("reading file %s: %w", filepath.Join(templateDir, filename), err)
			}
			println(currentProjectFileContents)
			println(currentTemplateFileContents)

			projectFirstCommitTimestamp, err := getFirstCommitTimestamp(".")
			if err != nil {
				return fmt.Errorf("could not read first commit timestamp: %w", err)
			}
			color.Printf("Found first commit timestamp: %v", projectFirstCommitTimestamp)

			/*fileAtProjectStart, err := getFileAtTimestamp(templateDir, filename, projectFirstCommitTimestamp)
			if err != nil {
				return fmt.Errorf("could not read file at project start: %w", err)
			}*/
			fileAtProjectStart := ""

			projectFiles, err := listFiles(".")
			if err != nil {
				return fmt.Errorf("could not list files from repo: %w", err)
			}

			templateFiles, err := listFiles(templateDir)
			if err != nil {
				return fmt.Errorf("could not list files from template repo: %w", err)
			}

			err = anthropicApi("anthropicPrompt.tmpl", syncPromptParams{
				ProjectStructure:     strings.Join(projectFiles, "\n"),
				TemplateStructure:    strings.Join(templateFiles, "\n"),
				CurrentFileName:      filename,
				CurrentProjectFile:   string(currentProjectFileContents),
				CurrentTemplateFile:  string(currentTemplateFileContents),
				OriginalTemplateFile: fileAtProjectStart,
			})
			if err != nil {
				return fmt.Errorf("calling anthropic API: %w", err)
			}

			return nil
		},
	}

	// Add flags
	command.Flags().StringVar(&templateDir, "template", "", "Directory containing the template git repository")
	command.Flags().BoolVar(&stdout, "stdout", false, "Output the modified file in stdout, instead of modifying in-place")

	return command
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

func anthropicApi(templateFileName string, data any) error {
	templates, err := template.ParseFS(templateFS, "*.tmpl")
	if err != nil {
		return fmt.Errorf("loading templates: %w", err)
	}
	var buf strings.Builder
	err = templates.ExecuteTemplate(&buf, templateFileName, data)

	if err != nil {
		return fmt.Errorf("rendering template: %w", err)
	}

	client := anthropic.NewClient(
	//option.WithAPIKey("my-anthropic-api-key"), // defaults to os.LookupEnv("ANTHROPIC_API_KEY")
	)
	message, err := client.Messages.New(context.Background(), anthropic.MessageNewParams{
		Model:     anthropic.F(anthropic.ModelClaude3_5SonnetLatest),
		MaxTokens: anthropic.F(int64(2048)),
		Messages: anthropic.F([]anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(buf.String())),
		}),
	})
	color.Printf("<op=italic;>%s</>", buf.String())
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("%+v\n", message.Content)
	return nil
}

func getFileAtTimestamp(repoPath string, filePath string, timestamp time.Time) (string, error) {
	// Open the repository
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return "", fmt.Errorf("failed to open repo: %w", err)
	}

	// Get repo HEAD reference
	ref, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD: %w", err)
	}

	// Get commit log
	cIter, err := repo.Log(&git.LogOptions{From: ref.Hash()})
	if err != nil {
		return "", fmt.Errorf("failed to get log: %w", err)
	}

	// Find the most recent commit before the timestamp
	var targetCommit *object.Commit
	err = cIter.ForEach(func(c *object.Commit) error {
		if c.Committer.When.Before(timestamp) || c.Committer.When.Equal(timestamp) {
			targetCommit = c
			return storer.ErrStop
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to iterate commits: %w", err)
	}

	if targetCommit == nil {
		return "", fmt.Errorf("no commit found before timestamp")
	}

	// Get the file from the commit's tree
	file, err := targetCommit.File(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to get file: %w", err)
	}

	// Get the file contents
	contents, err := file.Contents()
	if err != nil {
		return "", fmt.Errorf("failed to get contents: %w", err)
	}

	return contents, nil
}

func getFirstCommitTimestamp(repoPath string) (time.Time, error) {
	// Open the repository
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to open repo: %w", err)
	}

	// Get repo HEAD reference
	ref, err := repo.Head()
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get HEAD: %w", err)
	}

	// Get commit log
	cIter, err := repo.Log(&git.LogOptions{From: ref.Hash()})
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get log: %w", err)
	}

	// Find the last (first) commit
	var firstCommit *object.Commit
	err = cIter.ForEach(func(c *object.Commit) error {
		firstCommit = c
		return nil
	})
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to iterate commits: %w", err)
	}

	if firstCommit == nil {
		return time.Time{}, fmt.Errorf("no commits found")
	}

	return firstCommit.Committer.When, nil
}

func listFiles(repoPath string) ([]string, error) {
	// Open the repository
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repo: %w", err)
	}

	// Get HEAD reference
	ref, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	// Get the commit object
	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get commit: %w", err)
	}

	// Get the tree from the commit
	tree, err := commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get tree: %w", err)
	}

	var files []string
	// Walk the tree
	err = tree.Files().ForEach(func(f *object.File) error {
		files = append(files, f.Name)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk tree: %w", err)
	}

	return files, nil
}
