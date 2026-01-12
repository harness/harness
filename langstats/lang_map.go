// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package langstats

var progLangsByExt = map[string][]string{
	// --- Go ---
	".go": {"Go"},

	// --- Python ---
	".py":  {"Python"},
	".pyi": {"Python"},
	".pyw": {"Python"},

	// --- Java ---
	".java": {"Java"},

	// --- JavaScript / TypeScript ---
	".cjs": {"JavaScript"},
	".js":  {"JavaScript"},
	".jsx": {"JavaScript"},
	".mjs": {"JavaScript"},
	".ts":  {"TypeScript"},
	".tsx": {"TypeScript"},

	// --- C / C++ ---
	".c":   {"C"},
	".cc":  {"C++"},
	".cpp": {"C++"},
	".cxx": {"C++"},
	".h":   {"C", "C++"},
	".hh":  {"C++"},
	".hpp": {"C++"},

	// --- C# ---
	".cs": {"C#"},

	// --- Rust ---
	".rs": {"Rust"},

	// --- Ruby ---
	".rb":   {"Ruby"},
	".rake": {"Ruby"},

	// --- PHP ---
	".php":   {"PHP"},
	".phtml": {"PHP"},

	// --- Kotlin ---
	".kt":  {"Kotlin"},
	".kts": {"Kotlin"},

	// --- Swift / Obj-C ---
	".m":     {"Objective-C", "MATLAB"},
	".mm":    {"Objective-C++"},
	".swift": {"Swift"},

	// --- Scala ---
	".sc":    {"Scala", "SuperCollider"},
	".scala": {"Scala"},

	// --- R / Julia / MATLAB ---
	".jl":     {"Julia"},
	".matlab": {"MATLAB"},
	".r":      {"R"},

	// --- Lua ---
	".lua": {"Lua"},

	// --- Shell ---
	".bash": {"Shell"},
	".sh":   {"Shell"},
	".zsh":  {"Shell"},

	// --- PowerShell ---
	".ps1": {"PowerShell"},

	// --- SQL ---
	".sql": {"SQL"},

	// --- Build / Infra ---
	".cmake":      {"CMake"},
	".dockerfile": {"Dockerfile"},
	".make":       {"Makefile"},
	".mk":         {"Makefile"},

	// --- Web ---
	".css":  {"CSS"},
	".scss": {"SCSS"},
	".less": {"Less"},

	// --- Less common but realistic modern languages ---
	".clj":    {"Clojure"},
	".cljs":   {"Clojure"},
	".d":      {"D"},
	".dart":   {"Dart"},
	".erl":    {"Erlang"},
	".ex":     {"Elixir"},
	".exs":    {"Elixir"},
	".fs":     {"F#"},
	".fsi":    {"F#"},
	".fsx":    {"F#"},
	".groovy": {"Groovy"},
	".gradle": {"Groovy"},
	".hs":     {"Haskell"},
	".hx":     {"Haxe"},
	".ml":     {"OCaml", "Standard ML"},
	".mli":    {"OCaml"},
	".nim":    {"Nim"},
	".pas":    {"Pascal"},
	".v":      {"V", "Verilog"},
	".vala":   {"Vala"},
	".zig":    {"Zig"},

	// --- Traditional languages ---
	".b":   {"B"},
	".bas": {"BASIC"},
	".for": {"Fortran"},
}
