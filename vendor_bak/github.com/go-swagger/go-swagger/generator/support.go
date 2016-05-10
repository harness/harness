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

package generator

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/go-swagger/go-swagger/httpkit"
	"github.com/go-swagger/go-swagger/spec"
	"github.com/go-swagger/go-swagger/swag"
)

// GenerateServer generates a server application
func GenerateServer(name string, modelNames, operationIDs []string, opts GenOpts) error {
	generator, err := newAppGenerator(name, modelNames, operationIDs, &opts)
	if err != nil {
		return err
	}
	return generator.Generate()
}

// GenerateSupport generates the supporting files for an API
func GenerateSupport(name string, modelNames, operationIDs []string, opts GenOpts) error {

	generator, err := newAppGenerator(name, modelNames, operationIDs, &opts)
	if err != nil {
		return err
	}
	return generator.GenerateSupport(nil)
}

func newAppGenerator(name string, modelNames, operationIDs []string, opts *GenOpts) (*appGenerator, error) {

	if opts.TemplateDir != "" {
		if err := templates.LoadDir(opts.TemplateDir); err != nil {
			return nil, err
		}
	}

	compileTemplates()

	// Load the spec
	_, specDoc, err := loadSpec(opts.Spec)
	if err != nil {
		return nil, err
	}

	models, err := gatherModels(specDoc, modelNames)
	if err != nil {
		return nil, err
	}
	operations := gatherOperations(specDoc, operationIDs)
	if len(operations) == 0 {
		return nil, errors.New("no operations were selected")
	}

	defaultScheme := opts.DefaultScheme
	if defaultScheme == "" {
		defaultScheme = "http"
	}
	defaultProduces := opts.DefaultProduces
	if defaultProduces == "" {
		defaultProduces = "application/json"
	}

	apiPackage := mangleName(swag.ToFileName(opts.APIPackage), "api")
	return &appGenerator{
		Name:       appNameOrDefault(specDoc, name, "swagger"),
		Receiver:   "o",
		SpecDoc:    specDoc,
		Models:     models,
		Operations: operations,
		Target:     opts.Target,
		// Package:       filepath.Base(opts.Target),
		DumpData:        opts.DumpData,
		Package:         apiPackage,
		APIPackage:      apiPackage,
		ModelsPackage:   mangleName(swag.ToFileName(opts.ModelPackage), "definitions"),
		ServerPackage:   mangleName(swag.ToFileName(opts.ServerPackage), "server"),
		ClientPackage:   mangleName(swag.ToFileName(opts.ClientPackage), "client"),
		Principal:       opts.Principal,
		DefaultScheme:   defaultScheme,
		DefaultProduces: defaultProduces,
		GenOpts:         opts,
	}, nil
}

type appGenerator struct {
	Name            string
	Receiver        string
	SpecDoc         *spec.Document
	Package         string
	APIPackage      string
	ModelsPackage   string
	ServerPackage   string
	ClientPackage   string
	Principal       string
	Models          map[string]spec.Schema
	Operations      map[string]opRef
	Target          string
	DumpData        bool
	DefaultScheme   string
	DefaultProduces string
	GenOpts         *GenOpts
}

func baseImport(tgt string) string {
	p, err := filepath.Abs(tgt)
	if err != nil {
		log.Fatalln(err)
	}

	var pth string
	for _, gp := range filepath.SplitList(os.Getenv("GOPATH")) {
		pp := filepath.Join(gp, "src")
		if strings.HasPrefix(p, pp) {
			pth, err = filepath.Rel(pp, p)
			if err != nil {
				log.Fatalln(err)
			}
			break
		}
	}

	if pth == "" {
		log.Fatalln("target must reside inside a location in the $GOPATH/src")
	}
	return pth
}

