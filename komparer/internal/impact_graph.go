package internal

import (
	"bytes"
	"encoding/gob"

	"github.com/hmdsefi/gograph"
	"sigs.k8s.io/kustomize/api/hasher"
	"sigs.k8s.io/kustomize/api/resource"
)

type ImpactGraph struct {
	ResourceMap map[string]*resource.Resource
	// Resource ID => Environment Map Resource ID
	EnvDependencyMap gograph.Graph[string]
	Failures         map[string]string
}

func (ig *ImpactGraph) GobEncode() ([]byte, error) {
	var b bytes.Buffer
	var edList [][]string
	for _, edEdge := range ig.EnvDependencyMap.AllEdges() {
		edList = append(edList, []string{edEdge.Source().Label(), edEdge.Destination().Label()})
	}
	enc := gob.NewEncoder(&b)
	encodableResources := ig.serializableResourceMap()
	enc.Encode(encodableResources)
	enc.Encode(&edList)
	enc.Encode(ig.Failures)
	return b.Bytes(), nil
}

func (ig *ImpactGraph) GobDecode(b []byte) error {
	failures := make(map[string]string)
	resMap := make(map[string]string)
	newEnvDepMap := gograph.New[string](gograph.Directed())
	dec := gob.NewDecoder(bytes.NewReader(b))
	var edEdges [][]string
	dec.Decode(&resMap)
	dec.Decode(&edEdges)
	dec.Decode(&failures)
	for _, edge := range edEdges {
		srcV := gograph.NewVertex[string](edge[0])
		destV := gograph.NewVertex[string](edge[1])
		newEnvDepMap.AddEdge(srcV, destV)
	}
	ig.EnvDependencyMap = newEnvDepMap
	ig.ResourceMap = ig.unserializeResourceMap(resMap)
	ig.Failures = failures
	return nil
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
