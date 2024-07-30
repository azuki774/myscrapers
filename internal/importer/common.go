package importer

const defaultOutputDir = "/data"

type ImporterCommon struct {
	ws        string // browser (ex: "172.19.250.172:7317")
	outputDir string
}