func (a *appGenerator) Generate() error {

	app, err := a.makeCodegenApp()
	if err != nil {
		return err
	}

	if a.DumpData {
		bb, _ := json.MarshalIndent(swag.ToDynamicJSON(app), "", "  ")
		fmt.Fprintln(os.Stdout, string(bb))
		return nil
	}

	if a.GenOpts.IncludeModel {
		log.Printf("rendering %d models", len(app.Models))
		for _, mod := range app.Models {
			mod.IncludeValidator = true // a.GenOpts.IncludeValidator
			gen := &definitionGenerator{
				Name:    mod.Name,
				SpecDoc: a.SpecDoc,
				Target:  filepath.Join(a.Target, a.ModelsPackage),
				Data:    &mod,
			}
			if err := gen.generateModel(); err != nil {
				return err
			}
		}
	}

	if a.GenOpts.IncludeHandler {
		for _, opg := range app.OperationGroups {
			for _, op := range opg.Operations {
				gen := &opGen{
					data:              &op,
					pkg:               opg.Name,
					cname:             swag.ToGoName(op.Name),
					IncludeHandler:    a.GenOpts.IncludeHandler,
					IncludeParameters: a.GenOpts.IncludeParameters,
					IncludeResponses:  a.GenOpts.IncludeResponses,
					Doc:               a.SpecDoc,
					Target:            filepath.Join(a.Target, a.ServerPackage),
					APIPackage:        a.APIPackage,
				}

				if err := gen.Generate(); err != nil {
					return err
				}
			}
		}
	}

	if a.GenOpts.IncludeSupport {
		return a.GenerateSupport(&app)
	}
	return nil
}

func (a *appGenerator) GenerateSupport(ap *GenApp) error {
	var app *GenApp
	app = ap
	if ap == nil {
		ca, err := a.makeCodegenApp()
		if err != nil {
			return err
		}
		app = &ca
	}

	if a.GenOpts == nil || !a.GenOpts.ExcludeSpec {
		if err := a.generateEmbeddedSwaggerJSON(app); err != nil {
			return err
		}
	}

	importPath := filepath.ToSlash(filepath.Join(baseImport(a.Target), a.ServerPackage, a.APIPackage))
	app.DefaultImports = append(
		app.DefaultImports,
		filepath.ToSlash(filepath.Join(baseImport(a.Target), a.ServerPackage)),
		importPath,
	)

	if err := a.generateAPIBuilder(app); err != nil {
		return err
	}

	if err := a.generateAPIServer(app); err != nil {
		return err
	}

	if err := a.generateConfigureAPI(app); err != nil {
		return err
	}

	if err := a.generateDoc(app); err != nil {
		return err
	}

	if a.GenOpts == nil || a.GenOpts.IncludeMain {
		if err := a.generateMain(app); err != nil {
			return err
		}
	}

	return nil
}

func (a *appGenerator) generateConfigureAPI(app *GenApp) error {
	pth := filepath.Join(a.Target, app.APIPackage)
	nm := "Configure" + swag.ToGoName(app.Name)
	if fileExists(pth, nm) {
		log.Println("skipped (already exists) configure api template:", app.Package+".Configure"+swag.ToGoName(app.Name))
		return nil
	}

	buf := bytes.NewBuffer(nil)
	if err := configureAPITemplate.Execute(buf, app); err != nil {
		return err
	}
	log.Println("rendered configure api template:", app.Package+".Configure"+swag.ToGoName(app.Name))
	return writeToFileIfNotExist(pth, nm, buf.Bytes())
}

func (a *appGenerator) generateMain(app *GenApp) error {
	pth := filepath.Join(a.Target, "cmd", swag.ToCommandName(swag.ToGoName(app.Name)+"Server"))
	if fileExists(pth, "main") && !a.GenOpts.IncludeMain {
		log.Println("skipped (already exists) main template:", app.Package+".Main")
		return nil
	}
	buf := bytes.NewBuffer(nil)
	if err := mainTemplate.Execute(buf, app); err != nil {
		return err
	}
	log.Println("rendered main template:", "server."+swag.ToGoName(app.Name))
	return writeToFile(pth, "main", buf.Bytes())
}

func (a *appGenerator) generateEmbeddedSwaggerJSON(app *GenApp) error {
	buf := bytes.NewBuffer(nil)
	appc := *app
	appc.Package = app.APIPackage
	if err := embeddedSpecTemplate.Execute(buf, &appc); err != nil {
		return err
	}
	log.Println("rendered embedded Swagger JSON template:", app.APIPackage+"."+swag.ToGoName(app.Name))
	return writeToFile(filepath.Join(a.Target, a.ServerPackage), "embedded_spec", buf.Bytes())
}

func (a *appGenerator) generateAPIBuilder(app *GenApp) error {
	buf := bytes.NewBuffer(nil)
	if err := builderTemplate.Execute(buf, app); err != nil {
		return err
	}
	log.Println("rendered builder template:", app.Package+"."+swag.ToGoName(app.Name))
	return writeToFile(filepath.Join(a.Target, a.ServerPackage, app.Package), swag.ToGoName(app.Name)+"Api", buf.Bytes())
}

