package analysis

type LoadedFileHistory struct {
	paths []string
}

var loadedFiles *LoadedFileHistory

func init() {
	loadedFiles = &LoadedFileHistory{}
}

func ClearPaths() {
	loadedFiles.paths = []string{}
}

func GetPaths() []string {
	return loadedFiles.paths
}

func AddPath(p string) {
	loadedFiles.paths = append(loadedFiles.paths, p)
}

func GetLoadedFileHistory() []string {
	return loadedFiles.paths
}
