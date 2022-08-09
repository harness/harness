const generateStringTypes = require('../strings/generateTypes.cjs')

class GenerateStringTypesPlugin {
  apply(compiler) {
    compiler.hooks.emit.tapAsync('GenerateStringTypesPlugin', (compilation, callback) => {
      try {
        generateStringTypes().then(
          () => callback(),
          e => callback(e)
        )
      } catch (e) {
        callback(e)
      }
    })
  }
}

module.exports.GenerateStringTypesPlugin = GenerateStringTypesPlugin
