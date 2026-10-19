package metering

import "regexp"

var (
	ExcludeSigNozWorkspaceResourceAttrs = regexp.MustCompile(`^bylonis\.workspace\.`)
)
