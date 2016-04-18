package generator

import "github.com/go-swagger/go-swagger/spec"

// GenDefinition contains all the properties to generate a
// defintion from a swagger spec
type GenDefinition struct {
	GenSchema
	Package          string
	Imports          map[string]string
	DefaultImports   []string
	ExtraSchemas     []GenSchema
	DependsOn        []string
	IncludeValidator bool
}

// GenSchemaList is a list of schemas for generation.
//
// It can be sorted by name to get a stable struct layout for
// version control and such
type GenSchemaList []GenSchema

func (g GenSchemaList) Len() int           { return len(g) }
func (g GenSchemaList) Swap(i, j int)      { g[i], g[j] = g[j], g[i] }
func (g GenSchemaList) Less(i, j int) bool { return g[i].Name < g[j].Name }

// GenSchema contains all the information needed to generate the code
// for a schema
type GenSchema struct {
	resolvedType
	sharedValidations
	Example                 string
	Name                    string
	Suffix                  string
	Path                    string
	ValueExpression         string
	IndexVar                string
	KeyVar                  string
	Title                   string
	Description             string
	Location                string
	ReceiverName            string
	Items                   *GenSchema
	AllowsAdditionalItems   bool
	HasAdditionalItems      bool
	AdditionalItems         *GenSchema
	Object                  *GenSchema
	XMLName                 string
	Properties              GenSchemaList
	AllOf                   []GenSchema
	HasAdditionalProperties bool
	IsAdditionalProperties  bool
	AdditionalProperties    *GenSchema
	ReadOnly                bool
	IsVirtual               bool
	IsBaseType              bool
	HasBaseType             bool
	IsSubType               bool
	IsExported              bool
	DiscriminatorField      string
	DiscriminatorValue      string
	Discriminates           map[string]string
	Parents                 []string
}

type sharedValidations struct {
	Required            bool
	MaxLength           *int64
	MinLength           *int64
	Pattern             string
	MultipleOf          *float64
	Minimum             *float64
	Maximum             *float64
	ExclusiveMinimum    bool
	ExclusiveMaximum    bool
	Enum                []interface{}
	ItemsEnum           []interface{}
	HasValidations      bool
	MinItems            *int64
	MaxItems            *int64
	UniqueItems         bool
	HasSliceValidations bool
	NeedsSize           bool
	NeedsValidation     bool
	NeedsRequired       bool
}

// GenResponse represents a response object for code generation
type GenResponse struct {
	Package       string
	ModelsPackage string
	ReceiverName  string
	Name          string
	Description   string

	IsSuccess bool

	Code               int
	Method             string
	Path               string
	Headers            GenHeaders
	Schema             *GenSchema
	AllowsForStreaming bool

	Imports        map[string]string
	DefaultImports []string
}

// GenHeader represents a header on a response for code generation
type GenHeader struct {
	resolvedType
	sharedValidations

	Package      string
	ReceiverName string

	Name string
	Path string

	Title       string
	Description string
	Default     interface{}

	Converter string
	Formatter string
}

// GenHeaders is a sorted collection of headers for codegen
type GenHeaders []GenHeader

func (g GenHeaders) Len() int           { return len(g) }
func (g GenHeaders) Swap(i, j int)      { g[i], g[j] = g[j], g[i] }
func (g GenHeaders) Less(i, j int) bool { return g[i].Name < g[j].Name }

// GenParameter is used to represent
// a parameter or a header for code generation.
type GenParameter struct {
	resolvedType
	sharedValidations

	Name            string
	ModelsPackage   string
	Path            string
	ValueExpression string
	IndexVar        string
	ReceiverName    string
	Location        string
	Title           string
	Description     string
	Converter       string
	Formatter       string

	Schema *GenSchema

	CollectionFormat string

	Child  *GenItems
	Parent *GenItems

	BodyParam *GenParameter

	Default         interface{}
	Enum            []interface{}
	ZeroValue       string
	AllowEmptyValue bool
}

// IsQueryParam returns true when this parameter is a query param
func (g *GenParameter) IsQueryParam() bool {
	return g.Location == "query"
}

// IsPathParam returns true when this parameter is a path param
func (g *GenParameter) IsPathParam() bool {
	return g.Location == "path"
}

// IsFormParam returns true when this parameter is a form param
func (g *GenParameter) IsFormParam() bool {
	return g.Location == "formData"
}

