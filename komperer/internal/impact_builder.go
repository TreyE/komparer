package internal

import (
	"fmt"
	"strings"

	"github.com/hmdsefi/gograph"
)

type ImpactBuilder struct {
	rootPath string
	// uniqueness list for environment map names
	envMapList map[string]bool
	// resource ID => yaml file graph
	impactGraph gograph.Graph[string]
	// resource ID => environment map name graph
	envGraph gograph.Graph[string]
	// environment map name => environment definition yaml graph
	envDefGraph gograph.Graph[string]
	// List of failed builds
	buildFailures map[string]error
}

func NewImpactBuilder(root string) *ImpactBuilder {
	return &ImpactBuilder{
		rootPath:      root,
		envMapList:    make(map[string]bool),
		impactGraph:   gograph.New[string](gograph.Directed()),
		envGraph:      gograph.New[string](gograph.Directed()),
		envDefGraph:   gograph.New[string](gograph.Directed()),
		buildFailures: make(map[string]error),
	}
}

func (ib *ImpactBuilder) ConfigMapDependsOnFile(environment string, envName string, sAbsPath string) {
	eMapName := environment + "/" + envName
	fp, _ := strings.CutPrefix(sAbsPath, ib.rootPath)
	eVert := gograph.NewVertex(eMapName)
	fVert := gograph.NewVertex(fp)
	ib.envMapList[eMapName] = true
	ib.envDefGraph.AddEdge(eVert, fVert)
}

func (ib *ImpactBuilder) ResourceDependsOnFile(environment string, resAbsPath string, resId string, sAbsPath string) {
	resPath, _ := strings.CutPrefix(resAbsPath, ib.rootPath)
	tVert := gograph.NewVertex(environment + ":" + resPath + ":" + resId)
	fp, _ := strings.CutPrefix(sAbsPath, ib.rootPath)
	fVert := gograph.NewVertex(fp)
	ib.impactGraph.AddEdge(tVert, fVert)
}

func (ib *ImpactBuilder) ResourceDependsOnEnv(environment string, resAbsPath string, resId string, envName string) {
	eMapName := environment + "/" + envName
	ib.envMapList[eMapName] = true
	resPath, _ := strings.CutPrefix(resAbsPath, ib.rootPath)
	tVert := gograph.NewVertex(environment + ":" + resPath + ":" + resId)
	eVert := gograph.NewVertex(eMapName)
	ib.envGraph.AddEdge(tVert, eVert)
}

func (ib *ImpactBuilder) BuildGraph() ImpactGraph {
	for k, _ := range ib.envMapList {
		v1 := ib.envGraph.GetVertexByID(k)
		v2 := ib.envDefGraph.GetVertexByID(k)
		for _, v := range ib.envGraph.EdgesOf(v1) {
			if v.Destination().Label() == k {
				for _, vl := range ib.envDefGraph.EdgesOf(v2) {
					if vl.Source().Label() == k {
						vs := gograph.NewVertex(v.Source().Label())
						vd := gograph.NewVertex(vl.Destination().Label())
						ib.impactGraph.AddEdge(vs, vd)
					}
				}
			}
		}
	}
	return ImpactGraph{
		Data: ib.impactGraph,
	}
}

func (ib *ImpactBuilder) BuildFailure(environment string, absKustomizePath string, err error) {
	kPath, _ := strings.CutPrefix(absKustomizePath, ib.rootPath)
	fKey := environment + ":" + kPath
	ib.buildFailures[fKey] = err
}

func (ib *ImpactBuilder) ListFailures() {
	for k, v := range ib.buildFailures {
		fmt.Println(k)
		fmt.Println(v)
	}
}
