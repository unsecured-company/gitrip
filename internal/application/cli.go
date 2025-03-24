package application

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

func getConfigCli(args []string) (cfg *Config, mErr *MultiErr) {
	mErr = &MultiErr{}
	cfg = &Config{}
	cfg.Command = CmdHelp

	var rootCmd = &cobra.Command{
		Use:           os.Args[0],
		SilenceErrors: true,
	}

	cmdCheck := getConfigCheck(cfg)
	cmdFetch := getConfigFetch(cfg)
	cmdIndex := getConfigIndex(cfg)

	rootCmd.AddCommand(cmdCheck, cmdFetch, cmdIndex)
	rootCmd.SetArgs(args)

	mErr.Add(rootCmd.Execute())

	return
}

func getConfigCheck(cfg *Config) *cobra.Command {
	var checkCmd = &cobra.Command{
		Use:   CmdCheck + " [flags] [url]",
		Short: "Check URL or batch file of URLs",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cfg.Command = CmdCheck

			if len(args) > 0 {
				cfg.URL = args[0]
			}
		},
		PostRunE: func(cmd *cobra.Command, args []string) error {
			err := checkForUrlAndFile(cfg)

			return err
		},
	}

	addFetchFlags(cfg, checkCmd)

	return checkCmd
}

func getConfigFetch(cfg *Config) *cobra.Command {
	var fetchCmd = &cobra.Command{
		Use:   CmdFetch + " [flags] [url]",
		Short: "Fetch URL or batch file of URLs",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cfg.Command = CmdFetch

			if len(args) > 0 {
				cfg.URL = args[0]
			}
		},
		PostRunE: func(cmd *cobra.Command, args []string) error {
			err := checkForUrlAndFile(cfg)

			return err
		},
	}

	addFetchFlags(cfg, fetchCmd)
	fetchCmd.Flags().BoolVarP(&cfg.Update, "update", "u", false, "Update existing")

	return fetchCmd
}

func getConfigIndex(cfg *Config) *cobra.Command {
	var indexCmd = &cobra.Command{
		Use:   CmdIndex + " [flags] [path]",
		Short: "List files from .git/index",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cfg.Command = CmdIndex
			cfg.IndexFile = args[0]
		},
	}

	indexCmd.Flags().BoolVarP(&cfg.Verbose, "verbose", "v", false, "Enable verbose mode")
	indexCmd.Flags().BoolVar(&cfg.Tree, "tree", false, "Show as tree")
	indexCmd.Flags().BoolVar(&cfg.Raw, "raw", false, "Show as raw data")

	return indexCmd
}

func addFetchFlags(cfg *Config, cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&cfg.Verbose, "verbose", "v", false, "Enable verbose mode")
	cmd.Flags().StringVar(&cfg.BatchFile, FlagFile, "", "Batch file with URLs")
	cmd.Flags().IntVar(&cfg.Timeout, "timeout", DefaultTimeout, "Network timeout in seconds")
}

func checkForUrlAndFile(cfg *Config) (err error) {
	eFile := cfg.BatchFile != ""
	eUrl := cfg.URL != ""

	if eFile && eUrl {
		err = fmt.Errorf("Only %s or --%s can be provided", FlagUrl, FlagFile)
	} else if !eFile && !eUrl {
		err = fmt.Errorf("You must specify %s or --%s", FlagUrl, FlagFile)
	}

	return
}
