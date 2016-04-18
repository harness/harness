// Copyright 2015 go-swagger maintainers
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package scan

import (
	"fmt"
	"go/ast"
	goparser "go/parser"
	"log"
	"regexp"
	"strings"

	"golang.org/x/tools/go/loader"

	"github.com/go-swagger/go-swagger/spec"
	"github.com/go-swagger/go-swagger/swag"
)

const (
	rxMethod = "(\\p{L}+)"
	rxPath   = "((?:/[\\p{L}\\p{N}\\p{Pd}\\p{Pc}{}\\-\\._~%!$&'()*+,;=:@/]*)+/?)"
	rxOpTags = "(\\p{L}[\\p{L}\\p{N}\\p{Pd}\\p{Pc}\\p{Zs}]+)"
	rxOpID   = "((?:\\p{L}[\\p{L}\\p{N}\\p{Pd}\\p{Pc}]+)+)"

	rxMaximumFmt    = "%s[Mm]ax(?:imum)?\\p{Zs}*:\\p{Zs}*([\\<=])?\\p{Zs}*([\\+-]?(?:\\p{N}+\\.)?\\p{N}+)$"
	rxMinimumFmt    = "%s[Mm]in(?:imum)?\\p{Zs}*:\\p{Zs}*([\\>=])?\\p{Zs}*([\\+-]?(?:\\p{N}+\\.)?\\p{N}+)$"
	rxMultipleOfFmt = "%s[Mm]ultiple\\p{Zs}*[Oo]f\\p{Zs}*:\\p{Zs}*([\\+-]?(?:\\p{N}+\\.)?\\p{N}+)$"

	rxMaxLengthFmt        = "%s[Mm]ax(?:imum)?(?:\\p{Zs}*[\\p{Pd}\\p{Pc}]?[Ll]en(?:gth)?)\\p{Zs}*:\\p{Zs}*(\\p{N}+)$"
	rxMinLengthFmt        = "%s[Mm]in(?:imum)?(?:\\p{Zs}*[\\p{Pd}\\p{Pc}]?[Ll]en(?:gth)?)\\p{Zs}*:\\p{Zs}*(\\p{N}+)$"
	rxPatternFmt          = "%s[Pp]attern\\p{Zs}*:\\p{Zs}*(.*)$"
	rxCollectionFormatFmt = "%s[Cc]ollection(?:\\p{Zs}*[\\p{Pd}\\p{Pc}]?[Ff]ormat)\\p{Zs}*:\\p{Zs}*(.*)$"

	rxMaxItemsFmt = "%s[Mm]ax(?:imum)?(?:\\p{Zs}*|[\\p{Pd}\\p{Pc}]|\\.)?[Ii]tems\\p{Zs}*:\\p{Zs}*(\\p{N}+)$"
	rxMinItemsFmt = "%s[Mm]in(?:imum)?(?:\\p{Zs}*|[\\p{Pd}\\p{Pc}]|\\.)?[Ii]tems\\p{Zs}*:\\p{Zs}*(\\p{N}+)$"
	rxUniqueFmt   = "%s[Uu]nique\\p{Zs}*:\\p{Zs}*(true|false)$"

	rxItemsPrefixFmt = "(?:[Ii]tems[\\.\\p{Zs}]*){%d}"
)

