package cli

// isNotFound is a no-op placeholder. The baidu library does not define a
// sentinel ErrNotFound; network and decode errors are the only failure modes.
func isNotFound(_ error) bool {
	return false
}
