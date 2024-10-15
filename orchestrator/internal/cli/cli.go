package cli

import (
	"log"

	"my.org/novel_vmp/internal/config"
	"my.org/novel_vmp/internal/master"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewRootCommand() *cobra.Command {
	root_command := &cobra.Command{
		Use:   "novelvmpfang",
		Short: "novelvmpfang security scanner orchestrator",
	}

	root_command.AddCommand(
		&cobra.Command{
			Use:   "master",
			Short: "Start the master server",
			Run: func(cmd *cobra.Command, args []string) {
				server := master.NewServer()
				err := server.Start()
				if err != nil {
					log.Fatal(err)
				}
			},
		})

	flags := root_command.PersistentFlags()
	// flags.StringP("input", "i", "", "Path to input file")
	flags.Bool("scanner-test", false, "Run server in scanner test mode")

	viper.BindPFlags(flags)

	config.LoadViperConfig()
	config.InitKeys()

	return root_command
}
