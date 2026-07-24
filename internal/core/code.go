// Copyright (C) 2026 Godynheil A. Quisto <godynheil@quisto.ph>
// SPDX-License-Identifier: AGPL-3.0-or-later
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package core

import (
	"path/filepath"
	"strings"
)

var codeExtensions = map[string]string{
	".R":           "r",
	".asm":         "asm",
	".astro":       "astro",
	".bash":        "shell",
	".bat":         "batch",
	".c":           "c",
	".cbl":         "cobol",
	".clj":         "clojure",
	".cljs":        "clojure",
	".cljc":        "clojure",
	".cmd":         "batch",
	".cob":         "cobol",
	".cpp":         "cpp",
	".cr":          "crystal",
	".cs":          "csharp",
	".css":         "css",
	".dart":        "dart",
	".d":           "d",
	".dockerfile":  "dockerfile",
	".env":         "dotenv",
	".erl":         "erlang",
	".ex":          "elixir",
	".exs":         "elixir",
	".f03":         "fortran",
	".f90":         "fortran",
	".f95":         "fortran",
	".fish":        "shell",
	".gd":          "gdscript",
	".gdextension": "gdextension",
	".gdnlib":      "gdnative",
	".gdns":        "gdnative",
	".go":          "go",
	".gql":         "graphql",
	".gradle":      "groovy",
	".graphql":     "graphql",
	".groovy":      "groovy",
	".h":           "c",
	".hpp":         "cpp",
	".hrl":         "erlang",
	".hs":          "haskell",
	".html":        "html",
	".ini":         "ini",
	".ipynb":       "jupyter",
	".js":          "javascript",
	".jsx":         "javascript",
	".java":        "java",
	".jl":          "julia",
	".json":        "json",
	".kt":          "kotlin",
	".lhs":         "haskell",
	".lock":        "toml",
	".lua":         "lua",
	".m":           "matlab",
	".md":          "markdown",
	".ml":          "ocaml",
	".mli":         "ocaml",
	".nim":         "nim",
	".php":         "php",
	".pl":          "perl",
	".pm":          "perl",
	".pp":          "pascal",
	".proto":       "protobuf",
	".ps1":         "powershell",
	".py":          "python",
	".pas":         "pascal",
	".txt":         "text",
	".csv":         "text",
	".tsv":         "text",
	".log":         "text",
	".rtf":         "text",
	".tex":         "text",
	".rst":         "text",
	".adoc":        "text",
	".asciidoc":    "text",
	".norg":        "text",
	".org":         "text",
	".properties":  "text",
	".cfg":         "text",
	".conf":        "text",
	".nix":         "text",
	".key":         "text",
	".pem":         "text",
	".crt":         "text",
	".csr":         "text",
	".pub":         "text",
	".r":           "r",
	".rb":          "ruby",
	".rs":          "rust",
	".s":           "asm",
	".sbt":         "scala",
	".scala":       "scala",
	".sh":          "shell",
	".sql":         "sql",
	".svg":         "xml",
	".svelte":      "svelte",
	".swift":       "swift",
	".tf":          "terraform",
	".tfvars":      "terraform",
	".toml":        "toml",
	".tres":        "godot-resource",
	".ts":          "typescript",
	".tscn":        "godot-scene",
	".tsx":         "typescript",
	".v":           "v",
	".vim":         "vim",
	".vue":         "vue",
	".xml":         "xml",
	".yaml":        "yaml",
	".yml":         "yaml",
	".zig":         "zig",
	".mod":         "go",
	".sum":         "go",
	".work":        "go",
	".gitmodules":  "text",
	".hlsl":        "hlsl",
	".glsl":        "glsl",
	".usf":         "hlsl",
	".ush":         "hlsl",
	".shader":      "shader",
	".gdshader":    "gdshader",
	".cginc":       "hlsl",
	".cg":          "hlsl",
	".fx":          "hlsl",
	".meta":        "yaml",
	".asmdef":      "json",
	".asmref":      "json",
	".uproject":    "json",
	".uplugin":     "json",
	".import":      "ini",
	".cmake":       "cmake",
	".make":        "makefile",
	".build":       "xml",
	".plist":       "xml",
	".pbxproj":     "text",
}

var codeFilenames = map[string]string{
	"dockerfile": "dockerfile",
	"makefile":   "text",
}

func CodeLanguage(path string) (string, bool) {
	base := strings.ToLower(filepath.Base(path))
	if language, ok := codeFilenames[base]; ok {
		return language, true
	}
	if base == "license" || base == "licence" || base == "copying" || base == "readme" || base == "terms" || base == "privacy" || base == "authors" || base == "contributors" {
		return "text", true
	}
	if strings.HasPrefix(base, ".") {
		return "text", true
	}
	ext := strings.ToLower(filepath.Ext(path))
	language, ok := codeExtensions[ext]
	return language, ok
}
