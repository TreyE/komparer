package analysis

import (
	"slices"

	"go.yaml.in/yaml/v2"
	"sigs.k8s.io/kustomize/api/internal/utils"
)

const (
	SourcePathsAnnotation = utils.SourcePathsAnnotation
)

type SourcePath struct {
	Paths       []string
	stringValue string
}

func (s SourcePath) String() string {
	return s.stringValue
}

func SourcePathFromString(inString *string) (*SourcePath, error) {
	if inString == nil {
		return nil, nil
	}
	var paths []string
	err := yaml.Unmarshal([]byte(*inString), &paths)
	if err != nil {
		return nil, err
	}
	return &SourcePath{
		Paths:       paths,
		stringValue: *inString,
	}, nil
}

func SourcePathFromRawString(inString string) SourcePath {
	b, _ := yaml.Marshal([]string{inString})
	return SourcePath{
		Paths:       []string{inString},
		stringValue: string(b),
	}
}

func SourcePathFromArray(inPaths []string) SourcePath {
	sv, _ := yaml.Marshal(inPaths)
	return SourcePath{
		Paths:       inPaths,
		stringValue: string(sv),
	}
}

func MergeSourcePathLists(oldOriginString string, newOriginString string) string {
	oldOriginArray, err := SourcePathFromString(&oldOriginString)
	if err != nil {
		return newOriginString
	}
	newOriginArray, nerr := SourcePathFromString(&newOriginString)
	if nerr != nil {
		return newOriginString
	}
	originArray := slices.Concat(oldOriginArray.Paths, newOriginArray.Paths)
	var resultArray []string
	seen := make(map[string]bool)
	for _, v := range originArray {
		if _, found := seen[v]; !found {
			seen[v] = true
			resultArray = append(resultArray, v)
		}
	}
	return SourcePathFromArray(resultArray).stringValue
}

func MergeSourcePathAnnotationSets(originalAnnotations map[string]string, newAnnotations map[string]string) {
	if oVal, hasOrig := originalAnnotations[utils.SourcePathsAnnotation]; hasOrig {
		if nVal, hasNew := newAnnotations[utils.SourcePathsAnnotation]; hasNew {
			newAnnotations[utils.SourcePathsAnnotation] = MergeSourcePathLists(oVal, nVal)
		} else {
			newAnnotations[utils.SourcePathsAnnotation] = oVal
		}
	} else if nVal, hasNew := newAnnotations[utils.SourcePathsAnnotation]; hasNew {
		newAnnotations[utils.SourcePathsAnnotation] = nVal
	}
}
