package internal

import (
	"cmp"
	"strings"
)

const (
	Added    ChangeType = 0
	Removed  ChangeType = 1
	Modified ChangeType = 2
)

type ChangeType int

type ImpactedResource struct {
	ChangeKind  ChangeType
	Environment string
	Path        string
	ResourceID  string
}

func ImpactedResourceFromString(kind ChangeType, resString string) ImpactedResource {
	vals := strings.SplitN(resString, ":", 3)
	return ImpactedResource{
		ChangeKind:  kind,
		Environment: vals[0],
		Path:        vals[1],
		ResourceID:  vals[2],
	}
}

func SortImpactedResources(ir1, ir2 ImpactedResource) int {
	if ir1.Environment != ir2.Environment {
		return cmp.Compare(ir1.Environment, ir2.Environment)
	}
	if ir1.ChangeKind != ir2.ChangeKind {
		return cmp.Compare(int(ir1.ChangeKind), int(ir2.ChangeKind))
	}
	if ir1.Path != ir2.Path {
		return cmp.Compare(ir1.Path, ir2.Path)
	}
	if ir1.ResourceID != ir2.ResourceID {
		return cmp.Compare(ir1.ResourceID, ir2.ResourceID)
	}
	return 0
}
