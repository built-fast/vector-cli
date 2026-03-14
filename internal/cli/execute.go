package cli

// Execute creates the root command and runs it.
// It returns 0 on success or 1 on error.
func Execute() int {
	cmd := NewRootCmd()
	if err := cmd.Execute(); err != nil {
		return 1
	}
	return 0
}
