/*
Copyright © 2020 Sebastian Kurfuerst

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"log"
	"os"
	"strings"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use: "drydock",
}

type VSCodeAttachedContainerT struct {
	ContainerName string `json:"containerName"`
}

// see https://pkg.go.dev/github.com/docker/cli/cli-plugins/manager#Metadata
type DockerPluginMetadata struct {
	// SchemaVersion describes the version of this struct. Mandatory, must be "0.1.0"
	SchemaVersion string `json:",omitempty"`
	// Vendor is the name of the plugin vendor. Mandatory
	Vendor string `json:",omitempty"`
	// Version is the optional version of this plugin.
	Version string `json:",omitempty"`
	// ShortDescription should be suitable for a single line help message.
	ShortDescription string `json:",omitempty"`
	// URL is a pointer to the plugin's homepage.
	URL string `json:",omitempty"`
}

func buildDockerCliPluginMetadata(version, commit string) *cobra.Command {
	return &cobra.Command{
		Use:    "docker-cli-plugin-metadata",
		Hidden: true,
		Run: func(cmd *cobra.Command, args []string) {

			desc := "Run a command in a running container as ROOT user"
			println(os.Args[0])
			if len(os.Args) >= 1 && strings.Contains(os.Args[0], "docker-vscode") {
				desc = "Open VSCode as container"
			}

			metadata := DockerPluginMetadata{
				SchemaVersion:    "0.1.0",
				Vendor:           "sandstorm",
				Version:          fmt.Sprintf("%s - %s", version, commit),
				ShortDescription: desc,
			}
			res, err := json.Marshal(metadata)
			if err != nil {
				log.Fatalf("Error building up plugin metadata - should never happen: %s", err)
			}
			fmt.Println(string(res))
		},
	}
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(version, commit string) {
	rootCmd.AddCommand(buildDockerCliPluginMetadata(version, commit))
	rootCmd.AddCommand(buildExecRootCmd())
	rootCmd.AddCommand(buildVsCodeCommand())
	rootCmd.AddCommand(buildSpxCommand())
	rootCmd.AddCommand(buildXdebugCommand())
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize()
}