func (a *appGenerator) generateAPIServer(app *GenApp) error {
	buf := bytes.NewBuffer(nil)
	if err := serverTemplate.Execute(buf, app); err != nil {
		return err
	}
	log.Println("rendered server template:", app.APIPackage+".Server")
	return writeToFile(filepath.Join(a.Target, a.ServerPackage), "Server", buf.Bytes())
}

func (a *appGenerator) generateDoc(app *GenApp) error {
	buf := bytes.NewBuffer(nil)
	if err := mainDocTemplate.Execute(buf, app); err != nil {
		return err
	}
	log.Println("rendered doc template:", app.Package+"."+swag.ToGoName(app.Name))
	return writeToFile(filepath.Join(a.Target, a.ServerPackage), "Doc", buf.Bytes())
}

var mediaTypeNames = map[*regexp.Regexp]string{
	regexp.MustCompile("application/.*json"):         "json",
	regexp.MustCompile("application/.*yaml"):         "yaml",
	regexp.MustCompile("application/.*protobuf"):     "protobuf",
	regexp.MustCompile("application/.*capnproto"):    "capnproto",
	regexp.MustCompile("application/.*thrift"):       "thrift",
	regexp.MustCompile("(?:application|text)/.*xml"): "xml",
	regexp.MustCompile("text/.*markdown"):            "markdown",
	regexp.MustCompile("text/.*html"):                "html",
	regexp.MustCompile("text/.*csv"):                 "csv",
	regexp.MustCompile("text/.*tsv"):                 "tsv",
	regexp.MustCompile("text/.*javascript"):          "js",
	regexp.MustCompile("text/.*css"):                 "css",
	regexp.MustCompile("text/.*plain"):               "txt",
	regexp.MustCompile("application/.*octet-stream"): "bin",
	regexp.MustCompile("application/.*tar"):          "tar",
	regexp.MustCompile("application/.*gzip"):         "gzip",
	regexp.MustCompile("application/.*gz"):           "gzip",
}

var knownProducers = map[string]string{
	"json": "httpkit.JSONProducer",
	"yaml": "httpkit.YAMLProducer",
	"bin":  "httpkit.ByteStreamProducer",
}

var knownConsumers = map[string]string{
	"json": "httpkit.JSONConsumer",
	"yaml": "httpkit.YAMLConsumer",
	"bin":  "httpkit.ByteStreamConsumer",
}

func getSerializer(sers []GenSerGroup, ext string) (*GenSerGroup, bool) {
	for i := range sers {
		s := &sers[i]
		if s.Name == ext {
			return s, true
		}
	}
	return nil, false
}

func mediaTypeName(tn string) (string, bool) {
	for k, v := range mediaTypeNames {
		if k.MatchString(tn) {
			return v, true
		}
	}
	return "", false
}

func (a *appGenerator) makeConsumes() (consumes []GenSerGroup, consumesJSON bool) {
	for _, cons := range a.SpecDoc.RequiredConsumes() {
		cn, ok := mediaTypeName(cons)
		if !ok {
			continue
		}
		nm := swag.ToJSONName(cn)
		if nm == "json" {
			consumesJSON = true
		}

		if ser, ok := getSerializer(consumes, cn); ok {
			ser.AllSerializers = append(ser.AllSerializers, GenSerializer{
				AppName:        ser.AppName,
				ReceiverName:   ser.ReceiverName,
				Name:           ser.Name,
				MediaType:      cons,
				Implementation: knownConsumers[nm],
			})
			continue
		}

		ser := GenSerializer{
			AppName:        a.Name,
			ReceiverName:   a.Receiver,
			Name:           nm,
			MediaType:      cons,
			Implementation: knownConsumers[nm],
		}

		consumes = append(consumes, GenSerGroup{
			AppName:        ser.AppName,
			ReceiverName:   ser.ReceiverName,
			Name:           ser.Name,
			MediaType:      cons,
			AllSerializers: []GenSerializer{ser},
			Implementation: ser.Implementation,
		})
	}
	if len(consumes) == 0 {
		consumes = append(consumes, GenSerGroup{
			AppName:      a.Name,
			ReceiverName: a.Receiver,
			Name:         "json",
			MediaType:    httpkit.JSONMime,
			AllSerializers: []GenSerializer{GenSerializer{
				AppName:        a.Name,
				ReceiverName:   a.Receiver,
				Name:           "json",
				MediaType:      httpkit.JSONMime,
				Implementation: knownConsumers["json"],
			}},
			Implementation: knownConsumers["json"],
		})
		consumesJSON = true
	}
	return
}

