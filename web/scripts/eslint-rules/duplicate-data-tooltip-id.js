const { get } = require('lodash')
const toolTipValuesMap = {}
module.exports = {
  meta: {
    docs: {
      description: `Give warning for duplicate tooltip id's'`
    }
  },

  create: function (context) {
    return {
      JSXAttribute(node) {
        if (get(node, 'name.name') === 'data-tooltip-id' && get(node, 'value.type') === 'Literal') {
          if (toolTipValuesMap[get(node, 'value.value')]) {
            return context.report({
              node,
              message: 'Duplicate tooltip id'
            })
          } else {
            toolTipValuesMap[get(node, 'value.value')] = true
          }
        }
        return null
      }
    }
  }
}
