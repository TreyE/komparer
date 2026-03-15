package internal

import (
	"bytes"
	"encoding/gob"

	"github.com/hmdsefi/gograph"
	"sigs.k8s.io/kustomize/api/hasher"
	"sigs.k8s.io/kustomize/api/resource"
)

type ImpactGraph struct {
	Data        gograph.Graph[string]
	ResourceMap map[string]*resource.Resource
	// Resource ID => Environment Map Resource ID
	EnvDependencyMap gograph.Graph[string]
	Failures         map[string]string
}

func (ig *ImpactGraph) GobEncode() ([]byte, error) {
	var b bytes.Buffer
	var list [][]string
	var edList [][]string
	for _, edge := range ig.Data.AllEdges() {
		list = append(list, []string{edge.Source().Label(), edge.Destination().Label()})
	}
	for _, edEdge := range ig.EnvDependencyMap.AllEdges() {
		edList = append(edList, []string{edEdge.Source().Label(), edEdge.Destination().Label()})
	}
	enc := gob.NewEncoder(&b)
	encodableResources := ig.serializableResourceMap()
	enc.Encode(encodableResources)
	enc.Encode(&edList)
	enc.Encode(&list)
	enc.Encode(ig.Failures)
	return b.Bytes(), nil
}

func (ig *ImpactGraph) GobDecode(b []byte) error {
	failures := make(map[string]string)
	resMap := make(map[string]string)
	newData := gograph.New[string](gograph.Directed())
	newEnvDepMap := gograph.New[string](gograph.Directed())
	dec := gob.NewDecoder(bytes.NewReader(b))
	var edges [][]string
	var edEdges [][]string
	dec.Decode(&resMap)
	dec.Decode(&edEdges)
	dec.Decode(&edges)
	dec.Decode(&failures)
	for _, edge := range edges {
		srcV := gograph.NewVertex[string](edge[0])
		destV := gograph.NewVertex[string](edge[1])
		newData.AddEdge(srcV, destV)
	}
	for _, edge := range edEdges {
		srcV := gograph.NewVertex[string](edge[0])
		destV := gograph.NewVertex[string](edge[1])
		newEnvDepMap.AddEdge(srcV, destV)
	}
	ig.EnvDependencyMap = newEnvDepMap
	ig.ResourceMap = ig.unserializeResourceMap(resMap)
	ig.Data = newData
	ig.Failures = failures
	return nil
}

func findGraphDependents(graph gograph.Graph[string], src string) []string {
	dependsOnPath := gograph.NewVertex(src)
	res := []string{}
	for _, v := range graph.EdgesOf(dependsOnPath) {
		if v.Destination().Label() == src {
			res = append(res, v.Source().Label())
		}
	}
	return res
}

func (ig *ImpactGraph) FindDependentsOf(src string) []string {
	return findGraphDependents(ig.Data, src)
}

func (ig *ImpactGraph) unserializeResourceMap(resMap map[string]string) map[string]*resource.Resource {
	result := make(map[string]*resource.Resource)
	for k, v := range resMap {
		nf := resource.NewFactory(&hasher.Hasher{})
		r, _ := nf.FromBytes([]byte(v))
		result[k] = r
	}
	return result
}

func (ig *ImpactGraph) serializableResourceMap() map[string]string {
	result := make(map[string]string)
	for k, v := range ig.ResourceMap {
		asYaml, _ := v.AsYAML()
		result[k] = string(asYaml)
	}
	return result
}
