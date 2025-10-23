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

const path = require('path')
const fs = require('fs')

const yaml = require('yaml')
const _ = require('lodash')
const glob = require('glob')

const runPrettier = require('../utils/runPrettier.cjs')
const { getLayers } = require('../utils/HarnessModulesUtils.cjs')

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
  const files = glob.sync('src/ar/pages/**/strings.en.yaml')
  const layers = getLayers(true)

  files.push('src/ar/strings/strings.en.yaml') // TODO: remove this after migration

  const promises = layers.map(async ({ dirName, moduleRef }) => {
    const content = await fs.promises.readFile(
      path.resolve(process.cwd(), `src/ar/pages/${dirName}/strings/strings.en.yaml`),
      'utf8'
    )

    return {
      moduleRef,
      keys: flattenKeys(yaml.parse(content), [moduleRef])
    }
  })

  const allData = await Promise.all(promises)

  // TODO: remove this after migration
  const oldStrings = await fs.promises.readFile(path.resolve(process.cwd(), `src/ar/strings/strings.en.yaml`), 'utf8')
  allData.push({
    moduleRef: null,
    keys: flattenKeys(yaml.parse(oldStrings))
  })

  const licenseTxt = await fs.promises.readFile(
    path.resolve(process.cwd(), 'src/ar/scripts/license/.license-header-polyform-free-trial.txt'),
    'utf8'
  )

  let content = `
/**
 * ${licenseTxt.replace('<YEAR>', '2022').split('\n').join('\n * ')}
 */

/**
  * This file is auto-generated. Please do not modify this file manually.
  * Use the command \`yarn strings\` to regenerate this file.
  */
export interface StringsMap {`

  allData
    .slice(0, allData.length) // TODO: remove this line when strings are migrated
    .flatMap(({ keys }) => keys)
    .forEach(key => (content += `\n  '${key}': string`))

  content += `\n}`

  content = await runPrettier(content, 'typescript')

  await fs.promises.writeFile(path.resolve(process.cwd(), 'src/ar/strings/types.ts'), content, 'utf8')
}

module.exports = generateTypes
