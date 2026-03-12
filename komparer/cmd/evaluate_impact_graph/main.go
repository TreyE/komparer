package main

import (
	"encoding/csv"
	"fmt"
	"maps"
	"os"
	"slices"
	"sort"
	"strings"

	"github.com/alexflint/go-arg"
	"github.com/ideacrew/komparer/internal"
	"github.com/nao1215/markdown"
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
	Resources    []internal.ImpactedResource
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
	var impactedResources []internal.ImpactedResource
	sort.Strings(impactedFiles)
	for _, impactedFile := range impactedFiles {
		impactedEnvMap[strings.Split(impactedFile, ":")[0]] = true
		impactedResources = append(impactedResources, internal.ImpactedResourceFromString(impactedFile))
	}
	impactedEnvs := slices.Collect(maps.Keys(impactedEnvMap))
	sort.Strings(impactedEnvs)
	return ChangeResults{
		Resources:    impactedResources,
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
	var args struct {
		OldGraphDataPath     string `arg:"positional,required" help:"Path to impact graph for the earlier commit"`
		CurrentGraphDataPath string `arg:"positional,required" help:"Path to impact graph for the current commit"`
		DiffDataPath         string `arg:"positional,required" help:"Path to the diff file between the two commits"`
		Markdown             bool   `arg:"-m" help:"Emit results in markdown format"`
	}
	p, _ := arg.NewParser(arg.Config{}, &args)
	err := p.Parse(os.Args[1:])
	switch {
	case err == arg.ErrHelp: // indicates that user wrote "--help" on command line
		p.WriteHelp(os.Stdout)
		os.Exit(0)
	case err != nil:
		fmt.Printf("error: %v\n", err)
		p.WriteHelp(os.Stdout)
		os.Exit(1)
	}

	currentGraphDataPath := args.CurrentGraphDataPath
	oldGraphDataPath := args.OldGraphDataPath
	diffDatapath := args.DiffDataPath

	diffedFiles := extractDiffChanges(diffDatapath)
	var cg, og internal.ImpactGraph
	bc, _ := os.ReadFile(currentGraphDataPath)
	bo, _ := os.ReadFile(oldGraphDataPath)
	cg.GobDecode(bc)
	og.GobDecode(bo)

	allChanges := getDependentList(cg, og, diffedFiles)

	if args.Markdown {
		md := markdown.NewMarkdown(os.Stdout)

		md.H2("Summary").LF()

		md.PlainTextf("**Impacted Resources: %d**", len(allChanges.Resources)).LF()
		md.PlainTextf("**Failed Builds: %d**", len(cg.Failures)).LF()

		md.PlainText("**Impacted Environments:**").LF()
		formattedEnvList := make([]string, len(allChanges.Environments))
		for i, e := range allChanges.Environments {
			formattedEnvList[i] = "**" + e + "**"
		}

		md.OrderedList(formattedEnvList...).LF()

		var rows [][]string

		for _, c := range allChanges.Resources {
			rows = append(rows, []string{c.Environment, c.Path, c.ResourceID})
		}
		md.H3("Impacted Resources").LF()

		md.Table(
			markdown.TableSet{
				Header:    []string{"Env", "File", "Resource"},
				Rows:      rows,
				Alignment: []markdown.TableAlignment{markdown.AlignCenter, markdown.AlignLeft, markdown.AlignLeft},
			},
		).LF()

		if len(cg.Failures) > 0 {
			md.H3("Build Failures")
			md.LF()
			for k, v := range cg.Failures {
				md.H4(k).LF()
				md.CodeBlocks(markdown.SyntaxHighlightNone, v).LF()
			}
		}

		md.Build()
		fmt.Println("")
	} else {
		fmt.Fprintf(os.Stdout, "Changed Resources: %d", len(allChanges.Resources))
		if len(cg.Failures) > 0 {
			fmt.Fprintf(os.Stderr, "Build Failures: %d\n", len(cg.Failures))
		}
		if len(allChanges.Environments) > 0 {
			fmt.Fprint(os.Stdout, "Impacted Environments\n")
			for _, f := range allChanges.Environments {
				fmt.Println(f)
			}
		}
	}
}
