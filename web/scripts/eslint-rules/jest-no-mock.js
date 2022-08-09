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
