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
    docs: {
      description: `Give warning for statements 'expect(document.body).toMatchSnapshot()'`
    }
  },

  create: function (context) {
    return {
      CallExpression(node) {
        if (
          get(node, 'callee.object.callee.name') === 'expect' &&
          get(node, 'callee.object.arguments[0].object.name') === 'document' &&
          get(node, 'callee.object.arguments[0].property.name') === 'body' &&
          get(node, 'callee.property.name') === 'toMatchSnapshot'
        ) {
          return context.report({
            node,
            message: 'document.body match snapshot not allowed'
          })
        }
        return null
      }
    }
  }
}
