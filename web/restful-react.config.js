/**
 * Please match the config key to the directory under services.
 * This is required for the transform to work
 */
const customGenerator = require('./scripts/swagger-custom-generator.js')

module.exports = {
  pm: {
    output: 'src/services/policy-mgmt/index.tsx',
    file: '../design/gen/http/openapi3.json',
    customImport: `import { getConfigNew } from "../config";`,
    customProps: {
      base: `{getConfigNew("pm")}`
    }
  }
}
