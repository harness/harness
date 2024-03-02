/*
 * Copyright 2023 Harness, Inc.
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

import { useMemo } from 'react'
import { pdfjs } from 'react-pdf'
import { useAppContext } from 'AppContext'
import type { RepoFileContent } from 'services/code'
import type { GitInfoProps } from './GitUtils'

// TODO: Configure this to use a local worker/webpack loader
// Maybe use pdfjs directly: https://github.com/mozilla/pdf.js/blob/master/examples/webpack/webpack.config.js
pdfjs.GlobalWorkerOptions.workerSrc = `//unpkg.com/pdfjs-dist@${pdfjs.version}/build/pdf.worker.min.js`

type UseFileViewerDecisionProps = Pick<GitInfoProps, 'repoMetadata' | 'gitRef' | 'resourcePath' | 'resourceContent'>

interface UseFileViewerDecisionResult {
  category: FileCategory
  isFileTooLarge: boolean
  isViewable: string | boolean
  filename: string
  extension: string
  size: number
  base64Data: string
  rawURL: string
  isText: boolean
}

export interface RepoContentExtended extends RepoFileContent {
  size?: number
  target?: string
  commit_sha?: string
  url?: string
}

export function useFileContentViewerDecision({
  repoMetadata,
  gitRef,
  resourcePath,
  resourceContent
}: UseFileViewerDecisionProps): UseFileViewerDecisionResult {
  const { routingId } = useAppContext()
  const metadata = useMemo(() => {
    const filename = resourceContent.name as string
    const extension = filename?.split('.').pop() || ''
    const isSymlink = resourceContent?.type === 'symlink'
    const isSubmodule = resourceContent?.type === 'submodule'

    const isMarkdown = extension.toLowerCase() === 'md'
    const isPdf = extension.toLowerCase() === 'pdf'
    const isSVG = extension.toLowerCase() === 'svg'
    const isImage = ImageExtensions.includes(extension.toLowerCase())
    const isAudio = AudioExtensions.includes(extension.toLowerCase())
    const isVideo = VideoExtensions.includes(extension.toLowerCase())
    const isText = !!(
      SpecialTextFiles.find(name => name.toLowerCase() === filename?.toLowerCase()) ||
      TextExtensions.includes(extension.toLowerCase()) ||
      isSymlink ||
      isSubmodule ||
      !extension ||
      extension === filename ||
      '.' + extension === filename
    )

    const category = isMarkdown
      ? FileCategory.MARKDOWN
      : isSVG
      ? FileCategory.SVG
      : isPdf
      ? FileCategory.PDF
      : isImage
      ? FileCategory.IMAGE
      : isAudio
      ? FileCategory.AUDIO
      : isVideo
      ? FileCategory.VIDEO
      : isSymlink
      ? FileCategory.SYMLINK
      : isSubmodule
      ? FileCategory.SUBMODULE
      : isText
      ? FileCategory.TEXT
      : FileCategory.OTHER
    const isViewable = isPdf || isSVG || isImage || isAudio || isVideo || isText || isSubmodule || isSymlink
    const resourceData = resourceContent?.content as RepoContentExtended
    const isFileTooLarge =
      resourceData?.size && resourceData?.data_size ? resourceData?.size !== resourceData?.data_size : false
    const rawURL = `/code/api/v1/repos/${repoMetadata?.path}/+/raw/${resourcePath}?routingId=${routingId}&git_ref=${gitRef}`
    return {
      category,

      isFileTooLarge,
      isViewable,
      isText,

      filename,
      extension,
      size: resourceData?.size || 0,

      // base64 data returned from content API. This snapshot can be truncated by backend
      base64Data: resourceData?.data || resourceData?.target || resourceData?.url || '',

      rawURL
    }
  }, [resourceContent.content]) // eslint-disable-line react-hooks/exhaustive-deps

  return metadata
}

export const MAX_VIEWABLE_FILE_SIZE = 100 * 1024 * 1024 // 100 MB

export enum FileCategory {
  MARKDOWN = 'MARKDOWN',
  SVG = 'SVG',
  PDF = 'PDF',
  IMAGE = 'IMAGE',
  AUDIO = 'AUDIO',
  VIDEO = 'VIDEO',
  TEXT = 'TEXT',
  SYMLINK = 'SYMLINK',
  SUBMODULE = 'SUBMODULE',
  OTHER = 'OTHER'
}

// Parts are copied from https://github.com/sindresorhus/text-extensions
// MIT License
// Copyright (c) Sindre Sorhus <sindresorhus@gmail.com> (https://sindresorhus.com)
const TextExtensions = [
  'ada',
  'adb',
  'ads',
  'applescript',
  'as',
  'asc',
  'ascii',
  'ascx',
  'asm',
  'asmx',
  'asp',
  'aspx',
  'atom',
  'au3',
  'awk',
  'bas',
  'bash',
  'bashrc',
  'bat',
  'bbcolors',
  'bcp',
  'bazel',
  'bdsgroup',
  'bdsproj',
  'bib',
  'bowerrc',
  'c',
  'cbl',
  'cc',
  'cfc',
  'cfg',
  'cfm',
  'cfml',
  'cgi',
  'cjs',
  'clj',
  'cljs',
  'cls',
  'cmake',
  'cmd',
  'cnf',
  'cob',
  'code-snippets',
  'coffee',
  'coffeekup',
  'conf',
  'cp',
  'cpp',
  'cpt',
  'cpy',
  'crt',
  'cs',
  'csh',
  'cson',
  'csproj',
  'csr',
  'css',
  'csslintrc',
  'csv',
  'ctl',
  'curlrc',
  'cxx',
  'd',
  'dart',
  'dfm',
  'diff',
  'dof',
  'dpk',
  'dpr',
  'dproj',
  'dtd',
  'eco',
  'editorconfig',
  'ejs',
  'el',
  'elm',
  'emacs',
  'eml',
  'ent',
  'erb',
  'erl',
  'eslintignore',
  'eslintrc',
  'ex',
  'exs',
  'f',
  'f03',
  'f77',
  'f90',
  'f95',
  'fish',
  'for',
  'fpp',
  'frm',
  'fs',
  'fsproj',
  'fsx',
  'ftn',
  'gemrc',
  'gemspec',
  'gitattributes',
  'gitconfig',
  'gitignore',
  'gitkeep',
  'gitmodules',
  'go',
  'gpp',
  'gradle',
  'graphql',
  'groovy',
  'groupproj',
  'grunit',
  'gtmpl',
  'gvimrc',
  'h',
  'haml',
  'hbs',
  'hgignore',
  'hh',
  'hpp',
  'hrl',
  'hs',
  'hta',
  'htaccess',
  'htc',
  'htm',
  'html',
  'htpasswd',
  'hxx',
  'iced',
  'iml',
  'inc',
  'inf',
  'info',
  'ini',
  'ino',
  'int',
  'irbrc',
  'itcl',
  'itermcolors',
  'itk',
  'jade',
  'java',
  'jhtm',
  'jhtml',
  'js',
  'jscsrc',
  'jshintignore',
  'jshintrc',
  'json',
  'json5',
  'jsonld',
  'jsp',
  'jspx',
  'jsx',
  'ksh',
  'less',
  'lhs',
  'lisp',
  'log',
  'ls',
  'lsp',
  'lua',
  'm',
  'm4',
  'mak',
  'map',
  'markdown',
  'master',
  'md',
  'mdown',
  'mdwn',
  'mdx',
  'metadata',
  'mht',
  'mhtml',
  'mjs',
  'mk',
  'mkd',
  'mkdn',
  'mkdown',
  'ml',
  'mli',
  'mm',
  'mxml',
  'nfm',
  'nfo',
  'noon',
  'npmignore',
  'npmrc',
  'nuspec',
  'nvmrc',
  'ops',
  'pas',
  'pasm',
  'patch',
  'pbxproj',
  'pch',
  'pem',
  'pg',
  'php',
  'php3',
  'php4',
  'php5',
  'phpt',
  'phtml',
  'pir',
  'pl',
  'pm',
  'pmc',
  'pod',
  'pot',
  'prettierrc',
  'properties',
  'props',
  'pt',
  'pug',
  'purs',
  'py',
  'pyx',
  'r',
  'rake',
  'rb',
  'rbw',
  'rc',
  'rdoc',
  'rdoc_options',
  'resx',
  'rexx',
  'rhtml',
  'rjs',
  'rlib',
  'ron',
  'rs',
  'rss',
  'rst',
  'rtf',
  'rvmrc',
  'rxml',
  's',
  'sass',
  'scala',
  'scm',
  'scss',
  'seestyle',
  'sh',
  'shtml',
  'sln',
  'sls',
  'spec',
  'sql',
  'sqlite',
  'sqlproj',
  'srt',
  'ss',
  'sss',
  'st',
  'strings',
  'sty',
  'styl',
  'stylus',
  'sub',
  'sublime-build',
  'sublime-commands',
  'sublime-completions',
  'sublime-keymap',
  'sublime-macro',
  'sublime-menu',
  'sublime-project',
  'sublime-settings',
  'sublime-workspace',
  'sv',
  'svc',
  'svg',
  'swift',
  't',
  'tcl',
  'tcsh',
  'terminal',
  'tex',
  'text',
  'textile',
  'tg',
  'tk',
  'tmLanguage',
  'tmpl',
  'tmTheme',
  'tpl',
  'ts',
  'tsv',
  'tsx',
  'tt',
  'tt2',
  'ttml',
  'twig',
  'txt',
  'v',
  'vb',
  'vbproj',
  'vbs',
  'vcproj',
  'vcxproj',
  'vh',
  'vhd',
  'vhdl',
  'vim',
  'viminfo',
  'vimrc',
  'vm',
  'vue',
  'webapp',
  'webmanifest',
  'wsc',
  'x-php',
  'xaml',
  'xht',
  'xhtml',
  'xml',
  'xs',
  'xsd',
  'xsl',
  'xslt',
  'y',
  'yaml',
  'yml',
  'zsh',
  'zshrc',
  'ics',
  'rego',
  'tf',
  'hcl',
  'mod',
  'sum',
  'rst',
  'toml',
  'abap',
  'uos',
  'uot',
  'ahk',
  'asciidoc',
  'slk',
  'env',
  'alpine',
  'tf',
  'tfvars',
  'tfstate',
  'hcl',
  'ipynb'
]

const SpecialTextFiles = [
  'Dockerfile',
  '.dockerignore',
  '.gitignore',
  'yarn.lock',
  'README',
  'LICENSE',
  'CHANGELOG',
  'Makefile',
  'Procfile',
  '.env',
  '.alpine'
]

const ImageExtensions = ['jpg', 'jpeg', 'png', 'gif', 'svg', 'ico', 'bmp']

const VideoExtensions = ['ogg', 'mp4', 'webm', 'mpeg']

const AudioExtensions = ['mp3', 'wav']
