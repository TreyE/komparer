package internal

import "strings"

type ImpactedResource struct {
	Environment string
	Path        string
	ResourceID  string
}

func ImpactedResourceFromString(resString string) ImpactedResource {
	vals := strings.SplitN(resString, ":", 3)
	return ImpactedResource{
		Environment: vals[0],
		Path:        vals[1],
		ResourceID:  vals[2],
	}
}
