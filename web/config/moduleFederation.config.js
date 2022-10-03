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
  name: 'scm',
  filename: 'remoteEntry.js',
  library: {
    type: 'var',
    name: 'scmRemote'
  },
  exposes: {
    './App': './src/App.tsx',
    './Welcome': './src/views/Welcome/Welcome.tsx',
    './Repos': './src/views/Repos/Repos.tsx',
    './NewRepo': './src/views/NewRepo/NewRepo.tsx',
    './RepoFiles': './src/views/RepoFiles/RepoFiles.tsx',
    './RepoFileDetails': './src/views/RepoFileDetails/RepoFileDetails.tsx',
    './RepoCommits': './src/views/RepoCommits/RepoCommits.tsx',
    './RepoCommitDetails': './src/views/RepoCommitDetails/RepoCommitDetails.tsx',
    './RepoPullRequests': './src/views/RepoPullRequests/RepoPullRequests.tsx',
    './RepoPullRequestDetails': './src/views/RepoPullRequestDetails/RepoPullRequestDetails.tsx',
    './RepoSettings': './src/views/RepoSettings/RepoSettings.tsx'
  },
  shared: {
    formik: packageJSON.dependencies['formik'],
    ...mapValues(pick(packageJSON.dependencies, ExactSharedPackages), version => ({
      singleton: true,
      requiredVersion: version
    }))
  }
}
