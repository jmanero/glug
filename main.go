package main

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/jmanero/glug/pkg/logger"
	"github.com/spf13/cobra"
	"storj.io/common/memory"
)

// CLI for the program
var CLI = cobra.Command{
	Use:   "glug LOGFILE",
	Short: "Self-rotating file logger for runit",
	Args:  cobra.ExactArgs(1),
	RunE:  Logger,
}

// Flags for all subcommands
var Flags = CLI.PersistentFlags()

// Options populated by CLI flags
var Options = logger.RotatorOptions{
	MaxSize:    32 * memory.MiB,
	MinSize:    512 * memory.KiB,
	CreateMode: 0644,
}

func init() {
	Flags.BoolVar(&Options.Enabled, "rotate", true, "Enable log rotation")

	Flags.Var(&Options.MaxSize, "max-size", "Maximum byte-size of the output log-file")
	Flags.DurationVar(&Options.MaxAge, "max-age", time.Hour*24*7, "Maximum age for the output log-file")
	Flags.Var(&Options.MinSize, "min-size", "Block rotation of small log-files by age until they reach a minimum size threshold")
	Flags.IntVar(&Options.Count, "count", 4, "Number of rotated log-files to retain")
	Flags.StringVar(&Options.Pattern, "pattern", "%Y-%m-%dT%H%M%S", "strftime format string for rotated file name suffixes")
	Flags.Var(&Options.CreateMode, "mode", "Mode bits for log-file creation. Octal values are supported with a leading `0`")

	CLI.AddCommand(&cobra.Command{
		Use:   "rotate LOGFILE",
		Short: "Perform rotation upon the specified log file",
		Args:  cobra.ExactArgs(1),
		RunE:  Rotate,
	})
}

func main() {
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)

	err := CLI.ExecuteContext(ctx)
	if err != nil {
		CLI.PrintErrln(err)
		os.Exit(1)
	}
}

// Logger runs the log writer, reading from STDIN
func Logger(cmd *cobra.Command, args []string) error {
	return logger.Run(cmd.Context(), cmd.InOrStdin(), args[0], Options)
}

// Rotate applies rotation logic once to the current output file and rotated versions
func Rotate(cmd *cobra.Command, args []string) (err error) {
	_, err = logger.Rotate(cmd.Context(), args[0], Options)
	return
}
