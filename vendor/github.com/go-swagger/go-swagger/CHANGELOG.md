# Change Log

## [Unreleased](https://github.com/go-swagger/go-swagger/tree/HEAD)

[Full Changelog](https://github.com/go-swagger/go-swagger/compare/0.3.0...HEAD)

**Implemented enhancements:**

- Clean up tutorial [\#275](https://github.com/go-swagger/go-swagger/issues/275)

**Fixed bugs:**

- Handle missing Accept: from request header [\#298](https://github.com/go-swagger/go-swagger/issues/298)
- missing import: model not compiling [\#255](https://github.com/go-swagger/go-swagger/issues/255)

**Closed issues:**

- Support for cookies in the client runtime [\#308](https://github.com/go-swagger/go-swagger/issues/308)
- Generated server is misinterpreting request type as application/octet-stream [\#306](https://github.com/go-swagger/go-swagger/issues/306)
- Operation specific "produces" not overriding global "produces" [\#304](https://github.com/go-swagger/go-swagger/issues/304)
- Move things out of the main package [\#302](https://github.com/go-swagger/go-swagger/issues/302)
- `swagger generate client` with `-t` directory target puts client code in unexpected directory [\#230](https://github.com/go-swagger/go-swagger/issues/230)
- support not embedding the schema into the server generated code [\#222](https://github.com/go-swagger/go-swagger/issues/222)
- autodetect swagger base path  [\#120](https://github.com/go-swagger/go-swagger/issues/120)

**Merged pull requests:**

- adds a ResponderFunc helper [\#315](https://github.com/go-swagger/go-swagger/pull/315) ([casualjim](https://github.com/casualjim))
- fixes \#222 server optionally embeds spec [\#314](https://github.com/go-swagger/go-swagger/pull/314) ([casualjim](https://github.com/casualjim))
- 302 reprise [\#313](https://github.com/go-swagger/go-swagger/pull/313) ([casualjim](https://github.com/casualjim))
- anchor strfmt import \(fixes \#255\) [\#312](https://github.com/go-swagger/go-swagger/pull/312) ([casualjim](https://github.com/casualjim))
- 304 consumes produces override [\#311](https://github.com/go-swagger/go-swagger/pull/311) ([casualjim](https://github.com/casualjim))
- 306 octet stream consumer producer [\#310](https://github.com/go-swagger/go-swagger/pull/310) ([casualjim](https://github.com/casualjim))
- Add support for cookies in the client runtime [\#309](https://github.com/go-swagger/go-swagger/pull/309) ([stoyanr](https://github.com/stoyanr))
- fix file type parameter when the parameter is required [\#297](https://github.com/go-swagger/go-swagger/pull/297) ([adasescu](https://github.com/adasescu))
- 275 clean up tutorial [\#296](https://github.com/go-swagger/go-swagger/pull/296) ([andregmoeller](https://github.com/andregmoeller))

## [0.3.0](https://github.com/go-swagger/go-swagger/tree/0.3.0) (2016-02-15)
[Full Changelog](https://github.com/go-swagger/go-swagger/compare/0.2.17...0.3.0)

**Implemented enhancements:**

- Add http/unix socket transport [\#278](https://github.com/go-swagger/go-swagger/issues/278)

**Fixed bugs:**

- Enums are no longer pointers if not required [\#277](https://github.com/go-swagger/go-swagger/issues/277)
- Cannot use "." in route definition [\#271](https://github.com/go-swagger/go-swagger/issues/271)
- Enum generated Validate code has duplicate if checks [\#265](https://github.com/go-swagger/go-swagger/issues/265)
- fields with special chars lose the special chars [\#257](https://github.com/go-swagger/go-swagger/issues/257)
- Panic: interface is spec.Schema, not spec.Parameter for formField file [\#253](https://github.com/go-swagger/go-swagger/issues/253)

**Closed issues:**

- httpkit/client/runtime strips trailing slash from request path causing 301 [\#289](https://github.com/go-swagger/go-swagger/issues/289)
- Generate: No Producer or stub for "text/plain" [\#287](https://github.com/go-swagger/go-swagger/issues/287)
- DELETE method without body [\#264](https://github.com/go-swagger/go-swagger/issues/264)
- clientgen: Properties with "format": "date-time" that are not required are not generated as pointers [\#251](https://github.com/go-swagger/go-swagger/issues/251)
- build cross platform binaries [\#247](https://github.com/go-swagger/go-swagger/issues/247)
- race in Runtime.Submit [\#242](https://github.com/go-swagger/go-swagger/issues/242)
- swagger validate silently returns 0 when input does not exist [\#233](https://github.com/go-swagger/go-swagger/issues/233)
- Unmarshal error returned without further context [\#77](https://github.com/go-swagger/go-swagger/issues/77)

**Merged pull requests:**

- prepare for 0.3.0 release [\#295](https://github.com/go-swagger/go-swagger/pull/295) ([casualjim](https://github.com/casualjim))
- adds unix domain sockets to generated server [\#294](https://github.com/go-swagger/go-swagger/pull/294) ([casualjim](https://github.com/casualjim))
- generate stabler operation names [\#293](https://github.com/go-swagger/go-swagger/pull/293) ([casualjim](https://github.com/casualjim))
- Preserve trailing slash in URL path in runtime [\#292](https://github.com/go-swagger/go-swagger/pull/292) ([jonathaningram](https://github.com/jonathaningram))
- more lenient path matching for routes [\#291](https://github.com/go-swagger/go-swagger/pull/291) ([casualjim](https://github.com/casualjim))
- fixes both issues related to enums [\#290](https://github.com/go-swagger/go-swagger/pull/290) ([casualjim](https://github.com/casualjim))
- add text/plain in server generator [\#288](https://github.com/go-swagger/go-swagger/pull/288) ([casualjim](https://github.com/casualjim))
- transliterate some common special chars [\#286](https://github.com/go-swagger/go-swagger/pull/286) ([casualjim](https://github.com/casualjim))
- adds swagger:file annotation, fixes \#253 [\#285](https://github.com/go-swagger/go-swagger/pull/285) ([casualjim](https://github.com/casualjim))
- use sync.Once for client [\#284](https://github.com/go-swagger/go-swagger/pull/284) ([casualjim](https://github.com/casualjim))
- nullable dates [\#283](https://github.com/go-swagger/go-swagger/pull/283) ([casualjim](https://github.com/casualjim))
- returns a more useful error message [\#282](https://github.com/go-swagger/go-swagger/pull/282) ([casualjim](https://github.com/casualjim))
- readme update for distribution channels [\#281](https://github.com/go-swagger/go-swagger/pull/281) ([casualjim](https://github.com/casualjim))

## [0.2.17](https://github.com/go-swagger/go-swagger/tree/0.2.17) (2016-02-13)
[Full Changelog](https://github.com/go-swagger/go-swagger/compare/0.2.16...0.2.17)

## [0.2.16](https://github.com/go-swagger/go-swagger/tree/0.2.16) (2016-02-13)
[Full Changelog](https://github.com/go-swagger/go-swagger/compare/0.2.15...0.2.16)

## [0.2.15](https://github.com/go-swagger/go-swagger/tree/0.2.15) (2016-02-13)
[Full Changelog](https://github.com/go-swagger/go-swagger/compare/0.2.14...0.2.15)

## [0.2.14](https://github.com/go-swagger/go-swagger/tree/0.2.14) (2016-02-13)
[Full Changelog](https://github.com/go-swagger/go-swagger/compare/0.2.13...0.2.14)

## [0.2.13](https://github.com/go-swagger/go-swagger/tree/0.2.13) (2016-02-13)
[Full Changelog](https://github.com/go-swagger/go-swagger/compare/0.2.12...0.2.13)

## [0.2.12](https://github.com/go-swagger/go-swagger/tree/0.2.12) (2016-02-13)
[Full Changelog](https://github.com/go-swagger/go-swagger/compare/0.2.11...0.2.12)

## [0.2.11](https://github.com/go-swagger/go-swagger/tree/0.2.11) (2016-02-13)
[Full Changelog](https://github.com/go-swagger/go-swagger/compare/0.2.10...0.2.11)

## [0.2.10](https://github.com/go-swagger/go-swagger/tree/0.2.10) (2016-02-13)
[Full Changelog](https://github.com/go-swagger/go-swagger/compare/0.2.9...0.2.10)

## [0.2.9](https://github.com/go-swagger/go-swagger/tree/0.2.9) (2016-02-13)
[Full Changelog](https://github.com/go-swagger/go-swagger/compare/0.2.8...0.2.9)

## [0.2.8](https://github.com/go-swagger/go-swagger/tree/0.2.8) (2016-02-13)
[Full Changelog](https://github.com/go-swagger/go-swagger/compare/0.2.7...0.2.8)

## [0.2.7](https://github.com/go-swagger/go-swagger/tree/0.2.7) (2016-02-13)
[Full Changelog](https://github.com/go-swagger/go-swagger/compare/0.2.6...0.2.7)

## [0.2.6](https://github.com/go-swagger/go-swagger/tree/0.2.6) (2016-02-13)
[Full Changelog](https://github.com/go-swagger/go-swagger/compare/0.2.5...0.2.6)

## [0.2.5](https://github.com/go-swagger/go-swagger/tree/0.2.5) (2016-02-13)
[Full Changelog](https://github.com/go-swagger/go-swagger/compare/0.2.4...0.2.5)

## [0.2.4](https://github.com/go-swagger/go-swagger/tree/0.2.4) (2016-02-13)
[Full Changelog](https://github.com/go-swagger/go-swagger/compare/0.2.3...0.2.4)

**Closed issues:**

- spec generator strips out special characters in the beginning of lines [\#276](https://github.com/go-swagger/go-swagger/issues/276)
- Add Support for Extensions on the root Swagger Document [\#268](https://github.com/go-swagger/go-swagger/issues/268)
- server consuming `application/x-www-form-urlencoded` doesn't work [\#263](https://github.com/go-swagger/go-swagger/issues/263)

**Merged pull requests:**

- scan: strip only special chars that are used in annotations [\#280](https://github.com/go-swagger/go-swagger/pull/280) ([fsouza](https://github.com/fsouza))
- go generate the generator/bindata.go to fix \#263 [\#273](https://github.com/go-swagger/go-swagger/pull/273) ([MStoykov](https://github.com/MStoykov))
- Bugfix for POST Ops not setting Content Length [\#272](https://github.com/go-swagger/go-swagger/pull/272) ([akutz](https://github.com/akutz))
- Add Support for Extensions on the swagger root object. [\#269](https://github.com/go-swagger/go-swagger/pull/269) ([pytlesk4](https://github.com/pytlesk4))
- Fixes \#263. [\#267](https://github.com/go-swagger/go-swagger/pull/267) ([pieter-lazzaro](https://github.com/pieter-lazzaro))

## [0.2.3](https://github.com/go-swagger/go-swagger/tree/0.2.3) (2016-02-09)
[Full Changelog](https://github.com/go-swagger/go-swagger/compare/0.2.2...0.2.3)

**Merged pull requests:**

- make content type optional for delete method [\#266](https://github.com/go-swagger/go-swagger/pull/266) ([casualjim](https://github.com/casualjim))

## [0.2.2](https://github.com/go-swagger/go-swagger/tree/0.2.2) (2016-02-08)
[Full Changelog](https://github.com/go-swagger/go-swagger/compare/0.2.1...0.2.2)

**Merged pull requests:**

- disable bintray publishing for now [\#262](https://github.com/go-swagger/go-swagger/pull/262) ([casualjim](https://github.com/casualjim))
- update release to use tag from args [\#261](https://github.com/go-swagger/go-swagger/pull/261) ([casualjim](https://github.com/casualjim))
- update drone.sec [\#260](https://github.com/go-swagger/go-swagger/pull/260) ([casualjim](https://github.com/casualjim))
- first stab at automated releases through tags [\#259](https://github.com/go-swagger/go-swagger/pull/259) ([casualjim](https://github.com/casualjim))

## [0.2.1](https://github.com/go-swagger/go-swagger/tree/0.2.1) (2016-02-07)
[Full Changelog](https://github.com/go-swagger/go-swagger/compare/v0.2.0...0.2.1)

**Implemented enhancements:**

- Support not embedding swagger schema into generated Go code [\#190](https://github.com/go-swagger/go-swagger/issues/190)
- Add a command to initialize a swagger yaml spec [\#187](https://github.com/go-swagger/go-swagger/issues/187)
- Allow for client templates to be overridden with local versions [\#101](https://github.com/go-swagger/go-swagger/issues/101)
- validation: referencable references [\#16](https://github.com/go-swagger/go-swagger/issues/16)
- support huge file uploads [\#8](https://github.com/go-swagger/go-swagger/issues/8)
- documentation site [\#5](https://github.com/go-swagger/go-swagger/issues/5)

**Fixed bugs:**

- Kubernetes 2.0 Spec [\#239](https://github.com/go-swagger/go-swagger/issues/239)
- Generated models with shared enums do not compile [\#227](https://github.com/go-swagger/go-swagger/issues/227)
- Go swagger does not support plain values in bodies. [\#217](https://github.com/go-swagger/go-swagger/issues/217)
- Generated server should allow for cleanup on shutting down [\#198](https://github.com/go-swagger/go-swagger/issues/198)
- Crash when using swagger generate client [\#197](https://github.com/go-swagger/go-swagger/issues/197)
- Generated BindRequest has call to Validate, even if no validation is specified [\#196](https://github.com/go-swagger/go-swagger/issues/196)
- parameter integers are always send [\#195](https://github.com/go-swagger/go-swagger/issues/195)
- Wrong documentation for swagger:parameters annotation [\#194](https://github.com/go-swagger/go-swagger/issues/194)
- format of parameters leads to bad generated code [\#193](https://github.com/go-swagger/go-swagger/issues/193)
- Validation dereferencing non-pointer [\#186](https://github.com/go-swagger/go-swagger/issues/186)
- Generated Validate method does not dereference pointer [\#182](https://github.com/go-swagger/go-swagger/issues/182)
- Generated Validatator on slice from interface method incorrect [\#181](https://github.com/go-swagger/go-swagger/issues/181)
- Don't set query params if they are nil [\#174](https://github.com/go-swagger/go-swagger/issues/174)

**Closed issues:**

- Nested references in definitions cause failure [\#254](https://github.com/go-swagger/go-swagger/issues/254)
- Server/client with enums generate uncompilable golang [\#252](https://github.com/go-swagger/go-swagger/issues/252)
- array body parameters lead to uncompilable client code [\#249](https://github.com/go-swagger/go-swagger/issues/249)
- Optional query param enums are not validated [\#248](https://github.com/go-swagger/go-swagger/issues/248)
- provide a version command [\#246](https://github.com/go-swagger/go-swagger/issues/246)
- Map as property created as pointer on objects [\#243](https://github.com/go-swagger/go-swagger/issues/243)
- Make generated client use `consumes` in schema for Accept headers, rather than transport consumers [\#235](https://github.com/go-swagger/go-swagger/issues/235)
- Schemes passed into httpkit New ignored [\#228](https://github.com/go-swagger/go-swagger/issues/228)
- Example for generating spec with securityDefinitions? [\#225](https://github.com/go-swagger/go-swagger/issues/225)
- wrong identifier used  in generated code when validating parameter with not valid \(in golang\) identifier [\#223](https://github.com/go-swagger/go-swagger/issues/223)
- Delete requests with bodies cause a runtime error [\#219](https://github.com/go-swagger/go-swagger/issues/219)
- not an issue [\#216](https://github.com/go-swagger/go-swagger/issues/216)
- Empty string not validated in body schema [\#212](https://github.com/go-swagger/go-swagger/issues/212)
- Generated server main is currently always overwritten [\#210](https://github.com/go-swagger/go-swagger/issues/210)
- Allow addition of custom command line options to generated server code [\#207](https://github.com/go-swagger/go-swagger/issues/207)
- Sole TextMarshaler and TextUnmarshaler interfaces are left aside for embedded types [\#205](https://github.com/go-swagger/go-swagger/issues/205)
- Invalid code client generated in Default Parameter constructor [\#201](https://github.com/go-swagger/go-swagger/issues/201)
- Getting io.EOF error returned from successful HTTP response [\#192](https://github.com/go-swagger/go-swagger/issues/192)
- Client: Generated validator does not reference field [\#189](https://github.com/go-swagger/go-swagger/issues/189)
- Random model properties [\#180](https://github.com/go-swagger/go-swagger/issues/180)
- Submit server golang code generator in swagger-codegen. [\#110](https://github.com/go-swagger/go-swagger/issues/110)
- Submit client generator integration to swagger-codegen [\#109](https://github.com/go-swagger/go-swagger/issues/109)

**Merged pull requests:**

- fixes \#252 primitives are not nullable [\#258](https://github.com/go-swagger/go-swagger/pull/258) ([casualjim](https://github.com/casualjim))
- generator: Print unknown models out when gathering models [\#256](https://github.com/go-swagger/go-swagger/pull/256) ([chancez](https://github.com/chancez))
- don't share Params between requests of a Handler [\#240](https://github.com/go-swagger/go-swagger/pull/240) ([MStoykov](https://github.com/MStoykov))
- httpkit/client: Only set the content-type if the body isnt empty [\#238](https://github.com/go-swagger/go-swagger/pull/238) ([chancez](https://github.com/chancez))
- Better custom template solution [\#237](https://github.com/go-swagger/go-swagger/pull/237) ([pieter-lazzaro](https://github.com/pieter-lazzaro))
- Use schemas consumes to determine accept headers [\#236](https://github.com/go-swagger/go-swagger/pull/236) ([chancez](https://github.com/chancez))
- Fix for shared enums [\#234](https://github.com/go-swagger/go-swagger/pull/234) ([casualjim](https://github.com/casualjim))
- fixes generation of bitbucket client [\#231](https://github.com/go-swagger/go-swagger/pull/231) ([casualjim](https://github.com/casualjim))
- fixes \#217 skip validate for impossible types [\#229](https://github.com/go-swagger/go-swagger/pull/229) ([casualjim](https://github.com/casualjim))
- pascalize struct fields before concating them to struct identifiers F… [\#224](https://github.com/go-swagger/go-swagger/pull/224) ([MStoykov](https://github.com/MStoykov))
- \#101 Custom templates [\#221](https://github.com/go-swagger/go-swagger/pull/221) ([pieter-lazzaro](https://github.com/pieter-lazzaro))
- allow delete requests to have bodies [\#220](https://github.com/go-swagger/go-swagger/pull/220) ([azylman](https://github.com/azylman))
- Fix godoc for client code [\#214](https://github.com/go-swagger/go-swagger/pull/214) ([Xe](https://github.com/Xe))
- skip generating server main if it exists: [\#211](https://github.com/go-swagger/go-swagger/pull/211) ([dfuentes](https://github.com/dfuentes))
- Fix import path for operations that lack tags [\#209](https://github.com/go-swagger/go-swagger/pull/209) ([dfuentes](https://github.com/dfuentes))
- Added possibility for custom command line option parsing [\#208](https://github.com/go-swagger/go-swagger/pull/208) ([Tobi042](https://github.com/Tobi042))
- Change strfmt.DateTime and strfmt.Date types to alias [\#206](https://github.com/go-swagger/go-swagger/pull/206) ([aleksandr-vin](https://github.com/aleksandr-vin))
- generate bindata [\#204](https://github.com/go-swagger/go-swagger/pull/204) ([chancez](https://github.com/chancez))
- generator: Use new buffer to avoid reading empty buffer [\#203](https://github.com/go-swagger/go-swagger/pull/203) ([chancez](https://github.com/chancez))
- generator: Take address of default params for non-required, parameters with defaults [\#202](https://github.com/go-swagger/go-swagger/pull/202) ([chancez](https://github.com/chancez))
- FIXES \#193 [\#200](https://github.com/go-swagger/go-swagger/pull/200) ([MStoykov](https://github.com/MStoykov))
- Proposed solution to Issue \#198 [\#199](https://github.com/go-swagger/go-swagger/pull/199) ([Tobi042](https://github.com/Tobi042))

## [v0.2.0](https://github.com/go-swagger/go-swagger/tree/v0.2.0) (2015-12-25)
[Full Changelog](https://github.com/go-swagger/go-swagger/compare/v0.1.0...v0.2.0)

**Implemented enhancements:**

- Add appveyor build [\#153](https://github.com/go-swagger/go-swagger/issues/153)
- Document supported vendor extensions [\#131](https://github.com/go-swagger/go-swagger/issues/131)
- Add documentation for generated server [\#130](https://github.com/go-swagger/go-swagger/issues/130)

**Fixed bugs:**

- Polymorphic/generic subtypes: discriminator getter method, and unmarshal function do not use definition name [\#175](https://github.com/go-swagger/go-swagger/issues/175)
- Spec generator fails for swagger:route that has no tags [\#171](https://github.com/go-swagger/go-swagger/issues/171)
- client: Polymorphic types as parameter generates pointer to interface [\#169](https://github.com/go-swagger/go-swagger/issues/169)
- client should respect default values [\#135](https://github.com/go-swagger/go-swagger/issues/135)
- models: missing optional fields must not be rejected by validators and must have a distinguishable zero value [\#132](https://github.com/go-swagger/go-swagger/issues/132)

**Closed issues:**

- Add server support for default header values in responses [\#172](https://github.com/go-swagger/go-swagger/issues/172)
- doesn't generate the instagram api server [\#170](https://github.com/go-swagger/go-swagger/issues/170)

**Merged pull requests:**

- Refactor strfmt/time tests, add strfmt.NewDateTime function [\#177](https://github.com/go-swagger/go-swagger/pull/177) ([aleksandr-vin](https://github.com/aleksandr-vin))
- Add default value support for response headers, fixes \#172 [\#173](https://github.com/go-swagger/go-swagger/pull/173) ([aleksandr-vin](https://github.com/aleksandr-vin))
- fix test failing on windows. [\#168](https://github.com/go-swagger/go-swagger/pull/168) ([faguirre1](https://github.com/faguirre1))

## [v0.1.0](https://github.com/go-swagger/go-swagger/tree/v0.1.0) (2015-12-14)
**Implemented enhancements:**

- check licenses of dependencies  [\#154](https://github.com/go-swagger/go-swagger/issues/154)
- Empty or duplicate operation ids in codegen [\#134](https://github.com/go-swagger/go-swagger/issues/134)
- no empty names for path parameters [\#128](https://github.com/go-swagger/go-swagger/issues/128)
- Add validation for only body or formdata params [\#127](https://github.com/go-swagger/go-swagger/issues/127)
- Add support for security definitions to server codegen [\#113](https://github.com/go-swagger/go-swagger/issues/113)
- \[scanner\] security schemes [\#112](https://github.com/go-swagger/go-swagger/issues/112)
- \[scanner\] spec generation should fail when a struct is decorated with more than 1 annotation [\#92](https://github.com/go-swagger/go-swagger/issues/92)
- Struct references in operation parameter objects are generated as struct fields instead of pointers. [\#60](https://github.com/go-swagger/go-swagger/issues/60)
- validation: default values must validate against schema [\#18](https://github.com/go-swagger/go-swagger/issues/18)
- validation: referenced objects [\#17](https://github.com/go-swagger/go-swagger/issues/17)
- implement validation: circular ancestry [\#13](https://github.com/go-swagger/go-swagger/issues/13)
- implement validation: duplicate property name declaration [\#12](https://github.com/go-swagger/go-swagger/issues/12)
- Generate a swagger spec from go code [\#3](https://github.com/go-swagger/go-swagger/issues/3)

**Fixed bugs:**

- no code generated to handle unmarshalling a slice of a generic type [\#160](https://github.com/go-swagger/go-swagger/issues/160)
- swagger generate server with `-t` leads to non-compiliable generated code [\#155](https://github.com/go-swagger/go-swagger/issues/155)
- apiKey SecurityDefinitions work only if the header name=security definition name [\#152](https://github.com/go-swagger/go-swagger/issues/152)
- add allowEmptyValue support for a parameter [\#149](https://github.com/go-swagger/go-swagger/issues/149)
- Polymorphic validation code does not invoke generated Getter methods [\#146](https://github.com/go-swagger/go-swagger/issues/146)
- generate commands should work with urls too [\#145](https://github.com/go-swagger/go-swagger/issues/145)
- responses with a body of type interface{} don't render well [\#137](https://github.com/go-swagger/go-swagger/issues/137)
- responses with a schema render lots of extra schemas [\#136](https://github.com/go-swagger/go-swagger/issues/136)
- Validation fails with circular dependency [\#123](https://github.com/go-swagger/go-swagger/issues/123)
- server should have options for SSL when https scheme is present [\#115](https://github.com/go-swagger/go-swagger/issues/115)
- no enum detected for enum properties in combination with allOf [\#107](https://github.com/go-swagger/go-swagger/issues/107)
- Problem with query parameter with type array and collectionFormat: multi [\#106](https://github.com/go-swagger/go-swagger/issues/106)
- \[scanner\] Exported fields typed interface result in an invalid schema [\#93](https://github.com/go-swagger/go-swagger/issues/93)
- make go gettable without authentication [\#89](https://github.com/go-swagger/go-swagger/issues/89)
- client should infer schemes from the spec [\#88](https://github.com/go-swagger/go-swagger/issues/88)
- respect vendor specific media types [\#87](https://github.com/go-swagger/go-swagger/issues/87)
- client generation doesn't use tags [\#86](https://github.com/go-swagger/go-swagger/issues/86)
- client generation doesn't pick up on params from path level [\#85](https://github.com/go-swagger/go-swagger/issues/85)
- Enum model validation broken [\#79](https://github.com/go-swagger/go-swagger/issues/79)
- Client examples? [\#76](https://github.com/go-swagger/go-swagger/issues/76)
- When scanning a response the definition is not ref'ed [\#75](https://github.com/go-swagger/go-swagger/issues/75)
- enum validation for anonymous nested object is missing [\#74](https://github.com/go-swagger/go-swagger/issues/74)
- additional properties: false still gets treated as if it was set to true [\#73](https://github.com/go-swagger/go-swagger/issues/73)
- models composed with discriminators and allOf end up with empty bodies [\#65](https://github.com/go-swagger/go-swagger/issues/65)
- \[validation\] pathItems properties are not strictly validated [\#62](https://github.com/go-swagger/go-swagger/issues/62)
- Go-swagger validation issues [\#61](https://github.com/go-swagger/go-swagger/issues/61)
- Array references are generated as array of structs rather than array of pointers. [\#59](https://github.com/go-swagger/go-swagger/issues/59)
- Generated configure\_{app}.go has missing import and invalid reference [\#56](https://github.com/go-swagger/go-swagger/issues/56)
- bind{Param} function may need casting for UUID [\#55](https://github.com/go-swagger/go-swagger/issues/55)
- Generated imports for models are incorrect [\#54](https://github.com/go-swagger/go-swagger/issues/54)
- Validation:"swagger" field must validate against schema [\#53](https://github.com/go-swagger/go-swagger/issues/53)
- scanner.go panics with index out of bounds  [\#50](https://github.com/go-swagger/go-swagger/issues/50)
- generate spec doesn't add "swagger":2.0 at the beginning of the spec, but it's needed by swagger-ui [\#46](https://github.com/go-swagger/go-swagger/issues/46)
- may be bug on URI parse [\#37](https://github.com/go-swagger/go-swagger/issues/37)
- make generate spec support validations for nested collections [\#22](https://github.com/go-swagger/go-swagger/issues/22)
- make generated server support responses [\#21](https://github.com/go-swagger/go-swagger/issues/21)

**Closed issues:**

- server with valid schema and an extra slash \(/\) does not remove the extra [\#167](https://github.com/go-swagger/go-swagger/issues/167)
- divan/num2words causing `go get` failure [\#166](https://github.com/go-swagger/go-swagger/issues/166)
- Sample swagger.yml generating server fails for boolean, integer, number types in query params [\#163](https://github.com/go-swagger/go-swagger/issues/163)
- Does not support json keys that are numerical [\#162](https://github.com/go-swagger/go-swagger/issues/162)
- Support setting fields on interface/discriminated types [\#158](https://github.com/go-swagger/go-swagger/issues/158)
- Add HTTP/2 support [\#156](https://github.com/go-swagger/go-swagger/issues/156)
- Server does not compile if parameter description is missing [\#148](https://github.com/go-swagger/go-swagger/issues/148)
- Client GenCode tries to access field as method and `func\(\) httpkit.JSONConsumer` not being called. [\#147](https://github.com/go-swagger/go-swagger/issues/147)
- \[scanner\] support discriminators [\#142](https://github.com/go-swagger/go-swagger/issues/142)
- panic: assignment to entry in nil map [\#141](https://github.com/go-swagger/go-swagger/issues/141)
- main.go:XX: handler declared and not used [\#133](https://github.com/go-swagger/go-swagger/issues/133)
- codegen should account for reserved words [\#122](https://github.com/go-swagger/go-swagger/issues/122)
- look into using shippable [\#118](https://github.com/go-swagger/go-swagger/issues/118)
- inline schemas in responses fail to generate [\#116](https://github.com/go-swagger/go-swagger/issues/116)
- Consumers do not handle headers with charset in them [\#114](https://github.com/go-swagger/go-swagger/issues/114)
- Can't get models in the definitions [\#111](https://github.com/go-swagger/go-swagger/issues/111)
- untyped additional properties incorrectly flagged as having validations [\#108](https://github.com/go-swagger/go-swagger/issues/108)
- Inconsistent model method generation [\#105](https://github.com/go-swagger/go-swagger/issues/105)
- Inconsistent import/use of package in frontend\_client.go generation [\#104](https://github.com/go-swagger/go-swagger/issues/104)
- optional strfmt types are not always used by their pointers  [\#103](https://github.com/go-swagger/go-swagger/issues/103)
- Order of the generated operations should be consistent between generations [\#94](https://github.com/go-swagger/go-swagger/issues/94)
- generated server should have annotations to generate a spec [\#90](https://github.com/go-swagger/go-swagger/issues/90)
- Missed 'models' import in generated server when operations declared with 'default' response first [\#84](https://github.com/go-swagger/go-swagger/issues/84)
- various client generation fixes [\#82](https://github.com/go-swagger/go-swagger/issues/82)
- configure file for generated server is missing operations package [\#81](https://github.com/go-swagger/go-swagger/issues/81)
- Schemas without validations don't implement the Validate interface [\#80](https://github.com/go-swagger/go-swagger/issues/80)
- model: template: schemavalidations:78:62: executing "mapvalidator" at \<.AdditionalPropertie...\>: nil pointer evaluating \*generator.GenSchema.HasValidations [\#72](https://github.com/go-swagger/go-swagger/issues/72)
- nil pointer evaluating \*generator.GenSchema.HasValidations [\#71](https://github.com/go-swagger/go-swagger/issues/71)
- Possibility to add any headers [\#70](https://github.com/go-swagger/go-swagger/issues/70)
- Problems getting models in the definitions [\#67](https://github.com/go-swagger/go-swagger/issues/67)
- \[validation\] incorrect validation of path parameters defined in /parameters [\#63](https://github.com/go-swagger/go-swagger/issues/63)
- Validation failed for json swagger if the response schema defined type=array [\#58](https://github.com/go-swagger/go-swagger/issues/58)
- Nil pointer dereference exception while validate an swagger without path [\#52](https://github.com/go-swagger/go-swagger/issues/52)
- generate spec doesn't fill in the description field for responses, which is required. [\#51](https://github.com/go-swagger/go-swagger/issues/51)
- generate spec should remove request body params with the `json:"-"` tag [\#49](https://github.com/go-swagger/go-swagger/issues/49)
- Is there a way to set the "reason" or "response model" fields for a response? [\#48](https://github.com/go-swagger/go-swagger/issues/48)
- generate spec fails if code doesn't compile, breaking app engine support [\#47](https://github.com/go-swagger/go-swagger/issues/47)
- Support for JSON schema cross-file references. [\#45](https://github.com/go-swagger/go-swagger/issues/45)
- Generated fields with `format: "date-time"` have invalid validation code. [\#42](https://github.com/go-swagger/go-swagger/issues/42)
- Security names are mangled in generated `AuthenticatorsFor` method. [\#40](https://github.com/go-swagger/go-swagger/issues/40)
- Ability to generate `all` operations but not the model. [\#38](https://github.com/go-swagger/go-swagger/issues/38)
- Please export common embedded structs as public [\#36](https://github.com/go-swagger/go-swagger/issues/36)
- `strfmt` types should implement database marshaling methods. [\#35](https://github.com/go-swagger/go-swagger/issues/35)
- packages order [\#28](https://github.com/go-swagger/go-swagger/issues/28)
- import package issue [\#27](https://github.com/go-swagger/go-swagger/issues/27)
- Generated Go code fails to transform multiline descriptions [\#23](https://github.com/go-swagger/go-swagger/issues/23)
- go install undefined [\#20](https://github.com/go-swagger/go-swagger/issues/20)
- Add support for the password format [\#7](https://github.com/go-swagger/go-swagger/issues/7)
- Move all the packages to their proper homes [\#4](https://github.com/go-swagger/go-swagger/issues/4)
- Support password extended string format [\#1](https://github.com/go-swagger/go-swagger/issues/1)

**Merged pull requests:**

- Support commonly used x-nullable in addition to x-isnullable [\#157](https://github.com/go-swagger/go-swagger/pull/157) ([chancez](https://github.com/chancez))
- Update README.md [\#138](https://github.com/go-swagger/go-swagger/pull/138) ([frapposelli](https://github.com/frapposelli))
- Fix \#81 missing operations package [\#83](https://github.com/go-swagger/go-swagger/pull/83) ([aleksandr-vin](https://github.com/aleksandr-vin))
- Allow http status 201 to have a response body [\#78](https://github.com/go-swagger/go-swagger/pull/78) ([florindragos](https://github.com/florindragos))
- Ensure operation.Package has a value. [\#69](https://github.com/go-swagger/go-swagger/pull/69) ([jtopjian](https://github.com/jtopjian))
- Some space [\#66](https://github.com/go-swagger/go-swagger/pull/66) ([rgbkrk](https://github.com/rgbkrk))
- validation: show path in path parameter validation errors [\#64](https://github.com/go-swagger/go-swagger/pull/64) ([dolmen](https://github.com/dolmen))
- Fix generated code missing `strfmt` and `httpkit/middleware` imports. [\#43](https://github.com/go-swagger/go-swagger/pull/43) ([chakrit](https://github.com/chakrit))
- sql.Scanner and driver.Valuer implementation `strfmt` types. [\#39](https://github.com/go-swagger/go-swagger/pull/39) ([chakrit](https://github.com/chakrit))
- Fix package path handling on Windows [\#34](https://github.com/go-swagger/go-swagger/pull/34) ([magnushiie](https://github.com/magnushiie))
- Regenerate bindata for server templates. [\#33](https://github.com/go-swagger/go-swagger/pull/33) ([chakrit](https://github.com/chakrit))
- Revert "fix support.go known Consumers/Producers and configureApi func" [\#32](https://github.com/go-swagger/go-swagger/pull/32) ([casualjim](https://github.com/casualjim))
- fix support.go known Consumers/Producers and configureApi func [\#30](https://github.com/go-swagger/go-swagger/pull/30) ([midoblgsm](https://github.com/midoblgsm))
- Fixing package and target order [\#29](https://github.com/go-swagger/go-swagger/pull/29) ([midoblgsm](https://github.com/midoblgsm))
- Replace casualjim with go-swagger in deps and templates [\#26](https://github.com/go-swagger/go-swagger/pull/26) ([gbjk](https://github.com/gbjk))
- Fix -target path injected at end of operations instead of beginning [\#25](https://github.com/go-swagger/go-swagger/pull/25) ([gbjk](https://github.com/gbjk))
- Replace casualjim with go-swagger in imports [\#24](https://github.com/go-swagger/go-swagger/pull/24) ([gbjk](https://github.com/gbjk))
- Add a Gitter chat badge to README.md [\#2](https://github.com/go-swagger/go-swagger/pull/2) ([gitter-badger](https://github.com/gitter-badger))



\* *This Change Log was automatically generated by [github_changelog_generator](https://github.com/skywinder/Github-Changelog-Generator)*