var (
	rxSwaggerAnnotation  = regexp.MustCompile("swagger:([\\p{L}\\p{N}\\p{Pd}\\p{Pc}]+)")
	rxMeta               = regexp.MustCompile("swagger:meta")
	rxFileUpload         = regexp.MustCompile("swagger:file")
	rxStrFmt             = regexp.MustCompile("swagger:strfmt\\p{Zs}*(\\p{L}[\\p{L}\\p{N}\\p{Pd}\\p{Pc}]+)$")
	rxName               = regexp.MustCompile("swagger:name\\p{Zs}*(\\p{L}[\\p{L}\\p{N}\\p{Pd}\\p{Pc}\\.]+)$")
	rxAllOf              = regexp.MustCompile("swagger:allOf\\p{Zs}*(\\p{L}[\\p{L}\\p{N}\\p{Pd}\\p{Pc}\\.]+)?$")
	rxModelOverride      = regexp.MustCompile("swagger:model\\p{Zs}*(\\p{L}[\\p{L}\\p{N}\\p{Pd}\\p{Pc}]+)?$")
	rxDiscriminated      = regexp.MustCompile("swagger:discriminated\\p{Zs}*(\\p{L}[\\p{L}\\p{N}\\p{Pd}\\p{Pc}\\p{Zs}]+)$")
	rxResponseOverride   = regexp.MustCompile("swagger:response\\p{Zs}*(\\p{L}[\\p{L}\\p{N}\\p{Pd}\\p{Pc}]+)?$")
	rxParametersOverride = regexp.MustCompile("swagger:parameters\\p{Zs}*(\\p{L}[\\p{L}\\p{N}\\p{Pd}\\p{Pc}\\p{Zs}]+)$")
	rxRoute              = regexp.MustCompile(
		"swagger:route\\p{Zs}*" +
			rxMethod +
			"\\p{Zs}*" +
			rxPath +
			"(?:\\p{Zs}+" +
			rxOpTags +
			")?\\p{Zs}+" +
			rxOpID + "\\p{Zs}*$")

	rxSpace              = regexp.MustCompile("\\p{Zs}+")
	rxNotAlNumSpaceComma = regexp.MustCompile("[^\\p{L}\\p{N}\\p{Zs},]")
	rxPunctuationEnd     = regexp.MustCompile("\\p{Po}$")
	rxStripComments      = regexp.MustCompile("^[^\\p{L}\\p{N}\\p{Pd}\\p{Pc}\\+]*")
	rxStripTitleComments = regexp.MustCompile("^[^\\p{L}]*[Pp]ackage\\p{Zs}+[^\\p{Zs}]+\\p{Zs}*")

	rxIn              = regexp.MustCompile("[Ii]n\\p{Zs}*:\\p{Zs}*(query|path|header|body|formData)$")
	rxRequired        = regexp.MustCompile("[Rr]equired\\p{Zs}*:\\p{Zs}*(true|false)$")
	rxDiscriminator   = regexp.MustCompile("[Dd]iscriminator\\p{Zs}*:\\p{Zs}*(true|false)$")
	rxReadOnly        = regexp.MustCompile("[Rr]ead(?:\\p{Zs}*|[\\p{Pd}\\p{Pc}])?[Oo]nly\\p{Zs}*:\\p{Zs}*(true|false)$")
	rxConsumes        = regexp.MustCompile("[Cc]onsumes\\p{Zs}*:")
	rxProduces        = regexp.MustCompile("[Pp]roduces\\p{Zs}*:")
	rxSecuritySchemes = regexp.MustCompile("[Ss]ecurity\\p{Zs}*:")
	rxSecurity        = regexp.MustCompile("[Ss]ecurity\\p{Zs}*[Dd]efinitions:")
	rxResponses       = regexp.MustCompile("[Rr]esponses\\p{Zs}*:")
	rxSchemes         = regexp.MustCompile("[Ss]chemes\\p{Zs}*:\\p{Zs}*((?:(?:https?|HTTPS?|wss?|WSS?)[\\p{Zs},]*)+)$")
	rxVersion         = regexp.MustCompile("[Vv]ersion\\p{Zs}*:\\p{Zs}*(.+)$")
	rxHost            = regexp.MustCompile("[Hh]ost\\p{Zs}*:\\p{Zs}*(.+)$")
	rxBasePath        = regexp.MustCompile("[Bb]ase\\p{Zs}*-*[Pp]ath\\p{Zs}*:\\p{Zs}*" + rxPath + "$")
	rxLicense         = regexp.MustCompile("[Ll]icense\\p{Zs}*:\\p{Zs}*(.+)$")
	rxContact         = regexp.MustCompile("[Cc]ontact\\p{Zs}*-?(?:[Ii]info\\p{Zs}*)?:\\p{Zs}*(.+)$")
	rxTOS             = regexp.MustCompile("[Tt](:?erms)?\\p{Zs}*-?[Oo]f?\\p{Zs}*-?[Ss](?:ervice)?\\p{Zs}*:")
)

// Many thanks go to https://github.com/yvasiyarov/swagger
// this is loosely based on that implementation but for swagger 2.0

type setter func(interface{}, []string) error

