package main

import (
	"fmt"
	"os"
	"slices"
	"sort"

	"github.com/alexflint/go-arg"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/hmdsefi/gograph"
	"github.com/ideacrew/komparer/internal"
	"github.com/nao1215/markdown"
)

type ChangedResources struct {
	AddedResources    []string
	RemovedResources  []string
	ChangedResources  []string
	ImpactedResources []internal.ImpactedResource
	Environments      []string
}

func compareContentChanges(og, cg internal.ImpactGraph) ChangedResources {
	okset := mapset.NewSetFromMapKeys(og.ResourceMap)
	nkset := mapset.NewSetFromMapKeys(cg.ResourceMap)
	new_keys := nkset.Difference(okset)
	removed_keys := okset.Difference(nkset)
	common_keys := okset.Intersect(nkset)

	changedByContent := mapset.NewSet[string]()

	for _, k := range common_keys.ToSlice() {
		oldResource := og.ResourceMap[k]
		newResource := cg.ResourceMap[k]
		ory, _ := oldResource.AsYAML()
		nry, _ := newResource.AsYAML()
		if string(ory) != string(nry) {
			changedByContent.Add(k)
		}
	}

	changedByEnv := mapset.NewSet[string]()
	for _, k := range changedByContent.ToSlice() {
		searchVert := gograph.NewVertex(k)
		for _, edge := range cg.EnvDependencyMap.EdgesOf(searchVert) {
			if edge.Source() != searchVert {
				changedByEnv.Add(edge.Source().Label())
			}
		}
	}

	for _, k := range removed_keys.ToSlice() {
		searchVert := gograph.NewVertex(k)
		for _, edge := range og.EnvDependencyMap.EdgesOf(searchVert) {
			if edge.Source() != searchVert {
				if common_keys.Contains(edge.Source().Label()) {
					changedByEnv.Add(edge.Source().Label())
				}
			}
		}
	}

	finalChangeList := changedByContent.Union(changedByEnv)

	for _, nk := range new_keys.ToSlice() {
		finalChangeList.Remove(nk)
	}

	for _, rk := range removed_keys.ToSlice() {
		finalChangeList.Remove(rk)
	}

	var impactedRes []internal.ImpactedResource
	envSet := mapset.NewSet[string]()

	for _, nk := range new_keys.ToSlice() {
		ir := internal.ImpactedResourceFromString(internal.Added, nk)
		envSet.Add(ir.Environment)
		impactedRes = append(impactedRes, ir)
	}

	for _, rk := range removed_keys.ToSlice() {
		ir := internal.ImpactedResourceFromString(internal.Removed, rk)
		envSet.Add(ir.Environment)
		impactedRes = append(impactedRes, ir)
	}

	for _, ck := range finalChangeList.ToSlice() {
		ir := internal.ImpactedResourceFromString(internal.Modified, ck)
		envSet.Add(ir.Environment)
		impactedRes = append(impactedRes, ir)
	}

	envList := envSet.ToSlice()
	sort.Strings(envList)

	slices.SortStableFunc(impactedRes, internal.SortImpactedResources)

	return ChangedResources{
		AddedResources:    new_keys.ToSlice(),
		RemovedResources:  removed_keys.ToSlice(),
		ChangedResources:  finalChangeList.ToSlice(),
		ImpactedResources: impactedRes,
		Environments:      envList,
	}
}

func changeIconFor(ir internal.ImpactedResource) string {
	switch ir.ChangeKind {
	case internal.Added:
		return ":heavy_plus_sign: "
	case internal.Removed:
		return ":x: "
	default:
		return ""
	}
}

func markdownSummarySection(md *markdown.Markdown, cg internal.ImpactGraph, changesByContent ChangedResources) {

	md.H2("Summary").LF()

	var rows [][]string

	rows = append(rows, []string{"Added", fmt.Sprintf("%d", len(changesByContent.AddedResources))})
	rows = append(rows, []string{"Removed", fmt.Sprintf("%d", len(changesByContent.RemovedResources))})
	rows = append(rows, []string{"Modified", fmt.Sprintf("%d", len(changesByContent.ChangedResources))})
	rows = append(rows, []string{"**Total**", fmt.Sprintf("**%d**", len(changesByContent.ImpactedResources))})

	md.Table(
		markdown.TableSet{
			Header:    []string{"Kind", "Count"},
			Rows:      rows,
			Alignment: []markdown.TableAlignment{markdown.AlignCenter, markdown.AlignLeft, markdown.AlignRight},
		},
	).LF()

	md.PlainText("**Impacted Environments:**").LF()
	formattedEnvList := make([]string, len(changesByContent.Environments))
	for i, e := range changesByContent.Environments {
		formattedEnvList[i] = "**" + e + "**"
	}

	md.OrderedList(formattedEnvList...).LF()
}

func main() {
	var args struct {
		OldGraphDataPath     string `arg:"positional,required" help:"Path to impact graph for the earlier commit"`
		CurrentGraphDataPath string `arg:"positional,required" help:"Path to impact graph for the current commit"`
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

	var cg, og internal.ImpactGraph
	bc, _ := os.ReadFile(currentGraphDataPath)
	bo, _ := os.ReadFile(oldGraphDataPath)
	cg.GobDecode(bc)
	og.GobDecode(bo)

	changesByContent := compareContentChanges(og, cg)

	if args.Markdown {
		md := markdown.NewMarkdown(os.Stdout)

		markdownSummarySection(md, cg, changesByContent)

		var rows [][]string

		for _, c := range changesByContent.ImpactedResources {
			rows = append(rows, []string{c.Environment, c.Path, changeIconFor(c) + c.ResourceID})
		}
		md.H2("Impacted Resources").LF()

		md.Table(
			markdown.TableSet{
				Header:    []string{"Env", "File", "Resource"},
				Rows:      rows,
				Alignment: []markdown.TableAlignment{markdown.AlignCenter, markdown.AlignLeft, markdown.AlignLeft},
			},
		).LF()

		if len(cg.Failures) > 0 {
			md.H2("Build Failures").LF()

			md.PlainTextf("**Failed: %d**", len(cg.Failures)).LF()

			md.PlainText("<details>").LF()
			md.PlainText("<summary>Details</summary>").LF()
			md.LF()
			for k, v := range cg.Failures {
				md.H4(k).LF()
				md.CodeBlocks(markdown.SyntaxHighlightNone, v).LF()
			}
			md.PlainText("</details>")
		}

		md.Build()
		fmt.Println("")
	} else {
		fmt.Fprintf(os.Stdout, "Added Resources: %d\n", len(changesByContent.AddedResources))
		fmt.Fprintf(os.Stdout, "Removed Resources: %d\n", len(changesByContent.RemovedResources))
		fmt.Fprintf(os.Stdout, "Changed Resources: %d\n", len(changesByContent.ChangedResources))
		if len(cg.Failures) > 0 {
			fmt.Fprintf(os.Stderr, "Build Failures: %d\n", len(cg.Failures))
		}
		if len(changesByContent.Environments) > 0 {
			fmt.Fprint(os.Stdout, "Impacted Environments\n")
			for _, f := range changesByContent.Environments {
				fmt.Println(f)
			}
		}
	}
}
