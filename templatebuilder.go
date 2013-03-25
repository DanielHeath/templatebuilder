package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type TemplateFile struct {
	DestVarName    string
	DestConstName  string
	RawSrcLen      int
	RawSrc         []byte
	EncodedDataLen int
	EncodedData    []byte
	EncodedString  string
}

var srcName, destName string
var destPackageName string
var src, dest *os.File
var fileTemplate template.Template

func init() {
	flag.StringVar(&srcName, "src", "", "source folder for templates")
	flag.StringVar(&destName, "dest", "", "destination folder for templates")
	flag.Parse()
	if srcName == "" || destName == "" {
		fmt.Println("`src` and `dest` must be specified.")
		flag.Usage()
		os.Exit(1)
	}

	destPackageName = filepath.Base(destName)

	fileTemplate = *template.Must(template.New("section").Parse(`
		var {{ .DestVarName }} *html.Template
		func init() {
			{{ .DestVarName }} = html.Must(html.New("{{ .DestConstName }}").Parse({{.DestConstName}}))
		}
		const {{ .DestConstName }} = {{ .EncodedString }}
	`))

}

func rmContents(dir *os.File) {
	destNames, err := dir.Readdirnames(0)
	dieIf(err)

	for _, name := range destNames {
		dieIf(os.RemoveAll(name))
	}
}

func dieIf(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func writed(str string) {
	_, err := dest.WriteString(str)
	dieIf(err)
}

func main() {
	var err error
	src, err = os.Open(srcName)
	dieIf(err)

	destDir, err := os.Open(destName)
	dieIf(err)
	rmContents(destDir)

	dest, err = os.Create(destName + "/" + destPackageName + ".go")
	dieIf(err)
	writed("package " + destPackageName + "\n")
	writed(`import (
		html "html/template"
	)
	`)
	writed(`const backtick = "` + "`" + `"` + "\n")
	filepath.Walk(srcName, processFile)
}

func processFile(path string, info os.FileInfo, err error) error {
	var f TemplateFile

	if info.IsDir() {
		return nil
	}

	if !strings.HasPrefix(path, srcName) {
		log.Fatal("Path isnt in src: ", path, srcName)
	}

	if !strings.HasSuffix(path, ".html") {
		return nil
	}

	f.RawSrc = make([]byte, info.Size())
	file, err := os.Open(path)
	dieIf(err)
	_, err = file.Read(f.RawSrc)
	dieIf(err)

	f.EncodedString = fmt.Sprintf("%q", f.RawSrc)
	f.EncodedString = "`" + strings.Replace(string(f.RawSrc), "`", "` + backtick + `", -1) + "`"
	f.DestConstName = filepath.Clean(path[len(srcName)+1:])
	// Strip off '.html'
	f.DestConstName = f.DestConstName[0 : len(f.DestConstName)-5]
	// Replace slashes and dots with underscores
	f.DestConstName = strings.Replace(f.DestConstName, "/", "_", -1)
	f.DestConstName = strings.Replace(f.DestConstName, ".", "_", -1)

	// Make the variable name public
	f.DestVarName = strings.ToUpper(f.DestConstName[0:1]) + f.DestConstName[1:]
	dieIf(fileTemplate.Execute(dest, f))
	return nil
}
