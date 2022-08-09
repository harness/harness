process.env.TZ = 'GMT'

const config = require('./jest.config')
const { omit } = require('lodash')

module.exports = {
  ...omit(config, ['coverageThreshold', 'coverageReporters']),
  coverageReporters: ['text-summary', 'json-summary']
}
