package internal

import (
	"bytes"
	"encoding/gob"

	"github.com/hmdsefi/gograph"
)

type ImpactGraph struct {
	Data gograph.Graph[string]
}

func (ig *ImpactGraph) GobEncode() ([]byte, error) {
	var b bytes.Buffer
	var list [][]string
	for _, edge := range ig.Data.AllEdges() {
		list = append(list, []string{edge.Source().Label(), edge.Destination().Label()})
	}
	enc := gob.NewEncoder(&b)
	enc.Encode(&list)
	return b.Bytes(), nil
}

func (ig *ImpactGraph) GobDecode(b []byte) error {
	newData := gograph.New[string](gograph.Directed())
	dec := gob.NewDecoder(bytes.NewReader(b))
	var edges [][]string
	dec.Decode(&edges)
	for _, edge := range edges {
		srcV := gograph.NewVertex[string](edge[0])
		destV := gograph.NewVertex[string](edge[1])
		newData.AddEdge(srcV, destV)
	}
	ig.Data = newData
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
