const fs = require('fs')
const path = require('path')
const _ = require('lodash')
const yaml = require('js-yaml')
const stringify = require('fast-json-stable-stringify')

module.exports = inputSchema => {
  const argv = process.argv.slice(2)
  const config = argv[0]

  if (config) {
    const overridesFile = path.join('src/services', config, 'overrides.yaml')
    const transformFile = path.join('src/services', config, 'transform.js')

    let paths = inputSchema.paths

    if (fs.existsSync(overridesFile)) {
      const data = fs.readFileSync(overridesFile, 'utf8')
      const { allowpaths, operationIdOverrides } = yaml.safeLoad(data)

      if (!allowpaths.includes('*')) {
        paths = _.pick(paths, ...allowpaths)
      }

      _.forIn(operationIdOverrides, (value, key) => {
        const [path, method] = key.split('.')

        if (path && method && _.has(paths, path) && _.has(paths[path], method)) {
          _.set(paths, [path, method, 'operationId'], value)
        }
      })
    }

    inputSchema.paths = paths

    if (fs.existsSync(transformFile)) {
      const transform = require(path.resolve(process.cwd(), transformFile))

      inputSchema = transform(inputSchema)
    }
  }

  // stringify and parse json to get a stable object
  return JSON.parse(stringify(inputSchema))
}
