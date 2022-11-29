const packageJSON = require('../package.json')
const { pick, omit, mapValues } = require('lodash')

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
  name: 'code',
  filename: 'remoteEntry.js',
  library: {
    type: 'var',
    name: 'codeRemote'
  },
  exposes: {
    './App': './src/App.tsx',
    './RepositoriesListing': './src/pages/RepositoriesListing/RepositoriesListing.tsx',
    './Repository': './src/pages/Repository/Repository.tsx',
    './RepositoryFileEdit': './src/pages/RepositoryFileEdit/RepositoryFileEdit.tsx',
    './RepositoryCommits': './src/pages/RepositoryCommits/RepositoryCommits.tsx',
    './RepositoryBranches': './src/pages/RepositoryBranches/RepositoryBranches.tsx',
    './RepositorySettings': './src/pages/RepositorySettings/RepositorySettings.tsx',
    './RepositoryCreateWebhook': './src/pages/RepositoryCreateWebhook/RepositoryCreateWebhook.tsx'
  },
  shared: {
    formik: packageJSON.dependencies['formik'],
    ...mapValues(pick(packageJSON.dependencies, ExactSharedPackages), version => ({
      singleton: true,
      requiredVersion: version
    }))
  }
}
