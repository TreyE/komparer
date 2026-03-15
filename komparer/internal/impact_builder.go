package internal

import (
	"fmt"
	"strings"

	"github.com/hmdsefi/gograph"
	"sigs.k8s.io/kustomize/api/resource"
)

type ImpactBuilder struct {
	rootPath string
	resMap   map[string]*resource.Resource
	envMap   map[string]string
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
		resMap:        make(map[string]*resource.Resource),
		envMap:        make(map[string]string),
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

func (ib *ImpactBuilder) BuiltResource(environment string, resAbsPath string, resId string, res *resource.Resource) {
	resPath, _ := strings.CutPrefix(resAbsPath, ib.rootPath)
	tVert := environment + ":" + resPath + ":" + resId
	ib.resMap[tVert] = res
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

func (ib *ImpactBuilder) HasEnvMap(environment string, envMapName string, envFilePath string, envResourceId string) {
	eMapName := environment + "/" + envMapName
	resPath, _ := strings.CutPrefix(envFilePath, ib.rootPath)
	envResId := environment + ":" + resPath + ":" + envResourceId
	ib.envMap[eMapName] = envResId
}

func (ib *ImpactBuilder) BuildGraph() ImpactGraph {
	edMap := gograph.New[string](gograph.Directed())
	for _, edge := range ib.envGraph.AllEdges() {
		if envResId, hasEnvId := ib.envMap[edge.Destination().Label()]; hasEnvId {
			vs := gograph.NewVertex(edge.Source().Label())
			vd := gograph.NewVertex(envResId)
			edMap.AddEdge(vs, vd)
		}
	}
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

	failures := make(map[string]string)
	for k, v := range ib.buildFailures {
		failures[k] = fmt.Sprint(v)
	}
	return ImpactGraph{
		Data:             ib.impactGraph,
		ResourceMap:      ib.resMap,
		EnvDependencyMap: edMap,
		Failures:         failures,
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