func (a *appGenerator) makeProduces() (produces []GenSerGroup, producesJSON bool) {
	for _, prod := range a.SpecDoc.RequiredProduces() {
		pn, ok := mediaTypeName(prod)
		if !ok {
			continue
		}
		nm := swag.ToJSONName(pn)
		if nm == "json" {
			producesJSON = true
		}

		if ser, ok := getSerializer(produces, pn); ok {
			ser.AllSerializers = append(ser.AllSerializers, GenSerializer{
				AppName:        ser.AppName,
				ReceiverName:   ser.ReceiverName,
				Name:           ser.Name,
				MediaType:      prod,
				Implementation: knownProducers[nm],
			})
			continue
		}
		ser := GenSerializer{
			AppName:        a.Name,
			ReceiverName:   a.Receiver,
			Name:           nm,
			MediaType:      prod,
			Implementation: knownProducers[nm],
		}
		produces = append(produces, GenSerGroup{
			AppName:        ser.AppName,
			ReceiverName:   ser.ReceiverName,
			Name:           ser.Name,
			MediaType:      prod,
			Implementation: ser.Implementation,
			AllSerializers: []GenSerializer{ser},
		})
	}
	if len(produces) == 0 {
		produces = append(produces, GenSerGroup{
			AppName:      a.Name,
			ReceiverName: a.Receiver,
			Name:         "json",
			MediaType:    httpkit.JSONMime,
			AllSerializers: []GenSerializer{GenSerializer{
				AppName:        a.Name,
				ReceiverName:   a.Receiver,
				Name:           "json",
				MediaType:      httpkit.JSONMime,
				Implementation: knownProducers["json"],
			}},
			Implementation: knownProducers["json"],
		})
		producesJSON = true
	}

	return
}

func (a *appGenerator) makeSecuritySchemes() (security []GenSecurityScheme) {

	prin := a.Principal
	if prin == "" {
		prin = "interface{}"
	}
	for _, scheme := range a.SpecDoc.RequiredSecuritySchemes() {
		if req, ok := a.SpecDoc.Spec().SecurityDefinitions[scheme]; ok {
			if req.Type == "basic" || req.Type == "apiKey" {
				security = append(security, GenSecurityScheme{
					AppName:      a.Name,
					ID:           scheme,
					ReceiverName: a.Receiver,
					Name:         req.Name,
					IsBasicAuth:  strings.ToLower(req.Type) == "basic",
					IsAPIKeyAuth: strings.ToLower(req.Type) == "apikey",
					Principal:    prin,
					Source:       req.In,
				})
			}
		}
	}

	return
}

