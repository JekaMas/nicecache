package nicecache

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"
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
	cacheSize := flag.String("cacheSize", "1000", "Cache size")

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

	m := metadata(*storedType, *typePointer, pkgDir)
	if err := generator.Generate(writer, m); err != nil {
		panic(err)
	}

	fmt.Printf("Generated %s %s\n", *format, outputFile)
}

type Metadata struct {
	PackageName       string
	StoredTypePackage string
	StoredType        string

	IndexBuckets int
	CacheSize    int

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

func metadata(typeName string, pointerType bool, packageDir string) (m Metadata) {
	m.Object = "obj"
	m.Type = typeName
	m.PackageName = filepath.Base(packageDir)

	if pointerType {
		m.MarshalObject = m.Object
	} else {
		m.MarshalObject = fmt.Sprintf("&%s", m.Object)
	}

	return m
}

type Generator struct{}

func (g *Generator) Generate(writer io.WriteCloser, templ io.Reader, metadata Metadata) error {
	tmpl, err := g.template(templ)
	if err != nil {
		return nil
	}

	return tmpl.Execute(writer, metadata)
}

func (g *Generator) template(templ io.Reader) (*template.Template, error) {
	res, err := ioutil.ReadAll(templ)
	if err != nil {
		return nil, err
	}

	tmpl := template.New("nicecache")
	return tmpl.Parse(string(res))
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
