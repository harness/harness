/*
 * Copyright 2024 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

const { get } = require('lodash')
const fs = require('fs')
const path = require('path')

const modules = path.resolve(process.cwd(), 'src/ar/pages')
const srcDir = path.resolve(process.cwd(), 'src/ar/')
const dirs = fs
  .readdirSync(modules, { withFileTypes: true })
  .filter(dir => dir.isDirectory())
  .map(dir => dir.name)

module.exports = {
  meta: {
    docs: {
      description: `All the imports from modules must use aliases`
    }
  },

  create: function (context) {
    return {
      ImportDeclaration(node) {
        const source = get(node, 'source.value')
        const filename = context.getFilename() // current file (absolute path)
        const relativeFileName = path.relative(srcDir, filename) // relative path for current file w.r.t to src

        if (typeof source === 'string') {
          // if path directly starts with modules
          if (source.startsWith('pages')) {
            return context.report({
              node,
              message: `Import must use aliased module instead`
            })
          }

          // if it is a relative path
          if (source.startsWith('.')) {
            // TODO: change this to 2 later
            const match = source.match(/\.\.\//g)
            if (match && match.length > 3) {
              return context.report({
                node,
                message: `Import must use aliased module instead`
              })
            }

            const absolutePath = path.resolve(path.dirname(filename), source) // absolute path for the import
            const relativePath = path.relative(srcDir, absolutePath) // realtive path for the import w.r.t to src

            if (
              relativePath.startsWith('pages') && // path start with module
              (relativePath.split(path.sep)[1] !== relativeFileName.split(path.sep)[1] || // files are not in same module
                dirs.some(dir => source.includes(`/${dir}/`))) // file is imported from same module but path is relative to itself
            ) {
              return context.report({
                node,
                message: `Import must use aliased module instead`
              })
            }
          }
        }
      }
    }
  }
}
