package main

import (
	"fmt"
	"os"

	"github.com/hmdsefi/gograph"

	"github.com/ideacrew/komparer/internal"
)

func findDependentsOf(graph gograph.Graph[string], src string) []string {
	dependsOnPath := gograph.NewVertex(src)
	res := []string{}
	for _, v := range graph.EdgesOf(dependsOnPath) {
		if v.Destination().Label() == src {
			res = append(res, v.Source().Label())
		}
	}
	return res
}

func main() {
	graphDataPath := os.Args[1]

	var ig internal.ImpactGraph
	b, _ := os.ReadFile(graphDataPath)
	ig.GobDecode(b)
	fmt.Println(findDependentsOf(ig.Data, "base/config/redis-configmap.yaml"))
}
