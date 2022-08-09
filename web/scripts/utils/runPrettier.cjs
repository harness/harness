const fs = require('fs')
const path = require('path')

const prettier = require('prettier')

/**
 * Run prettier on given content using the specified parser
 * @param content {String}
 * @param parser {String}
 */
async function runPrettier(content, parser) {
  const prettierConfig = await prettier.resolveConfig(process.cwd())

  return prettier.format(content, { ...prettierConfig, parser })
}

module.exports = runPrettier