func joinDropLast(lines []string) string {
	l := len(lines)
	lns := lines
	if l > 0 && len(strings.TrimSpace(lines[l-1])) == 0 {
		lns = lines[:l-1]
	}
	return strings.Join(lns, "\n")
}

func removeEmptyLines(lines []string) (notEmpty []string) {
	for _, l := range lines {
		if len(strings.TrimSpace(l)) > 0 {
			notEmpty = append(notEmpty, l)
		}
	}
	return
}

func rxf(rxp, ar string) *regexp.Regexp {
	return regexp.MustCompile(fmt.Sprintf(rxp, ar))
}

// The Opts for the application scanner.
type Opts struct {
	BasePath   string
	Input      *spec.Swagger
	ScanModels bool
}

// Application scans the application and builds a swagger spec based on the information from the code files.
// When there are includes provided, only those files are considered for the initial discovery.
// Similarly the excludes will exclude an item from initial discovery through scanning for annotations.
// When something in the discovered items requires a type that is contained in the includes or excludes it will still be
// in the spec.
func Application(opts Opts) (*spec.Swagger, error) {
	parser, err := newAppScanner(&opts, nil, nil)
	if err != nil {
		return nil, err
	}
	return parser.Parse()
}

// appScanner the global context for scanning a go application
// into a swagger specification
type appScanner struct {
	loader      *loader.Config
	prog        *loader.Program
	classifier  *programClassifier
	discovered  []schemaDecl
	input       *spec.Swagger
	definitions map[string]spec.Schema
	responses   map[string]spec.Response
	operations  map[string]*spec.Operation
	scanModels  bool

	// MainPackage the path to find the main class in
	MainPackage string
}

// newAppScanner creates a new api parser
func newAppScanner(opts *Opts, includes, excludes packageFilters) (*appScanner, error) {
	var ldr loader.Config
	ldr.ParserMode = goparser.ParseComments
	ldr.ImportWithTests(opts.BasePath)
	prog, err := ldr.Load()
	if err != nil {
		return nil, err
	}

	input := opts.Input
	if input == nil {
		input = new(spec.Swagger)
		input.Swagger = "2.0"
	}

	if input.Paths == nil {
		input.Paths = new(spec.Paths)
	}
	if input.Definitions == nil {
		input.Definitions = make(map[string]spec.Schema)
	}
	if input.Responses == nil {
		input.Responses = make(map[string]spec.Response)
	}

	return &appScanner{
		MainPackage: opts.BasePath,
		prog:        prog,
		input:       input,
		loader:      &ldr,
		operations:  collectOperationsFromInput(input),
		definitions: input.Definitions,
		responses:   input.Responses,
		scanModels:  opts.ScanModels,
		classifier: &programClassifier{
			Includes: includes,
			Excludes: excludes,
		},
	}, nil
}

func collectOperationsFromInput(input *spec.Swagger) map[string]*spec.Operation {
	operations := make(map[string]*spec.Operation)
	if input != nil && input.Paths != nil {
		for _, pth := range input.Paths.Paths {
			if pth.Get != nil {
				operations[pth.Get.ID] = pth.Get
			}
			if pth.Post != nil {
				operations[pth.Post.ID] = pth.Post
			}
			if pth.Put != nil {
				operations[pth.Put.ID] = pth.Put
			}
			if pth.Patch != nil {
				operations[pth.Patch.ID] = pth.Patch
			}
			if pth.Delete != nil {
				operations[pth.Delete.ID] = pth.Delete
			}
			if pth.Head != nil {
				operations[pth.Head.ID] = pth.Head
			}
			if pth.Options != nil {
				operations[pth.Options.ID] = pth.Options
			}
		}
	}
	return operations
}

