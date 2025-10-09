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

import fs from 'fs'
import path from 'path'

import _ from 'lodash'

import { getModules } from '../utils/HarnessModulesUtils.cjs'
import runPrettier from '../utils/runPrettier.cjs'

const packageJsonData = await fs.promises.readFile(path.resolve(process.cwd(), 'package.json'), 'utf8')
const packageJson = JSON.parse(packageJsonData)
const { extensionToLanguageMap } = packageJson.i18nSettings
const copyright = ``

async function makeLangLoaderFile(modules, extensionEntries) {
  let content = `${copyright}

/* eslint-disable */
/**
 * This file is auto-generated. Please do not modify this file manually.
 * Use the command \`yarn strings\` to regenerate this file.
 */
`

  modules.forEach(({ moduleName, moduleRef }) => {
    content += `import ${moduleRef} from '@ar/pages/${moduleName}/strings/strings.en.yaml'\n`
  })

  content += `
export default function languageLoader() {
  return { ${modules.map(({ moduleRef }) => moduleRef).join(', ')} }
}`
  content = await runPrettier(content, 'typescript')

  await fs.promises.writeFile(
    path.resolve(process.cwd(), `src/ar/frameworks/strings/languageLoader.ts`),
    content,
    'utf8'
  )
}

// create common modules
const modules = await getModules()

// const commonChunkIndex = modules.indexOf(commonChunkLimit)
// const commons = modules.slice(0, commonChunkIndex + 1)
const commonChunkIndex = 0
const restModules = modules.slice(commonChunkIndex)
const extensionEntries = Object.entries(extensionToLanguageMap)

await makeLangLoaderFile(restModules, extensionEntries)
console.log('✅  Generated Language Loader file successfully!')
