package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"path"
	"runtime"

	t "github.com/JekaMas/nicecache/template"
)

const generatedSuffix = "_nicecache.go"

func main() {
	storedType := flag.String("type", "", "Type to cache")
	cachePackage := flag.String("cachePackage", "", "Package to store generated cache")
	cacheSize := flag.Int64("cacheSize", 1000, "Cache size")

	flag.Parse()

	pkgDir, err := packageDir(*cachePackage)
	if err != nil {
		panic(err)
	}

	outputFile := formatFileName(*storedType)

	cacheFilePath := filepath.Join(pkgDir, outputFile)
	writer, err := os.Create(cacheFilePath)
	if err != nil {
		panic(err)
	}
	defer writer.Close()

	generator := &Generator{}

	m := metadata(*storedType, *cachePackage, *cacheSize, pkgDir)

	if err := generator.Generate(writer, t.Code, m); err != nil {
		panic(err)
	}
}

//Metadata for template code generation
type Metadata struct {
	PackageName       string
	StoredTypePackage string
	StoredType        string

	IndexBuckets int
	CacheSize    int64

	CacheSizeConstName string
	IndexBucketsName   string

	FreeBatchPercentName   string
	AlphaName              string
	MaxFreeRatePercentName string

	GcTimeName         string
	GcChunkPercentName string
	GcChunkSizeName    string

	DeletedValueFlagName string

	FreeBatchSizeName string
	DeletedValueName  string

	StoredValueType string
	CacheType       string
	InnerCacheType  string
}

func metadata(storedType string, cachePackage string, cacheSize int64, packageDir string) (m Metadata) {
	withSuffix := withTypeSuffix(storedType)

	m.StoredType = storedType
	m.CacheType = "Cache" + storedType
	m.InnerCacheType = "innerCache" + storedType
	m.PackageName = filepath.Base(packageDir)

	m.CacheSize = cacheSize
	m.CacheSizeConstName = withSuffix("cacheSize")

	m.IndexBuckets = 100
	m.IndexBucketsName = withSuffix("indexBuckets")

	m.StoredValueType = withSuffix("storedValue")

	m.FreeBatchPercentName = withSuffix("freeBatchPercent")
	m.AlphaName = withSuffix("alpha")
	m.MaxFreeRatePercentName = withSuffix("maxFreeRatePercent")

	m.GcTimeName = withSuffix("gcTime")
	m.GcChunkPercentName = withSuffix("gcChunkPercent")
	m.GcChunkSizeName = withSuffix("gcChunkSize")

	m.DeletedValueFlagName = withSuffix("deletedValueFlag")

	m.FreeBatchSizeName = withSuffix("freeBatchSize")
	m.DeletedValueName = withSuffix("deletedValue")

	if cachePackage != "" {
		p, _ := os.Getwd()
		goPath := os.Getenv("GOPATH") + "/src/"
		m.StoredTypePackage = fmt.Sprintf("\n\t. \"%s\"", strings.TrimPrefix(p, goPath))
	}

	return m
}

func withTypeSuffix(suffix string) func(string) string {
	return func(s string) string {
		return s + suffix
	}
}

//Generator template code generator
type Generator struct{}

//Generate code by template and metadata
func (g *Generator) Generate(writer io.WriteCloser, templ string, metadata Metadata) error {
	tmpl, err := g.template(templ)
	if err != nil {
		return err
	}

	return tmpl.Execute(writer, metadata)
}

func (g *Generator) template(templateCode string) (*template.Template, error) {
	templateGenerator := template.New("nicecache")

	return templateGenerator.Parse(templateCode)
}

func formatFileName(typeName string) string {
	return fmt.Sprintf("%s"+generatedSuffix, strings.ToLower(typeName))
}

func packageDir(packageName string) (string, error) {
	if packageName == "" {
		return os.Getwd()
	}

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("No caller information")
	}
	sourcePathParts := strings.Split(path.Dir(filename), string(os.PathSeparator))

	// remove last part
	sourcePathParts = sourcePathParts[:len(sourcePathParts)-1]

	destinationPath := strings.Split(packageName, string(os.PathSeparator))

	workPathParts := append(sourcePathParts, destinationPath...)
	workDir := string(os.PathSeparator) + filepath.Join(workPathParts...)

	return workDir, nil
}
