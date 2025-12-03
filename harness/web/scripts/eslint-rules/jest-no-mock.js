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

const { get } = require('lodash')

module.exports = {
  meta: {
    schema: [
      {
        type: 'object',
        properties: {
          module: {
            type: 'object'
          }
        },
        additionalProperties: false
      }
    ],
    docs: {
      description: `Restrict some properties from being mocked in jest`
    }
  },

  create: function (context) {
    return {
      CallExpression(node) {
        const moduleList = context.options[0].module
        if (
          get(node, 'callee.type') === 'MemberExpression' &&
          get(node, 'callee.object.type') === 'Identifier' &&
          get(node, 'callee.object.name') === 'jest' &&
          get(node, 'callee.property.name') === 'mock' &&
          get(node, 'arguments[0].type') === 'Literal' &&
          moduleList.hasOwnProperty(get(node, 'arguments[0].value'))
        ) {
          const errorMessage = moduleList[get(node, 'arguments[0].value')]
          return context.report({
            node,
            message: errorMessage
          })
        }
        return null
      }
    }
  }
}
