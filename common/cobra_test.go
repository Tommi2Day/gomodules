package common

import (
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

var (
	// RootCmd function to execute in tests
	RootCmd = &cobra.Command{
		Use:   "",
		Short: "flags – unit test for Cobra",
		Long:  ``,
	}
	FlagCmd = &cobra.Command{
		Use:   "flags",
		Short: "flags – unit test for Cobra",
		Long:  ``,
		Run: func(cmd *cobra.Command, _ []string) {
			log.SetLevel(log.DebugLevel)
			log.SetOutput(cmd.OutOrStdout())
			if debugFlag {
				log.Debugf("Flag is set")
			} else {
				log.Debugf("Flag is not set")
			}
		},
	}
	debugFlag = false
)

func init() {
	// parse commandline
	RootCmd.PersistentFlags().BoolVarP(&debugFlag, "debug", "", false, "verbose debug output")
	RootCmd.AddCommand(FlagCmd)
}

func TestCobraFlagsChanged(t *testing.T) {
	actual := CmdFlagChanged(RootCmd, "test")
	assert.Equal(t, false, actual)
}
func TestCobraCmd(t *testing.T) {
	args := []string{
		"flags",
		"--debug",
	}
	actual, err := CmdRun(RootCmd, args)
	assert.NoError(t, err)
	assert.Contains(t, actual, "Flag is set")
}
