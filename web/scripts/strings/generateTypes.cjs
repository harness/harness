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

const path = require('path')
const fs = require('fs')

const yaml = require('yaml')
const _ = require('lodash')
const glob = require('glob')

const runPrettier = require('../utils/runPrettier.cjs')

function flattenKeys(data, parentPath = []) {
  const keys = []

  _.keys(data).forEach(key => {
    const value = data[key]
    const newPath = [...parentPath, key]

    if (Array.isArray(value)) {
      throw new TypeError(`Array is not supported in strings.yaml\nPath: "${newPath.join('.')}"`)
    }

    if (_.isPlainObject(data[key])) {
      keys.push(...flattenKeys(data[key], [...parentPath, key]))
    } else {
      keys.push([...parentPath, key].join('.'))
    }
  })

  keys.sort()

  return keys
}

async function generateTypes() {
  const i18nContent = await fs.promises.readFile(path.resolve(process.cwd(), `src/i18n/strings.en.yaml`), 'utf8')
  const i18nCDEContent = await fs.promises.readFile(
    path.resolve(process.cwd(), `src/cde-gitness/strings/strings.en.yaml`),
    'utf8'
  )

  const allData = [
    {
      moduleRef: null,
      keys: flattenKeys(yaml.parse(i18nContent))
    },
    {
      moduleRef: 'cde',
      keys: flattenKeys(yaml.parse(i18nCDEContent), ['cde'])
    }
  ]

  let content = `
/**
  * This file is auto-generated. Please do not modify this file manually.
  * Use the command \`yarn strings\` to regenerate this file.
  */
export interface StringsMap {`

  allData
    .flatMap(({ keys }) => keys)
    .forEach(key => {
      content += `\n  '${key}': string`
    })

  content += `\n}`

  content = await runPrettier(content, 'typescript')

  await fs.promises.writeFile(path.resolve(process.cwd(), 'src/framework/strings/stringTypes.ts'), content, 'utf8')
}

module.exports = generateTypes
