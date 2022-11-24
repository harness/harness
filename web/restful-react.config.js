/**
 * Please match the config key to the directory under services.
 * This is required for the transform to work
 */
const customGenerator = require('./scripts/swagger-custom-generator.js')

module.exports = {
  code: {
    output: 'src/services/code/index.tsx',
    file: 'src/services/code/swagger.yaml',
    customImport: `import { getConfigNew } from "../config";`,
    customProps: {
      base: `{getConfigNew("code")}`
    }
  }
}
