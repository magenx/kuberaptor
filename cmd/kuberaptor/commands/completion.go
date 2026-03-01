package commands

import (
	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion script",
	Long: `Generate shell completion script for the specified shell.

Examples:
  # Bash
  source <(kuberaptor completion bash)
  # To load completions for each session, execute once:
  kuberaptor completion bash > /etc/bash_completion.d/kuberaptor

  # Zsh
  source <(kuberaptor completion zsh)
  # To load completions for each session, execute once:
  kuberaptor completion zsh > "${fpath[1]}/_kuberaptor"

  # Fish
  kuberaptor completion fish | source
  # To load completions for each session, execute once:
  kuberaptor completion fish > ~/.config/fish/completions/kuberaptor.fish

  # PowerShell
  kuberaptor completion powershell | Out-String | Invoke-Expression
  # To load completions for each session, execute once:
  kuberaptor completion powershell > kuberaptor.ps1
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return cmd.Root().GenBashCompletion(cmd.OutOrStdout())
		case "zsh":
			return cmd.Root().GenZshCompletion(cmd.OutOrStdout())
		case "fish":
			return cmd.Root().GenFishCompletion(cmd.OutOrStdout(), true)
		case "powershell":
			return cmd.Root().GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
		}
		return nil
	},
}

func init() {
	// This command will be added to the root command in root.go
	//
}
