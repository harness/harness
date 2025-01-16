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

package types

import "github.com/harness/gitness/types/enum"

type JetBrainsIDEDownloadURLTemplates struct {
	Version  string
	Amd64Sha string
	Arm64Sha string
	Amd64    string
	Arm64    string
}

var JetBrainsIDEDownloadURLTemplateMap = map[enum.IDEType]JetBrainsIDEDownloadURLTemplates{
	enum.IDETypeIntelliJ: {
		// list of versions: https://www.jetbrains.com/idea/download/other.html
		Version: "2024.3.1.1",
		Amd64:   "https://download.jetbrains.com/idea/ideaIU-%s.tar.gz",
		Arm64:   "https://download.jetbrains.com/idea/ideaIU-%s-aarch64.tar.gz",
	},
	enum.IDETypeGoland: {
		// list of versions: https://www.jetbrains.com/go/download/other.html
		Version: "2024.3.1",
		Amd64:   "https://download.jetbrains.com/go/goland-%s.tar.gz",
		Arm64:   "https://download.jetbrains.com/go/goland-%s-aarch64.tar.gz",
	},
	enum.IDETypePyCharm: {
		// list of versions: https://www.jetbrains.com/pycharm/download/other.html
		Version: "2024.3.1.1",
		Amd64:   "https://download.jetbrains.com/python/pycharm-professional-%s.tar.gz",
		Arm64:   "https://download.jetbrains.com/python/pycharm-professional-%s-aarch64.tar.gz",
	},
	enum.IDETypeWebStorm: {
		// list of versions: https://www.jetbrains.com/webstorm/download/other.html
		Version: "2024.3.1.1",
		Amd64:   "https://download.jetbrains.com/webstorm/WebStorm-%s.tar.gz",
		Arm64:   "https://download.jetbrains.com/webstorm/WebStorm-%s-aarch64.tar.gz",
	},
	enum.IDETypeCLion: {
		// list of versions: https://www.jetbrains.com/clion/download/other.html
		Version: "2024.3.1.1",
		Amd64:   "https://download.jetbrains.com/cpp/CLion-%s.tar.gz",
		Arm64:   "https://download.jetbrains.com/cpp/CLion-%s-aarch64.tar.gz",
	},
	enum.IDETypePHPStorm: {
		// list of versions: https://www.jetbrains.com/phpstorm/download/other.html
		Version: "2024.3.1.1",
		Amd64:   "https://download.jetbrains.com/webide/PhpStorm-%s.tar.gz",
		Arm64:   "https://download.jetbrains.com/webide/PhpStorm-%s-aarch64.tar.gz",
	},
	enum.IDETypeRubyMine: {
		// list of versions: https://www.jetbrains.com/ruby/download/other.html
		Version: "2024.3.1.1",
		Amd64:   "https://download.jetbrains.com/ruby/RubyMine-%s.tar.gz",
		Arm64:   "https://download.jetbrains.com/ruby/RubyMine-%s-aarch64.tar.gz",
	},
	enum.IDETypeRider: {
		// list of versions: https://www.jetbrains.com/ruby/download/other.html
		Version: "2024.3.3",
		Amd64:   "https://download.jetbrains.com/rider/JetBrains.Rider-%s.tar.gz",
		Arm64:   "https://download.jetbrains.com/rider/JetBrains.Rider-%s-aarch64.tar.gz",
	},
}

type JetBrainsSpecs struct {
	IDEType      enum.IDEType
	DownloadURls JetBrainsIDEDownloadURLTemplates
}
