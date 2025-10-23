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

const fs = require('fs')
const path = require('path')
const _ = require('lodash')

const MODULE_REGEX = /^(\d{2,3})-([a-zA-Z0-9-]*)/

/**
 * Get the info about (Harness) modules system
 */
function getModules() {
  const modules = path.resolve(process.cwd(), 'src/ar/pages')

  const dirs = fs.readdirSync(modules, { withFileTypes: true })
  const numberedDirs = dirs
    .map((dir, index) => {
      if (!dir.isDirectory()) return null

      return {
        dirName: dir.name,
        moduleName: dir.name,
        moduleRef: _.camelCase(dir.name),
        number: index
      }
    })
    .filter(mod => mod)

  return _.sortBy(numberedDirs, mod => mod.dirName)
}

/**
 * Get the info about (Harness) modules system as layers
 * @param flatten {Boolean}
 */
function getLayers(flatten) {
  const modules = getModules()

  let layers = []

  modules.forEach(mod => {
    if (!Array.isArray(layers[mod.number])) {
      layers[mod.number] = []
    }

    layers[mod.number].push(mod)
  })

  layers = layers.filter(layer => layer)

  return flatten ? _.flatten(layers) : layers
}

module.exports.MODULE_REGEX = MODULE_REGEX
module.exports.getModules = getModules
module.exports.getLayers = getLayers
