package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/hmdsefi/gograph"

	"encoding/json"

	"sigs.k8s.io/kustomize/api/analysis"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/yaml"
	kyaml "sigs.k8s.io/yaml"

	"github.com/oliveagle/jsonpath"

	"github.com/ideacrew/komparer/internal"
)

func listEnvs(rootDir string) []string {
	var eList []string
	environmentPath := rootDir + "/environments/"
	fEntries, _ := os.ReadDir(environmentPath)
	for _, de := range fEntries {
		if de.IsDir() {
			n, _ := strings.CutPrefix(de.Name(), environmentPath)
			eList = append(eList, n)
		}
	}
	return eList
}

func glob(dir string, exts []string) ([]string, error) {
	files := []string{}
	err := filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		if slices.Contains(exts, filepath.Base(path)) {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func getMapField(node yaml.RNode, field string) *yaml.RNode {
	var res *yaml.RNode
	fMap := node.Field(field)
	if fMap != nil {
		res = fMap.Value
	}
	return res
}

func statsForEnv(environment string, pRelative string, graph gograph.Graph[string], envgraph gograph.Graph[string], envdefgraph gograph.Graph[string], envMapList *map[string]bool) {
	exts := []string{"kustomization.yml", "kustomization.yaml"}
	matches, err := glob(pRelative+"environments/"+environment+"/", exts)
	if err != nil {
		return
	}
	fsys := filesys.MakeFsOnDisk()
	opts := krusty.MakeDefaultOptions()
	k := krusty.MakeKustomizer(opts)
	for _, match := range matches {
		kPath := filepath.Dir(match)
		tPath, _ := strings.CutPrefix(match, pRelative)
		resMap, nerr := k.Run(fsys, kPath)
		if nerr != nil {
			continue
		}

		/*for _, r := range resMap.Resources() {
			fmt.Println(r)
		}*/

		for _, rId := range resMap.AllIds() {
			res, lookupErr := resMap.GetByCurrentId(rId)
			if lookupErr != nil {
				panic(lookupErr)
			}
			resIdString := rId.String()
			tVert := gograph.NewVertex(resIdString)
			annos := res.GetAnnotations()
			if spVal, hasSPKey := annos[analysis.SourcePathsAnnotation]; hasSPKey {
				sp, spErr := analysis.SourcePathFromString(&spVal)
				if spErr != nil {
					panic(spErr)
				}
				resMapYaml, _ := res.AsYAML()
				resMapJson, _ := kyaml.YAMLToJSON(resMapYaml)

				var json_data interface{}
				json.Unmarshal(resMapJson, &json_data)

				jpres, _ := jsonpath.JsonPathLookup(json_data, "$..configMapKeyRef")
				if jpres != nil {
					rList, _ := jpres.([]interface{})
					for _, item := range rList {
						rMap, _ := item.(map[string]interface{})
						rName, _ := rMap["name"].(string)
						eMapName := environment + "/" + rName
						(*envMapList)[eMapName] = true
						fVertex := gograph.NewVertex(eMapName)
						envgraph.AddEdge(tVert, fVertex)
					}
				}

				cmr, _ := jsonpath.JsonPathLookup(json_data, "$..configMapRef")
				if cmr != nil {
					rList, _ := cmr.([]interface{})
					for _, item := range rList {
						rMap, _ := item.(map[string]interface{})
						rName, _ := rMap["name"].(string)
						eMapName := environment + "/" + rName
						(*envMapList)[eMapName] = true
						fVertex := gograph.NewVertex(eMapName)
						envgraph.AddEdge(tVert, fVertex)
					}
				}

				for _, sPath := range sp.Paths {
					fp := filepath.Join(tPath, "..", sPath)
					fVert := gograph.NewVertex(fp)
					graph.AddEdge(tVert, fVert)
					if "ConfigMap" == res.GetKind() {
						eMapName := environment + "/" + res.GetName()
						(*envMapList)[eMapName] = true
						eVert := gograph.NewVertex(eMapName)
						envdefgraph.AddEdge(eVert, fVert)
					}
				}

			}

		}
	}
}

func main() {
	rootPath := os.Args[1]
	storePath := os.Args[2]
	// kustomize.yaml => other yamls graph
	// also our final dependency graph
	graph := gograph.New[string](gograph.Directed())
	// kustomize.yaml => environment map name graph
	envgraph := gograph.New[string](gograph.Directed())
	// environment map name => environment definition yaml graph
	envdefgraph := gograph.New[string](gograph.Directed())
	// list for environment map names
	envMaplist := make(map[string]bool)
	envDirList := listEnvs(rootPath)
	for _, edn := range envDirList {
		statsForEnv(edn, rootPath, graph, envgraph, envdefgraph, &envMaplist)
	}
	//statsForEnv("pvt", rootPath, graph, envgraph, envdefgraph, &envMaplist)
	// statsForEnv("pvt-2", rootPath, graph, envgraph, envdefgraph, &envMaplist)
	// Link the maps
	for k, _ := range envMaplist {
		v1 := envgraph.GetVertexByID(k)
		v2 := envdefgraph.GetVertexByID(k)
		for _, v := range envgraph.EdgesOf(v1) {
			if v.Destination().Label() == k {
				for _, vl := range envdefgraph.EdgesOf(v2) {
					if vl.Source().Label() == k {
						vs := gograph.NewVertex(v.Source().Label())
						vd := gograph.NewVertex(vl.Destination().Label())
						graph.AddEdge(vs, vd)
					}
				}
			}
		}
	}
	vData := internal.ImpactGraph{
		Data: graph,
	}
	b, _ := vData.GobEncode()
	os.WriteFile(storePath, b, 0644)
}
