package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"encoding/json"

	"sigs.k8s.io/kustomize/api/analysis"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/kyaml/filesys"
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

func statsForEnv(environment string, impactBuilder *internal.ImpactBuilder, pRelative string) {
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
		resMap, nerr := k.Run(fsys, kPath)
		if nerr != nil {
			continue
		}

		for _, rId := range resMap.AllIds() {
			res, lookupErr := resMap.GetByCurrentId(rId)
			if lookupErr != nil {
				panic(lookupErr)
			}
			resIdString := rId.String()
			annos := res.GetAnnotations()
			if spVal, hasSPKey := annos[analysis.SourcePathsAnnotation]; hasSPKey {
				sp, spErr := analysis.SourcePathFromString(&spVal)
				if spErr != nil {
					panic(spErr)
				}

				for _, sPath := range sp.Paths {
					impactBuilder.ResourceDependsOnFile(environment, match, resIdString, sPath)
					if "ConfigMap" == res.GetKind() {
						impactBuilder.ConfigMapDependsOnFile(environment, res.GetName(), sPath)
					}
				}

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
					impactBuilder.ResourceDependsOnEnv(environment, match, resIdString, rName)
				}
			}

			cmr, _ := jsonpath.JsonPathLookup(json_data, "$..configMapRef")
			if cmr != nil {
				rList, _ := cmr.([]interface{})
				for _, item := range rList {
					rMap, _ := item.(map[string]interface{})
					rName, _ := rMap["name"].(string)
					impactBuilder.ResourceDependsOnEnv(environment, match, resIdString, rName)
				}
			}
		}
	}
}

func main() {
	rootPath := os.Args[1]
	storePath := os.Args[2]
	impactBuilder := internal.NewImpactBuilder(rootPath)
	envDirList := listEnvs(rootPath)
	for _, edn := range envDirList {
		statsForEnv(edn, impactBuilder, rootPath)
	}
	vData := impactBuilder.BuildGraph()
	b, _ := vData.GobEncode()
	os.WriteFile(storePath, b, 0644)
}
