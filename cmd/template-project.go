package cmd

import (
	"github.com/sandstorm/drydock/cmd/templateProject"
	"github.com/spf13/cobra"
)

func buildTemplateProjectCommand() *cobra.Command {
	var command = &cobra.Command{
		Use:   "template-project",
		Short: "Sub-Commands for syncing the project with a template project",
	}

	command.AddCommand(templateProject.BuildSyncCommand())

	return command
}
