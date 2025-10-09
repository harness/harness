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

process.env.TZ = 'GMT'

const { pathsToModuleNameMapper } = require('ts-jest')
const { compilerOptions } = require('./tsconfig')

module.exports = {
  globals: {
    __DEV__: false
  },
  snapshotFormat: {
    escapeString: true,
    printBasicPrototype: true
  },
  setupFilesAfterEnv: ['<rootDir>/scripts/jest/setup-file.js'],
  collectCoverage: true,
  collectCoverageFrom: [
    'src/**/*.{ts,tsx}',
    '!src/index.tsx',
    '!src/App.tsx',
    '!src/bootstrap.tsx',
    '!src/framework/strings/**',
    '!src/services/**',
    '!src/**/*.d.ts',
    '!src/**/*.test.{ts,tsx}',
    '!src/**/*.stories.{ts,tsx}',
    '!src/**/__test__/**',
    '!src/**/__tests__/**',
    '!src/utils/test/**',
    '!src/AppUtils.ts'
  ],
  coverageReporters: ['lcov', 'json-summary', 'json'],
  testEnvironment: 'jsdom',
  transform: {
    '^.+\\.tsx?$': [
      'ts-jest',
      {
        tsconfig: '<rootDir>/tsconfig.json',
        isolatedModules: true,
        diagnostics: false,
        __DEV__: false,
        __ON_PREM__: false
      }
    ],
    '^.+\\.jsx?$': [
      'ts-jest',
      {
        tsconfig: '<rootDir>/tsconfig.json',
        isolatedModules: true,
        diagnostics: false,
        __DEV__: false,
        __ON_PREM__: false
      }
    ],
    '^.+\\.ya?ml$': '<rootDir>/scripts/jest/yaml-transform.js',
    '^.+\\.gql$': '<rootDir>/scripts/jest/gql-loader.js'
  },
  moduleDirectories: ['node_modules', 'src'],
  testMatch: ['**/?(*.)+(spec|test).[jt]s?(x)'],
  moduleNameMapper: {
    '\\.s?css$': 'identity-obj-proxy',
    'monaco-editor': '<rootDir>/node_modules/react-monaco-editor',
    '\\.(jpg|jpeg|png|gif|svg|eot|otf|webp|ttf|woff|woff2|mp4|webm|wav|mp3|m4a|aac|oga)$':
      '<rootDir>/scripts/jest/file-mock.js',
    '\\.svg?.url': '<rootDir>/scripts/jest/file-mock.js',
    '@uiw/react-markdown-preview': '<rootDir>/node_modules/@uiw/react-markdown-preview/dist/markdown.min.js',
    ...pathsToModuleNameMapper(compilerOptions.paths)
  },
  // TODO: removing coverage check for now
  // coverageThreshold: {
  //   global: {
  //     statements: 60,
  //     branches: 40,
  //     functions: 40,
  //     lines: 60
  //   }
  // },
  transformIgnorePatterns: [
    'node_modules/(?!(date-fns|lodash-es|@harnessio/uicore|@harnessio/design-system|@harnessio/react-har-service-client|@harnessio/react-ssca-manager-client|@harnessio/react-ng-manager-client)/)'
  ],
  testPathIgnorePatterns: ['<rootDir>/dist', '<rootDir>/src/static']
}
