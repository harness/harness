/**
 * Please match the config key to the directory under services.
 * This is required for the transform to work
 */
const customGenerator = require('./scripts/swagger-custom-generator.js')

module.exports = {
  scm: {
    output: 'src/services/scm/index.tsx',
    file: 'src/services/scm/swagger.yaml',
    customImport: `import { getConfigNew } from "../config";`,
    customProps: {
      base: `{getConfigNew("scm")}`
    }
  }
}
