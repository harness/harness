/*
 * Copyright 2023 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

const packageJSON = require('../package.json')
const { pick, mapValues } = require('lodash')

/**
 * These packages must be stricly shared with exact versions
 */
const ExactSharedPackages = [
  'react-dom',
  'react',
  'react-router-dom',
  '@blueprintjs/core',
  '@blueprintjs/select',
  '@blueprintjs/datetime',
  'restful-react'
]

/**
 * @type {import('webpack').ModuleFederationPluginOptions}
 */
module.exports = {
  name: 'codeRemote',
  filename: 'remoteEntry.js',
  exposes: {
    './App': './src/App.tsx',
    './Repositories': './src/pages/RepositoriesListing/RepositoriesListing.tsx',
    './Repository': './src/pages/Repository/Repository.tsx',
    './FileEdit': './src/pages/RepositoryFileEdit/RepositoryFileEdit.tsx',
    './Commits': './src/pages/RepositoryCommits/RepositoryCommits.tsx',
    './Commit': './src/pages/RepositoryCommit/RepositoryCommit.tsx',
    './Branches': './src/pages/RepositoryBranches/RepositoryBranches.tsx',
    './PullRequests': './src/pages/PullRequests/PullRequests.tsx',
    './Tags': './src/pages/RepositoryTags/RepositoryTags.tsx',
    './PullRequest': './src/pages/PullRequest/PullRequest.tsx',
    './Compare': './src/pages/Compare/Compare.tsx',
    './Settings': './src/pages/RepositorySettings/RepositorySettings.tsx',
    './Webhooks': './src/pages/Webhooks/Webhooks.tsx',
    './WebhookNew': './src/pages/WebhookNew/WebhookNew.tsx',
    './Search': './src/pages/Search/CodeSearchPage.tsx',
    './Labels': './src/pages/ManageSpace/ManageLabels/ManageLabels.tsx',
    './WebhookDetails': './src/pages/WebhookDetails/WebhookDetails.tsx',
    './NewRepoModalButton': './src/components/NewRepoModalButton/NewRepoModalButton.tsx',
    './HAREnterpriseApp': './src/ar/app/EnterpriseApp.tsx',
    './HARCreateRegistryButton': './src/ar/views/CreateRegistryButton/CreateRegistryButton.tsx'
  },
  shared: {
    formik: packageJSON.dependencies['formik'],
    ...mapValues(pick(packageJSON.dependencies, ExactSharedPackages), version => ({
      singleton: true,
      requiredVersion: version
    }))
  }
}
