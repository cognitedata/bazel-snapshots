/* Copyright 2022 Cognite AS */

package main

// Execute adds all child commands to the root command HugoCmd and sets flags appropriately.
// The args are usually filled with os.Args[1:].
func Execute(args []string) error {
	rootCmd := newRootCmd()
	cmd := rootCmd.cmd
	cmd.SetArgs(args)

	return cmd.Execute()
}
