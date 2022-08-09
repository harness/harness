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
