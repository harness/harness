/*
 * Copyright 2024 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

export const LANGUAGE_COLORS: Record<string, string> = {
  Go: '#00ADD8',
  Rust: '#000000',
  C: '#A8B9CC',
  'C++': '#00599C',
  'C#': '#239120',
  D: '#B03931',
  Zig: '#F7A41D',
  V: '#5D87BF',
  Nim: '#FFE953',
  Java: '#B07219',
  Kotlin: '#7F52FF',
  Scala: '#DE3423',
  Groovy: '#4298B8',
  Clojure: '#5881D8',
  Python: '#3776AB',
  Ruby: '#CC342D',
  PHP: '#777BB4',
  Lua: '#000080',
  Shell: '#4EAA25',
  PowerShell: '#5391FE',
  JavaScript: '#F7DF1E',
  TypeScript: '#3178C6',
  CSS: '#1572B6',
  SCSS: '#CF649A',
  Less: '#1D365D',
  Haskell: '#5E5086',
  Elixir: '#4E2A8E',
  Erlang: '#A90533',
  'F#': '#376BBA',
  OCaml: '#EC6813',
  Swift: '#F05138',
  'Objective-C': '#0B5A9D',
  Dart: '#0175C2',
  R: '#276DC3',
  Julia: '#9558B2',
  MATLAB: '#E16731',
  SQL: '#CC342D',
  CMake: '#064F8C',
  Dockerfile: '#2496ED',
  Pascal: '#273A8F',
  Fortran: '#734F96',
  BASIC: '#B03931',
  Haxe: '#EA8220'
}

export const DEFAULT_LANGUAGE_COLOR = '#8B8B8B'

export const getLanguageColor = (language: string): string => LANGUAGE_COLORS[language] ?? DEFAULT_LANGUAGE_COLOR
