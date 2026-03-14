package cli

// Execute creates the root command and runs it.
func Execute() error {
	cmd := NewRootCmd()
	return cmd.Execute()
}
