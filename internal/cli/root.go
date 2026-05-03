package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/hamdyelbatal122/lynxeye/internal/app"
	"github.com/hamdyelbatal122/lynxeye/internal/version"
)

func NewRootCommand() *cobra.Command {
	var configPath string
	var once bool

	rootCmd := &cobra.Command{
		Use:           version.BinaryName,
		Short:         "Cluster logs into patterns and alert on anomaly spikes",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Start the streaming detection pipeline",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return app.Run(cmd.Context(), configPath, once)
		},
	}
	runCmd.Flags().StringVarP(&configPath, "config", "c", "config.example.yaml", "Path to YAML configuration")
	runCmd.Flags().BoolVar(&once, "once", false, "Process finite input and exit instead of tailing indefinitely")

	sampleConfigCmd := &cobra.Command{
		Use:   "sample-config",
		Short: "Print guidance for the starter configuration file",
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintln(cmd.OutOrStdout(), "Copy config.example.yaml to config.yaml and adjust your sources and alert credentials.")
		},
	}

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "%s %s (%s)\n", version.DisplayName, version.Version, version.Commit)
		},
	}

	rootCmd.AddCommand(runCmd, sampleConfigCmd, versionCmd)
	return rootCmd
}