// Parse produces a swagger object for an application
func (a *appScanner) Parse() (*spec.Swagger, error) {
	// classification still includes files that are completely commented out
	cp, err := a.classifier.Classify(a.prog)
	if err != nil {
		return nil, err
	}

	// build models dictionary
	if a.scanModels {
		for _, modelsFile := range cp.Models {
			if err := a.parseSchema(modelsFile); err != nil {
				return nil, err
			}
		}
	}

	// build parameters dictionary
	for _, paramsFile := range cp.Parameters {
		if err := a.parseParameters(paramsFile); err != nil {
			return nil, err
		}
	}

	// build responses dictionary
	for _, responseFile := range cp.Responses {
		if err := a.parseResponses(responseFile); err != nil {
			return nil, err
		}
	}

	// build definitions dictionary
	if err := a.processDiscovered(); err != nil {
		return nil, err
	}

	// build paths dictionary
	for _, routeFile := range cp.Operations {
		if err := a.parseRoutes(routeFile); err != nil {
			return nil, err
		}
	}

	// build swagger object
	for _, metaFile := range cp.Meta {
		if err := a.parseMeta(metaFile); err != nil {
			return nil, err
		}
	}

	if a.input.Swagger == "" {
		a.input.Swagger = "2.0"
	}

	return a.input, nil
}

func (a *appScanner) processDiscovered() error {
	// loop over discovered until all the items are in definitions
	keepGoing := len(a.discovered) > 0
	for keepGoing {
		var queue []schemaDecl
		for _, d := range a.discovered {
			if _, ok := a.definitions[d.Name]; !ok {
				queue = append(queue, d)
			}
		}
		a.discovered = nil
		for _, sd := range queue {
			if err := a.parseSchema(sd.File); err != nil {
				return err
			}
		}
		keepGoing = len(a.discovered) > 0
	}

	return nil
}

func (a *appScanner) parseSchema(file *ast.File) error {
	sp := newSchemaParser(a.prog)
	if err := sp.Parse(file, a.definitions); err != nil {
		return err
	}
	a.discovered = append(a.discovered, sp.postDecls...)
	return nil
}

func (a *appScanner) parseRoutes(file *ast.File) error {
	rp := newRoutesParser(a.prog)
	rp.operations = a.operations
	rp.definitions = a.definitions
	rp.responses = a.responses
	if err := rp.Parse(file, a.input.Paths); err != nil {
		return err
	}
	return nil
}

func (a *appScanner) parseParameters(file *ast.File) error {
	rp := newParameterParser(a.prog)
	if err := rp.Parse(file, a.operations); err != nil {
		return err
	}
	a.discovered = append(a.discovered, rp.postDecls...)
	a.discovered = append(a.discovered, rp.scp.postDecls...)
	return nil
}

func (a *appScanner) parseResponses(file *ast.File) error {
	rp := newResponseParser(a.prog)
	if err := rp.Parse(file, a.responses); err != nil {
		return err
	}
	a.discovered = append(a.discovered, rp.postDecls...)
	a.discovered = append(a.discovered, rp.scp.postDecls...)
	return nil
}

func (a *appScanner) parseMeta(file *ast.File) error {
	return newMetaParser(a.input).Parse(file.Doc)
}

// MustExpandPackagePath gets the real package path on disk
func (a *appScanner) MustExpandPackagePath(packagePath string) string {
	pkgRealpath := swag.FindInGoSearchPath(packagePath)
	if pkgRealpath == "" {
		log.Fatalf("Can't find package %s \n", packagePath)
	}

	return pkgRealpath
}

type swaggerTypable interface {
	Typed(string, string)
	SetRef(spec.Ref)
	Items() swaggerTypable
	Schema() *spec.Schema
	Level() int
}

func swaggerSchemaForType(typeName string, prop swaggerTypable) error {
	switch typeName {
	case "bool":
		prop.Typed("boolean", "")
	case "error", "rune", "string":
		prop.Typed("string", "")
	case "int8":
		prop.Typed("integer", "int8")
	case "int16":
		prop.Typed("integer", "int16")
	case "int32":
		prop.Typed("integer", "int32")
	case "int", "int64":
		prop.Typed("integer", "int64")
	case "uint8":
		prop.Typed("integer", "uint8")
	case "uint16":
		prop.Typed("integer", "uint16")
	case "uint32":
		prop.Typed("integer", "uint32")
	case "uint", "uint64":
		prop.Typed("integer", "uint64")
	case "float32":
		prop.Typed("number", "float")
	case "float64":
		prop.Typed("number", "double")
	default:
		return fmt.Errorf("unknown primitive %q", typeName)
	}
	return nil
}

func newMultiLineTagParser(name string, parser valueParser) tagParser {
	return tagParser{
		Name:      name,
		MultiLine: true,
		Parser:    parser,
	}
}

