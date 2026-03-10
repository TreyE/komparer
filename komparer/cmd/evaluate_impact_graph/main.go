package main

import (
	"encoding/csv"
	"fmt"
	"maps"
	"os"
	"slices"
	"sort"
	"strings"

	"github.com/ideacrew/komparer/internal"
)

const (
	Added = iota
	Removed
	Modified
)

var changeKindMapping = map[string]int{
	"A": Added,
	"D": Removed,
	"M": Modified,
}

type FileChange struct {
	Kind int
	Path string
}

type ChangeResults struct {
	Paths        []string
	Environments []string
}

func extractDiffChanges(diffDataPath string) []FileChange {
	var fcs []FileChange

	ddf, _ := os.Open(diffDataPath)

	csvReader := csv.NewReader(ddf)

	// 3. Set the delimiter to a tab character
	csvReader.Comma = '\t'

	records, _ := csvReader.ReadAll()

	for _, r := range records {
		fcs = append(
			fcs,
			FileChange{
				Kind: changeKindMapping[r[0]],
				Path: r[1],
			},
		)
	}
	return fcs
}

func makeChangeResults(impactedFiles []string) ChangeResults {
	impactedEnvMap := make(map[string]bool)
	sort.Strings(impactedFiles)
	for _, impactedFile := range impactedFiles {
		e, f := strings.CutPrefix(impactedFile, "environments/")
		if f {
			impactedEnvMap[strings.Split(e, "/")[0]] = true
		}
	}
	impactedEnvs := slices.Collect(maps.Keys(impactedEnvMap))
	sort.Strings(impactedEnvs)
	return ChangeResults{
		Paths:        impactedFiles,
		Environments: impactedEnvs,
	}
}

func getDependentList(currentGraph internal.ImpactGraph, oldGraph internal.ImpactGraph, diffedFiles []FileChange) ChangeResults {
	allChanges := make(map[string]bool)
	for _, df := range diffedFiles {
		if df.Kind != Removed {
			deps := currentGraph.FindDependentsOf(df.Path)
			for _, d := range deps {
				allChanges[d] = true
			}
		}
	}
	for _, df := range diffedFiles {
		if df.Kind == Removed {
			deps := oldGraph.FindDependentsOf(df.Path)
			for _, d := range deps {
				allChanges[d] = true
			}
		}
	}
	stringies := slices.Collect(maps.Keys(allChanges))
	return makeChangeResults(stringies)
}

func main() {
	currentGraphDataPath := os.Args[1]
	oldGraphDataPath := os.Args[2]
	diffDatapath := os.Args[3]

	diffedFiles := extractDiffChanges(diffDatapath)
	var cg, og internal.ImpactGraph
	bc, _ := os.ReadFile(currentGraphDataPath)
	bo, _ := os.ReadFile(oldGraphDataPath)
	cg.GobDecode(bc)
	og.GobDecode(bo)

	allChanges := getDependentList(cg, og, diffedFiles)

	for _, c := range allChanges.Paths {
		fmt.Println(c)
	}
}
