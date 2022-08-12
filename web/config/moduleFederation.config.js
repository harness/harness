const packageJSON = require('../package.json');
const { pick, omit, mapValues } = require('lodash');

/**
 * These packages must be stricly shared with exact versions
 */
 const ExactSharedPackages = [
    'react',
    'react-dom',
    'react-router-dom',
    '@harness/use-modal',
    '@blueprintjs/core',
    '@blueprintjs/select',
    '@blueprintjs/datetime',
    'restful-react',
    '@harness/monaco-yaml',
    'monaco-editor',
    'monaco-editor-core',
    'monaco-languages',
    'monaco-plugin-helpers',
    'react-monaco-editor'
  ]

/**
 * @type {import('webpack').ModuleFederationPluginOptions}
 */
module.exports = {
    name: 'governance',
    filename: 'remoteEntry.js',
    library: {
      type: 'var',
      name: 'governance'
    },
    exposes: {
      './App': './src/App.tsx',
      './EvaluationModal': './src/modals/EvaluationModal/EvaluationModal.tsx',
      './PipelineGovernanceView': './src/views/PipelineGovernanceView/PipelineGovernanceView.tsx',
      './EvaluationView': './src/views/EvaluationView/EvaluationView.tsx',
      './PolicySetWizard': './src/pages/PolicySets/components/PolicySetWizard.tsx'
    },
    shared: {
      formik: packageJSON.dependencies['formik'],
      ...mapValues(pick(packageJSON.dependencies, ExactSharedPackages), version => ({
        singleton: true,
        requiredVersion: version
      }))
    }
};