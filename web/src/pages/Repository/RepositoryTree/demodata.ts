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

//
// @see https://rct.lukasbach.com/docs/getstarted#providing-the-data-for-the-tree
//
// const readTemplate = (template: any, data: any = { items: {} }) => {
//   for (const [key, value] of Object.entries(template)) {
//     data.items[key] = {
//       index: key,
//       canMove: true,
//       hasChildren: value !== null,
//       children: value !== null ? Object.keys(value as Record<string, unknown>) : undefined,
//       data: key,
//       canRename: true
//     }

//     if (value !== null) {
//       readTemplate(value, data)
//     }
//   }
//   return data
// }

// const sampleTreeData = {
//   root: {
//     config: {
//       'moduleFederation.config.js': null,
//       'webpack.common.js': null,
//       'webpack.dev.js': null,
//       'webpack.prod.js': null
//     },
//     cypress: {
//       integration: {},
//       videos: {}
//     },
//     scripts: {
//       'clean-css-types.js': null,
//       'swagger-transform.js': null
//     },
//     src: {
//       '.eslintignore': null,
//       'dist.go': null,
//       'jest.config.js': null,
//       components: {
//         NameIdDescriptionTags: {},
//         OptionsMenuButton: {},
//         Permissions: {},
//         SpinnerWrapper: {},
//         TrialBanner: {
//           'TrialBanner.tsx': null,
//           'TrialBanner.module.scss': null,
//           'TrialBanner.module.scss.d.ts': null
//         }
//       },
//       views: {},
//       utils: {},
//       framework: {},
//       i18n: {},
//       services: {},
//       hooks: {},
//       'bootstrap.tsx': null,
//       'index.html': null,
//       'index.ts': null,
//       'RouteDefinitions.ts': null,
//       'RouteDestinations.ts': null,
//       'bootstrap.scss': null,
//       'global.d.ts': null,
//       'App.tsx': null,
//       'AppContext.tsx': null,
//       'App.scss': null,
//       'AppUtils.ts': null,
//       'AppProps.ts': null
//     },
//     'README.md': null
//   }
// }

// const sort = o =>
//   Object(o) !== o || Array.isArray(o)
//     ? o
//     : Object.keys(o)
//         .sort()
//         .reduce((a, k) => ({ ...a, [k]: sort(o[k]) }), {})

