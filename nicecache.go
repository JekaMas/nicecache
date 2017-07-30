package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	t "nicecache/template"
)

/*
1. взять текущее имя пакета или переданное в dest
2. взять имя текущее имя пакета
3. взять имя текущего типа
4. взять размер кэша
*/

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
	writer, err := os.Create(filepath.Join(pkgDir, outputFile))
	if err != nil {
		panic(err)
	}
	defer writer.Close()

	generator := &Generator{}

	m := metadata(*storedType, *cachePackage, *cacheSize, pkgDir)

	if err := generator.Generate(writer, t.Template, m); err != nil {
		panic(err)
	}
}

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

	m.FreeBatchPercentName  = withSuffix("freeBatchPercent")
	m.AlphaName = withSuffix("alpha")
	m.MaxFreeRatePercentName = withSuffix("maxFreeRatePercent")

	m.GcTimeName  = withSuffix("gcTime")
	m.GcChunkPercentName  = withSuffix("gcChunkPercent")
	m.GcChunkSizeName  = withSuffix("gcChunkSize")

	m.DeletedValueFlagName  = withSuffix("deletedValueFlag")

	m.FreeBatchSizeName  = withSuffix("freeBatchSize")
	m.DeletedValueName  = withSuffix("deletedValue")

	// todo add import of Stored type package
	return m
}

func withTypeSuffix(suffix string) func(string) string {
	return func(s string) string {
		return s + suffix
	}
}

type Generator struct{}

func (g *Generator) Generate(writer io.WriteCloser, templ string, metadata Metadata) error {
	tmpl, err := g.template(templ)
	if err != nil {
		return err
	}

	return tmpl.Execute(writer, metadata)
}

func (g *Generator) template(templ string) (*template.Template, error) {
	tmpl := template.New("nicecache")

	return tmpl.Parse(templ)
}

func formatFileName(typeName string) string {
	return fmt.Sprintf("%s"+generatedSuffix, strings.ToLower(typeName))
}

func packageDir(packageName string) (string, error) {
	if packageName == "" {
		return os.Getwd()
	}

	path := os.Getenv("GOPATH")
	if path == "" {
		return "", errors.New("GOPATH is not set")
	}

	workDir := filepath.Join(path, "src", packageName)
	if _, err := os.Stat(workDir); err != nil {
		return "", err
	}

	return workDir, nil
}
