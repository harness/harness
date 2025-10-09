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

const { get, flatten } = require('lodash')
const { getLayers, MODULE_REGEX } = require('../utils/HarnessModulesUtils.cjs')

const layers = getLayers()
const modulesPath = path.resolve(process.cwd(), 'src/pages')

function checkReferences({ node, restrictedModulesRefs, context }) {
  switch (node.type) {
    case 'Literal':
      const value = node.value || ''

      restrictedModulesRefs.forEach(ref => {
        if (value.startsWith(`${ref}.`)) {
          context.report({
            node,
            message: `Using a string ref "${value}" from restricted module is not allowed`
          })
        }
      })
      break
    case 'JSXExpressionContainer':
      break
    default:
      break
  }
}

module.exports = {
  meta: {
    docs: {
      description: `Restrict usage of string accroding to modules`
    }
  },

  create: function (context) {
    const file = context.getFilename()
    const relativePath = path.relative(modulesPath, file)
    const isFileInsideAModule = !relativePath.startsWith('.')
    const match = relativePath.match(MODULE_REGEX)
    const layerIndex = match ? layers.findIndex(layer => layer.find(mod => mod.dirName === match[0])) : -1
    const restrictedModules = layerIndex > -1 ? layers.slice(layerIndex) : []
    const restrictedModulesRefs = match
      ? flatten(restrictedModules)
          .filter(({ dirName }) => dirName !== match[0])
          .map(({ moduleRef }) => moduleRef)
      : []

    return {
      JSXElement(node) {
        if (!isFileInsideAModule || !match) return null

        if (get(node, 'openingElement.name.name') === 'String') {
          const attrs = get(node, 'openingElement.attributes') || []
          const stringIdAttr = attrs.find(attr => get(attr, 'name.name') === 'stringID')

          if (stringIdAttr && stringIdAttr.value) {
            checkReferences({ node: stringIdAttr.value, restrictedModulesRefs, context })
          }
        }
        return null
      },
      CallExpression(node) {
        if (!isFileInsideAModule || !match) return null

        if (get(node, 'callee.name') === 'getString') {
          const idNode = get(node, 'arguments[0]')

          if (idNode) {
            checkReferences({ node: idNode, restrictedModulesRefs, context })
          }
        }
      }
    }
  }
}
