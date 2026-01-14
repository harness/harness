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

const restrictedVariables = ['formName', 'data-tooltip-id']

module.exports = {
  meta: {
    docs: {
      description: 'Disallow specific variable identifiers'
    }
  },

  create: function (context) {
    return {
      VariableDeclaration(node) {
        const variableName = get(node, 'declarations[0].id.name')
        if (restrictedVariables.includes(variableName)) {
          return context.report({ node, message: `restricted variable name ${variableName} not allowed` })
        }
        return null
      }
    }
  }
}
