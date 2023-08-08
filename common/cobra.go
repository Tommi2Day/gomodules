package common

import (
	"bytes"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// CmdFlagChanged checks if the named commandline flag has been supplied and changed
func CmdFlagChanged(cmd *cobra.Command, name string) (changed bool) {
	changed = false
	f := cmd.Flags().Lookup(name)
	if f != nil {
		changed = f.Changed
		if changed {
			log.Debugf("Flag %s  has been changed", name)
		}
	} else {
		log.Warnf("Flag %s  unknown", name)
	}
	return
}

// CmdRun starts a command and returns captured output
func CmdRun(cmd *cobra.Command, args []string) (out string, err error) {
	b := bytes.NewBufferString("")
	log.SetOutput(cmd.OutOrStdout())
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs(args)
	err = cmd.Execute()
	out = b.String()
	return
}