func (a *appGenerator) makeCodegenApp() (GenApp, error) {
	log.Println("building a plan for generation")
	sw := a.SpecDoc.Spec()
	receiver := a.Receiver

	var defaultImports []string

	jsonb, _ := json.MarshalIndent(sw, "", "  ")

	consumes, _ := a.makeConsumes()
	produces, _ := a.makeProduces()

	prin := a.Principal
	if prin == "" {
		prin = "interface{}"
	}
	security := a.makeSecuritySchemes()

	var genMods []GenDefinition
	importPath := filepath.ToSlash(filepath.Join(baseImport(a.Target), a.ModelsPackage))
	defaultImports = append(defaultImports, importPath)

	log.Println("planning definitions")
	for mn, m := range a.Models {
		mod, err := makeGenDefinition(
			mn,
			a.ModelsPackage,
			m,
			a.SpecDoc,
		)
		if err != nil {
			return GenApp{}, err
		}
		//mod.ReceiverName = receiver
		genMods = append(genMods, *mod)
	}

	log.Println("planning operations")
	tns := make(map[string]struct{})
	var genOps GenOperations
	for on, opp := range a.Operations {
		o := opp.Op
		o.ID = on
		var bldr codeGenOpBuilder
		bldr.ModelsPackage = a.ModelsPackage
		bldr.Principal = prin
		bldr.Target = a.Target
		bldr.DefaultImports = defaultImports
		bldr.DefaultScheme = a.DefaultScheme
		bldr.Doc = &(*a.SpecDoc)
		// TODO: change operation name to something safe
		bldr.Name = on
		bldr.Operation = *o
		bldr.Method = opp.Method
		bldr.Path = opp.Path
		bldr.Authed = len(a.SpecDoc.SecurityRequirementsFor(o)) > 0
		ap := a.APIPackage
		bldr.RootAPIPackage = swag.ToFileName(a.APIPackage)
		bldr.WithContext = a.GenOpts != nil && a.GenOpts.WithContext
		if len(o.Tags) > 0 {
			for _, tag := range o.Tags {
				tns[tag] = struct{}{}
				bldr.APIPackage = mangleName(swag.ToFileName(tag), a.APIPackage)
				op, err := bldr.MakeOperation()
				if err != nil {
					return GenApp{}, err
				}
				op.ReceiverName = receiver
				genOps = append(genOps, op)
			}
		} else {
			bldr.APIPackage = swag.ToFileName(ap)
			op, err := bldr.MakeOperation()
			if err != nil {
				return GenApp{}, err
			}
			op.ReceiverName = receiver
			genOps = append(genOps, op)
		}
	}
	for k := range tns {
		importPath := filepath.ToSlash(filepath.Join(baseImport(a.Target), a.ServerPackage, a.APIPackage, swag.ToFileName(k)))
		defaultImports = append(defaultImports, importPath)
	}
	sort.Sort(genOps)

	log.Println("grouping operations into packages")
	opsGroupedByTag := make(map[string]GenOperations)
	for _, operation := range genOps {
		if operation.Package == "" {
			operation.Package = a.Package
		}
		opsGroupedByTag[operation.Package] = append(opsGroupedByTag[operation.Package], operation)
	}

	var opGroups GenOperationGroups
	for k, v := range opsGroupedByTag {
		sort.Sort(v)
		opGroup := GenOperationGroup{
			Name:           k,
			Operations:     v,
			DefaultImports: []string{filepath.ToSlash(filepath.Join(baseImport(a.Target), a.ModelsPackage))},
			RootPackage:    a.APIPackage,
			WithContext:    a.GenOpts != nil && a.GenOpts.WithContext,
		}
		opGroups = append(opGroups, opGroup)
		var importPath string
		if k == a.APIPackage {
			importPath = filepath.ToSlash(filepath.Join(baseImport(a.Target), a.ServerPackage, a.APIPackage))
		} else {
			importPath = filepath.ToSlash(filepath.Join(baseImport(a.Target), a.ServerPackage, a.APIPackage, k))
		}
		defaultImports = append(defaultImports, importPath)
	}
	sort.Sort(opGroups)

	log.Println("planning meta data and facades")
	defaultConsumes := "application/json"
	rc := a.SpecDoc.RequiredConsumes()
	if len(rc) > 0 {
		defaultConsumes = rc[0]
	}

	defaultProduces := "application/json"
	rp := a.SpecDoc.RequiredProduces()
	if len(rp) > 0 {
		defaultProduces = rp[0]
	}

	var collectedSchemes []string
	var extraSchemes []string
	for _, op := range genOps {
		collectedSchemes = concatUnique(collectedSchemes, op.Schemes)
		extraSchemes = concatUnique(extraSchemes, op.ExtraSchemes)
	}

	host := "localhost"
	if sw.Host != "" {
		host = sw.Host
	}

	basePath := "/"
	if sw.BasePath != "" {
		basePath = "/"
	}

	return GenApp{
		APIPackage:          a.ServerPackage,
		Package:             a.Package,
		ReceiverName:        receiver,
		Name:                a.Name,
		Host:                host,
		BasePath:            basePath,
		Schemes:             schemeOrDefault(collectedSchemes, a.DefaultScheme),
		ExtraSchemes:        extraSchemes,
		ExternalDocs:        sw.ExternalDocs,
		Info:                sw.Info,
		Consumes:            consumes,
		Produces:            produces,
		DefaultConsumes:     defaultConsumes,
		DefaultProduces:     defaultProduces,
		DefaultImports:      defaultImports,
		SecurityDefinitions: security,
		Models:              genMods,
		Operations:          genOps,
		OperationGroups:     opGroups,
		Principal:           prin,
		SwaggerJSON:         fmt.Sprintf("%#v", jsonb),
		ExcludeSpec:         a.GenOpts != nil && a.GenOpts.ExcludeSpec,
		WithContext:         a.GenOpts != nil && a.GenOpts.WithContext,
	}, nil
}
