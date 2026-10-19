package metering

import "regexp"

var (
	ExcludeBylonisWorkspaceResourceAttrs = regexp.MustCompile(`^bylonis\.workspace\.`)
)
