const yaml = require('yaml')

module.exports = {
  process(src) {
    const json = yaml.parse(src)

    return { code: `module.exports = ${JSON.stringify(json)}` }
  }
}
