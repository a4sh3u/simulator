package cmd

import (
	"log"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var cfgFile string
var rootCmd = &cobra.Command{
	Use:   "simulator",
	Short: "Simulator command line",
	Long: `
A distributed systems and infrastructure simulator for attacking and
debugging Kubernetes
`,
}

var logger *zap.SugaredLogger

func NewCmdRoot(logger *zap.SugaredLogger) *cobra.Command {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config-file", "c", "", "Path to the simulator config file")
	cobra.OnInitialize(initConfig)

	rootCmd.AddCommand(newConfigCommand(logger))
	rootCmd.AddCommand(newInfraCommand(logger))
	rootCmd.AddCommand(newScenarioCommand(logger))
	rootCmd.AddCommand(newSSHCommand(logger))
	rootCmd.AddCommand(newVersionCommand(logger))
	rootCmd.AddCommand(newCompletionCmd(logger))

	rootCmd.PersistentFlags().StringP("bucket", "b", "",
		"The name of the s3 bucket to use.  Must be globally unique and will be prefixed with 'simulator-'")
	rootCmd.MarkFlagRequired("bucket")
	viper.BindPFlag("bucket", rootCmd.PersistentFlags().Lookup("bucket"))

	rootCmd.PersistentFlags().StringP("loglevel", "l", "info", "Level of detail in output logging")
	viper.BindPFlag("loglevel", rootCmd.PersistentFlags().Lookup("loglevel"))

	rootCmd.PersistentFlags().StringP("tf-dir", "t", "./terraform/deployments/AWS",
		"Path to a directory containing the infrastructure scripts")
	viper.BindPFlag("tf-dir", rootCmd.PersistentFlags().Lookup("tf-dir"))

	// TODO: (rem) this is also used to locate the perturb.sh script which may be subsumed by this app
	rootCmd.PersistentFlags().StringP("scenarios-dir", "s", "./simulation-scripts",
		"Path to a directory containing a scenario manifest")
	viper.BindPFlag("scenarios-dir", rootCmd.PersistentFlags().Lookup("scenarios-dir"))

	return rootCmd
}

func initConfig() {
	viper.SetConfigType("yaml")
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(".")
		viper.SetConfigName("simulator")
	}

	err := viper.ReadInConfig()
	if err != nil {
		// todo(ajm) this errors if not in the same dir as `simulator.yaml`. Move those vars here?
		panic(errors.Wrapf(err, "Error reading config file"))
	}

	// read config from environment too
	viper.SetEnvPrefix("simulator")
	replacer := strings.NewReplacer("-", "_")
	viper.SetEnvKeyReplacer(replacer)
	viper.AutomaticEnv()
}

func newCompletionCmd(logger *zap.SugaredLogger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion",
		Short: "Generates Bash completion scripts",
		Long: `To load completion run

. <(simulator completion)

To configure your Bash shell to load completions for each session add to your bashrc

# ~/.bashrc or ~/.profile
. <(simulator completion)
`,
		Run: func(cmd *cobra.Command, args []string) {
			rootCmd.GenBashCompletion(os.Stdout)
		},
	}
	return cmd
}

func Execute() error {
	var err error
	logger, err = NewLogger("debug", "console")
	if err != nil {
		log.Fatalf("can't initialize zap logger: %v", err)
	}

	// TODO(ajm) not working as expected
	// flags aren't parsed until cmd.Execute() is called
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		var err error

		// logger writes to stderr
		logger, err = NewLogger(viper.GetString("loglevel"), "console")
		if err != nil {
			log.Fatalf("can't re-initialize zap logger: %v", err)
		}

		defer logger.Sync()

		logger.Debug("Starting CLI")

		// TODO(ajm) this doesn't propagate to subcommands. How to do the above on the logger object that's been passed to NewCmdRoot(logger)?
	}

	cmd := NewCmdRoot(logger)

	return cmd.Execute()
}