// IsHeaderParam returns true when this parameter is a header param
func (g *GenParameter) IsHeaderParam() bool {
	return g.Location == "header"
}

// IsBodyParam returns true when this parameter is a body param
func (g *GenParameter) IsBodyParam() bool {
	return g.Location == "body"
}

// IsFileParam returns true when this parameter is a file param
func (g *GenParameter) IsFileParam() bool {
	return g.SwaggerType == "file"
}

// GenParameters represents a sorted parameter collection
type GenParameters []GenParameter

func (g GenParameters) Len() int           { return len(g) }
func (g GenParameters) Less(i, j int) bool { return g[i].Name < g[j].Name }
func (g GenParameters) Swap(i, j int)      { g[i], g[j] = g[j], g[i] }

// GenItems represents the collection items for a collection parameter
type GenItems struct {
	sharedValidations
	resolvedType

	Name             string
	Path             string
	ValueExpression  string
	CollectionFormat string
	Child            *GenItems
	Parent           *GenItems
	Converter        string
	Formatter        string

	Location string
}

// GenOperationGroup represents a named (tagged) group of operations
type GenOperationGroup struct {
	Name       string
	Operations GenOperations

	Summary        string
	Description    string
	Imports        map[string]string
	DefaultImports []string
	RootPackage    string
	WithContext    bool
}

// GenOperationGroups is a sorted collection of operation groups
type GenOperationGroups []GenOperationGroup

func (g GenOperationGroups) Len() int           { return len(g) }
func (g GenOperationGroups) Swap(i, j int)      { g[i], g[j] = g[j], g[i] }
func (g GenOperationGroups) Less(i, j int) bool { return g[i].Name < g[j].Name }

// GenOperation represents an operation for code generation
type GenOperation struct {
	Package      string
	ReceiverName string
	Name         string
	Summary      string
	Description  string
	Method       string
	Path         string
	Tags         []string
	RootPackage  string

	Imports        map[string]string
	DefaultImports []string
	ExtraSchemas   []GenSchema

	Authorized bool
	Principal  string

	SuccessResponse *GenResponse
	Responses       map[int]GenResponse
	DefaultResponse *GenResponse

	Params         GenParameters
	QueryParams    GenParameters
	PathParams     GenParameters
	HeaderParams   GenParameters
	FormParams     GenParameters
	HasQueryParams bool
	HasFormParams  bool
	HasFileParams  bool

	Schemes            []string
	ExtraSchemes       []string
	ProducesMediaTypes []string
	ConsumesMediaTypes []string
	WithContext        bool
}

// GenOperations represents a list of operations to generate
// this implements a sort by operation id
type GenOperations []GenOperation

func (g GenOperations) Len() int           { return len(g) }
func (g GenOperations) Less(i, j int) bool { return g[i].Name < g[j].Name }
func (g GenOperations) Swap(i, j int)      { g[i], g[j] = g[j], g[i] }

// GenApp represents all the meta data needed to generate an application
// from a swagger spec
type GenApp struct {
	APIPackage          string
	Package             string
	ReceiverName        string
	Name                string
	Principal           string
	DefaultConsumes     string
	DefaultProduces     string
	Host                string
	BasePath            string
	Info                *spec.Info
	ExternalDocs        *spec.ExternalDocumentation
	Imports             map[string]string
	DefaultImports      []string
	Schemes             []string
	ExtraSchemes        []string
	Consumes            []GenSerGroup
	Produces            []GenSerGroup
	SecurityDefinitions []GenSecurityScheme
	Models              []GenDefinition
	Operations          GenOperations
	OperationGroups     GenOperationGroups
	SwaggerJSON         string
	ExcludeSpec         bool
	WithContext         bool
}

// GenSerGroup represents a group of serializers, most likely this is a media type to a list of
// prioritized serializers.
type GenSerGroup struct {
	ReceiverName   string
	AppName        string
	Name           string
	MediaType      string
	Implementation string
	AllSerializers []GenSerializer
}

// GenSerializer represents a single serializer for a particular media type
type GenSerializer struct {
	ReceiverName   string
	AppName        string
	Name           string
	MediaType      string
	Implementation string
}

// GenSecurityScheme represents a security scheme for code generation
type GenSecurityScheme struct {
	AppName      string
	ID           string
	Name         string
	ReceiverName string
	IsBasicAuth  bool
	IsAPIKeyAuth bool
	Source       string
	Principal    string
}