func newSingleLineTagParser(name string, parser valueParser) tagParser {
	return tagParser{
		Name:      name,
		MultiLine: false,
		Parser:    parser,
	}
}

type tagParser struct {
	Name      string
	MultiLine bool
	Lines     []string
	Parser    valueParser
}

func (st *tagParser) Matches(line string) bool {
	return st.Parser.Matches(line)
}

func (st *tagParser) Parse(lines []string) error {
	return st.Parser.Parse(lines)
}

// aggregates lines in header until it sees a tag.
type sectionedParser struct {
	header     []string
	matched    map[string]tagParser
	annotation valueParser

	seenTag        bool
	skipHeader     bool
	setTitle       func([]string)
	setDescription func([]string)
	workedOutTitle bool
	taggers        []tagParser
	currentTagger  *tagParser
	title          []string
	description    []string
}

func (st *sectionedParser) cleanup(lines []string) []string {
	// bail early when there is nothing to parse
	if len(lines) == 0 {
		return lines
	}
	seenLine := -1
	var lastContent int
	var uncommented []string
	for i, v := range lines {
		str := regexp.MustCompile(`^[\p{Zs}\t/\*-]*`).ReplaceAllString(v, "")
		uncommented = append(uncommented, str)
		if str != "" {
			if seenLine < 0 {
				seenLine = i
			}
			lastContent = i
		}
	}
	// fixes issue #50
	if seenLine == -1 {
		return nil
	}
	return uncommented[seenLine : lastContent+1]
}

func (st *sectionedParser) collectTitleDescription() {
	if st.workedOutTitle {
		return
	}
	if st.setTitle == nil {
		st.header = st.cleanup(st.header)
		return
	}
	hdrs := st.cleanup(st.header)

	st.workedOutTitle = true
	idx := -1
	for i, line := range hdrs {
		if strings.TrimSpace(line) == "" {
			idx = i
			break
		}
	}

	if idx > -1 {

		st.title = hdrs[:idx]
		if len(hdrs) > idx+1 {
			st.header = hdrs[idx+1:]
		} else {
			st.header = nil
		}
		return
	}

	if len(hdrs) > 0 {
		line := hdrs[0]
		if rxPunctuationEnd.MatchString(line) {
			st.title = []string{line}
			st.header = hdrs[1:]
		} else {
			st.header = hdrs
		}
	}
}

func (st *sectionedParser) Title() []string {
	st.collectTitleDescription()
	return st.title
}

func (st *sectionedParser) Description() []string {
	st.collectTitleDescription()
	return st.header
}

func (st *sectionedParser) Parse(doc *ast.CommentGroup) error {
	if doc == nil {
		return nil
	}
COMMENTS:
	for _, c := range doc.List {
		for _, line := range strings.Split(c.Text, "\n") {
			if rxSwaggerAnnotation.MatchString(line) {
				if st.annotation == nil || !st.annotation.Matches(line) {
					break COMMENTS // a new swagger: annotation terminates this parser
				}

				st.annotation.Parse([]string{line})
				if len(st.header) > 0 {
					st.seenTag = true
				}
				continue
			}

			var matched bool
			for _, tagger := range st.taggers {
				if tagger.Matches(line) {
					st.seenTag = true
					st.currentTagger = &tagger
					matched = true
					break
				}
			}

			if st.currentTagger == nil {
				if !st.skipHeader && !st.seenTag {
					st.header = append(st.header, line)
				}
				// didn't match a tag, moving on
				continue
			}

			if st.currentTagger.MultiLine && matched {
				// the first line of a multiline tagger doesn't count
				continue
			}

			ts, ok := st.matched[st.currentTagger.Name]
			if !ok {
				ts = *st.currentTagger
			}
			ts.Lines = append(ts.Lines, line)
			if st.matched == nil {
				st.matched = make(map[string]tagParser)
			}
			st.matched[st.currentTagger.Name] = ts

			if !st.currentTagger.MultiLine {
				st.currentTagger = nil
			}
		}
	}
	if st.setTitle != nil {
		st.setTitle(st.Title())
	}
	if st.setDescription != nil {
		st.setDescription(st.Description())
	}
	for _, mt := range st.matched {
		if err := mt.Parse(st.cleanup(mt.Lines)); err != nil {
			return err
		}
	}
	return nil
}