// export const sampleTree1 = readTemplate(sort(sampleTreeData))
// console.log({ sampleTree1 })
export const sampleTree = {
  items: {
    root: {
      index: 'root',
      canMove: true,
      hasChildren: true,
      children: ['config', 'cypress', 'scripts', 'src', 'README.md'],
      data: 'root',
      canRename: true
    },
    config: {
      index: 'config',
      canMove: true,
      hasChildren: true,
      children: ['moduleFederation.config.js', 'webpack.common.js', 'webpack.dev.js', 'webpack.prod.js'],
      data: 'config',
      canRename: true
    },
    'moduleFederation.config.js': {
      index: 'moduleFederation.config.js',
      canMove: true,
      hasChildren: false,
      data: 'moduleFederation.config.js',
      canRename: true
    },
    'webpack.common.js': {
      index: 'webpack.common.js',
      canMove: true,
      hasChildren: false,
      data: 'webpack.common.js',
      canRename: true
    },
    'webpack.dev.js': {
      index: 'webpack.dev.js',
      canMove: true,
      hasChildren: false,
      data: 'webpack.dev.js',
      canRename: true
    },
    'webpack.prod.js': {
      index: 'webpack.prod.js',
      canMove: true,
      hasChildren: false,
      data: 'webpack.prod.js',
      canRename: true
    },
    cypress: {
      index: 'cypress',
      canMove: true,
      hasChildren: true,
      children: ['integration', 'videos'],
      data: 'cypress',
      canRename: true
    },
    integration: {
      index: 'integration',
      canMove: true,
      hasChildren: true,
      children: [],
      data: 'integration',
      canRename: true
    },
    videos: {
      index: 'videos',
      canMove: true,
      hasChildren: true,
      children: [],
      data: 'videos',
      canRename: true
    },
    scripts: {
      index: 'scripts',
      canMove: true,
      hasChildren: true,
      children: ['clean-css-types.js', 'swagger-transform.js'],
      data: 'scripts',
      canRename: true
    },
    'clean-css-types.js': {
      index: 'clean-css-types.js',
      canMove: true,
      hasChildren: false,
      data: 'clean-css-types.js',
      canRename: true
    },
    'swagger-transform.js': {
      index: 'swagger-transform.js',
      canMove: true,
      hasChildren: false,
      data: 'swagger-transform.js',
      canRename: true
    },
    src: {
      index: 'src',
      canMove: true,
      hasChildren: true,
      children: [
        '.eslintignore',
        'App.scss',
        'App.tsx',
        'AppContext.tsx',
        'AppProps.ts',
        'AppUtils.ts',
        'RouteDefinitions.ts',
        'RouteDestinations.ts',
        'bootstrap.scss',
        'bootstrap.tsx',
        'components',
        'dist.go',
        'framework',
        'global.d.ts',
        'hooks',
        'i18n',
        'index.html',
        'index.ts',
        'jest.config.js',
        'services',
        'utils',
        'views'
      ],
      data: 'src',
      canRename: true
    },
    '.eslintignore': {
      index: '.eslintignore',
      canMove: true,
      hasChildren: false,
      data: '.eslintignore',
      canRename: true
    },
    'App.scss': {
      index: 'App.scss',
      canMove: true,
      hasChildren: false,
      data: 'App.scss',
      canRename: true
    },
    'App.tsx': {
      index: 'App.tsx',
      canMove: true,
      hasChildren: false,
      data: 'App.tsx',
      canRename: true
    },
    'AppContext.tsx': {
      index: 'AppContext.tsx',
      canMove: true,
      hasChildren: false,
      data: 'AppContext.tsx',
      canRename: true
    },
    'AppProps.ts': {
      index: 'AppProps.ts',
      canMove: true,
      hasChildren: false,
      data: 'AppProps.ts',
      canRename: true
    },
    'AppUtils.ts': {
      index: 'AppUtils.ts',
      canMove: true,
      hasChildren: false,
      data: 'AppUtils.ts',
      canRename: true
    },
    'RouteDefinitions.ts': {
      index: 'RouteDefinitions.ts',
      canMove: true,
      hasChildren: false,
      data: 'RouteDefinitions.ts',
      canRename: true
    },
    'RouteDestinations.ts': {
      index: 'RouteDestinations.ts',
      canMove: true,
      hasChildren: false,
      data: 'RouteDestinations.ts',
      canRename: true
    },
    'bootstrap.scss': {
      index: 'bootstrap.scss',
      canMove: true,
      hasChildren: false,
      data: 'bootstrap.scss',
      canRename: true
    },
    'bootstrap.tsx': {
      index: 'bootstrap.tsx',
      canMove: true,
      hasChildren: false,
      data: 'bootstrap.tsx',
      canRename: true
    },
    components: {
      index: 'components',
      canMove: true,
      hasChildren: true,
      children: ['NameIdDescriptionTags', 'OptionsMenuButton', 'Permissions', 'SpinnerWrapper', 'TrialBanner'],
      data: 'components',
      canRename: true
    },
    NameIdDescriptionTags: {
      index: 'NameIdDescriptionTags',
      canMove: true,
      hasChildren: true,
      children: [],
      data: 'NameIdDescriptionTags',
      canRename: true
    },
    OptionsMenuButton: {
      index: 'OptionsMenuButton',
      canMove: true,
      hasChildren: true,
      children: [],
      data: 'OptionsMenuButton',
      canRename: true
    },
    Permissions: {
      index: 'Permissions',
      canMove: true,
      hasChildren: true,
      children: [],
      data: 'Permissions',
      canRename: true
    },
    SpinnerWrapper: {
      index: 'SpinnerWrapper',
      canMove: true,
      hasChildren: true,
      children: [],
      data: 'SpinnerWrapper',
      canRename: true
    },
    TrialBanner: {
      index: 'TrialBanner',
      canMove: true,
      hasChildren: true,
      children: ['TrialBanner.module.scss', 'TrialBanner.module.scss.d.ts', 'TrialBanner.tsx'],
      data: 'TrialBanner',
      canRename: true
    },
    'TrialBanner.module.scss': {
      index: 'TrialBanner.module.scss',
      canMove: true,
      hasChildren: false,
      data: 'TrialBanner.module.scss',
      canRename: true
    },
    'TrialBanner.module.scss.d.ts': {
      index: 'TrialBanner.module.scss.d.ts',
      canMove: true,
      hasChildren: false,
      data: 'TrialBanner.module.scss.d.ts',
      canRename: true
    },
    'TrialBanner.tsx': {
      index: 'TrialBanner.tsx',
      canMove: true,
      hasChildren: false,
      data: 'TrialBanner.tsx',
      canRename: true
    },
    'dist.go': {
      index: 'dist.go',
      canMove: true,
      hasChildren: false,
      data: 'dist.go',
      canRename: true
    },
    framework: {
      index: 'framework',
      canMove: true,
      hasChildren: true,
      children: [],
      data: 'framework',
      canRename: true
    },
    'global.d.ts': {
      index: 'global.d.ts',
      canMove: true,
      hasChildren: false,
      data: 'global.d.ts',
      canRename: true
    },
    hooks: {
      index: 'hooks',
      canMove: true,
      hasChildren: true,
      children: [],
      data: 'hooks',
      canRename: true
    },
    i18n: {
      index: 'i18n',
      canMove: true,
      hasChildren: true,
      children: [],
      data: 'i18n',
      canRename: true
    },
    'index.html': {
      index: 'index.html',
      canMove: true,
      hasChildren: false,
      data: 'index.html',
      canRename: true
    },
    'index.ts': {
      index: 'index.ts',
      canMove: true,
      hasChildren: false,
      data: 'index.ts',
      canRename: true
    },
    'jest.config.js': {
      index: 'jest.config.js',
      canMove: true,
      hasChildren: false,
      data: 'jest.config.js',
      canRename: true
    },
    services: {
      index: 'services',
      canMove: true,
      hasChildren: true,
      children: [],
      data: 'services',
      canRename: true
    },
    utils: {
      index: 'utils',
      canMove: true,
      hasChildren: true,
      children: [],
      data: 'utils',
      canRename: true
    },
    views: {
      index: 'views',
      canMove: true,
      hasChildren: true,
      children: [],
      data: 'views',
      canRename: true
    },
    'README.md': {
      index: 'README.md',
      canMove: true,
      hasChildren: false,
      data: 'README.md',
      canRename: true
    }
  }
}
