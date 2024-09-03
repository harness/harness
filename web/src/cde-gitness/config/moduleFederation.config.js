/*
 * Copyright 2024 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

const packageJSON = require('../../../package.json')
const { pick, mapValues } = require('lodash')

/**
 * These packages must be stricly shared with exact versions
 */
const ExactSharedPackages = ['formik', 'react-dom', 'react', 'react-router-dom']

/**
 * @type {import('webpack').ModuleFederationPluginOptions}
 */
module.exports = {
  name: 'cdeRemote',
  filename: 'remoteEntry.js',
  exposes: {
    './App': './src/App.tsx',
    './Gitspaces': './src/cde-gitness/pages/GitspaceCreate/GitspaceCreate.tsx',
    './GitspaceDetail': './src/cde-gitness/pages/GitspaceDetails/GitspaceDetails.tsx',
    './GitspaceList': './src/cde-gitness/pages/GitspaceListing/GitspaceListing.tsx'
  },
  shared: Object.assign(
    {},
    mapValues(pick(packageJSON.dependencies, ExactSharedPackages), version => ({
      singleton: true,
      requiredVersion: version
    }))
  )
}
