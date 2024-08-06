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

const path = require('path')

const {
  container: { ModuleFederationPlugin },
  DefinePlugin
} = require('webpack')
const MiniCssExtractPlugin = require('mini-css-extract-plugin')
const HTMLWebpackPlugin = require('html-webpack-plugin')
const TsconfigPathsPlugin = require('tsconfig-paths-webpack-plugin')
const GenerateStringTypesPlugin = require('../scripts/webpack/GenerateStringTypesPlugin').GenerateStringTypesPlugin
const { RetryChunkLoadPlugin } = require('webpack-retry-chunk-load-plugin')
const MonacoWebpackPlugin = require('monaco-editor-webpack-plugin')
const moduleFederationConfig = require('./moduleFederation.config')
const moduleFederationConfigCDE = require('./cde/moduleFederation.config')
const CONTEXT = process.cwd()
const DEV = process.env.NODE_ENV === 'development'

const getModuleFields = () => {
  if (process.env.MODULE === 'cde') {
    return {
      moduleFederationConfigEntryName: moduleFederationConfigCDE.name,
      moduleFederationPlugin: new ModuleFederationPlugin(moduleFederationConfigCDE)
    }
  } else {
    return {
      moduleFederationConfigEntryName: moduleFederationConfig.name,
      moduleFederationPlugin: new ModuleFederationPlugin(moduleFederationConfig)
    }
  }
}

const { moduleFederationConfigEntryName, moduleFederationPlugin } = getModuleFields()

module.exports = {
  target: 'web',
  context: CONTEXT,
  stats: {
    modules: false,
    children: false
  },
  entry: {
    [moduleFederationConfigEntryName]: './src/public-path'
  },
  output: {
    publicPath: 'auto',
    pathinfo: false,
    filename: '[name].[contenthash:6].js',
    chunkFilename: '[name].[id].[contenthash:6].js'
  },
  module: {
    rules: [
      {
        test: /\.m?js$/,
        include: /node_modules/,
        type: 'javascript/auto'
      },
      {
        test: /\.(j|t)sx?$/,
        exclude: /node_modules/,
        use: [
          {
            loader: 'ts-loader',
            options: {
              transpileOnly: true
            }
          }
        ]
      },
      {
        test: /\.module\.scss$/,
        exclude: /node_modules/,
        use: [
          MiniCssExtractPlugin.loader,
          {
            loader: 'css-loader',
            options: {
              importLoaders: 1,
              modules: {
                mode: 'local',
                localIdentName: DEV ? '[name]_[local]_[hash:base64:6]' : '[hash:base64:6]',
                exportLocalsConvention: 'camelCaseOnly'
              }
            }
          },
          {
            loader: 'sass-loader',
            options: {
              sassOptions: {
                includePaths: [path.join(CONTEXT, 'src')]
              },
              sourceMap: false,
              implementation: require('sass')
            }
          }
        ]
      },
      {
        test: /(?<!\.module)\.scss$/,
        exclude: /node_modules/,
        use: [
          MiniCssExtractPlugin.loader,
          {
            loader: 'css-loader',
            options: {
              importLoaders: 1,
              modules: false
            }
          },
          {
            loader: 'sass-loader',
            options: {
              sassOptions: {
                includePaths: [path.join(CONTEXT, 'src')]
              },
              implementation: require('sass')
            }
          }
        ]
      },
      {
        test: /\.(jpg|jpeg|png|gif)$/,
        use: [
          {
            loader: 'url-loader',
            options: {
              limit: 2000,
              fallback: 'file-loader'
            }
          }
        ]
      },
      {
        test: /\.svg$/i,
        type: 'asset',
        resourceQuery: /url/ // *.svg?url
      },
      {
        test: /\.svg$/i,
        issuer: /\.[jt]sx?$/,
        resourceQuery: { not: [/url/] }, // exclude react component if *.svg?url
        use: ['@svgr/webpack']
      },
      {
        test: /\.css$/,
        use: ['style-loader', 'css-loader']
      },
      {
        test: /\.ttf$/,
        loader: 'file-loader',
        mimetype: 'application/octet-stream'
      },
      {
        test: /\.ya?ml$/,
        type: 'json',
        use: [
          {
            loader: 'yaml-loader'
          }
        ]
      },
      {
        test: /\.gql$/,
        type: 'asset/source'
      },
      {
        test: /\.(mp4)$/,
        use: [
          {
            loader: 'file-loader'
          }
        ]
      },
      {
        test: /\.md$/,
        use: [
          {
            loader: 'raw-loader',
            options: {
              esModule: false
            }
          }
        ]
      }
    ]
  },
  resolve: {
    extensions: ['.mjs', '.js', '.ts', '.tsx', '.json', '.ttf', '.scss'],
    plugins: [new TsconfigPathsPlugin()],
    alias: {
      'react/jsx-dev-runtime': 'react/jsx-dev-runtime.js',
      'react/jsx-runtime': 'react/jsx-runtime.js'
    }
  },
  plugins: [
    new MiniCssExtractPlugin({
      ignoreOrder: true,
      filename: '[name].[contenthash:6].css',
      chunkFilename: '[name].[id].[contenthash:6].css'
    }),
    new HTMLWebpackPlugin({
      template: 'src/index.html',
      filename: 'index.html',
      favicon: 'src/favicon.svg',
      minify: false,
      templateParameters: {}
    }),
    new HTMLWebpackPlugin({
      template: 'src/index_public.html',
      filename: 'index_public.html',
      favicon: 'src/favicon.svg',
      minify: false,
      templateParameters: {}
    }),
    moduleFederationPlugin,
    new DefinePlugin({
      'process.env': '{}', // required for @blueprintjs/core
      __DEV__: DEV
    }),
    new GenerateStringTypesPlugin(),
    new RetryChunkLoadPlugin({
      maxRetries: 5
    }),
    new MonacoWebpackPlugin({
      // available options are documented at https://github.com/Microsoft/monaco-editor-webpack-plugin#options
      languages: [
        'abap',
        'apex',
        'azcli',
        'bat',
        'bicep',
        'cameligo',
        'clojure',
        'coffee',
        'cpp',
        'csharp',
        'csp',
        'css',
        'cypher',
        'dart',
        'dockerfile',
        'ecl',
        'elixir',
        'flow9',
        'freemarker2',
        'fsharp',
        'go',
        'graphql',
        'handlebars',
        'hcl',
        'html',
        'ini',
        'java',
        'javascript',
        'json',
        'julia',
        'kotlin',
        'less',
        'lexon',
        'liquid',
        'lua',
        'm3',
        'markdown',
        'mips',
        'msdax',
        'mysql',
        'objective-c',
        'pascal',
        'pascaligo',
        'perl',
        'pgsql',
        'php',
        'pla',
        'postiats',
        'powerquery',
        'powershell',
        'protobuf',
        'pug',
        'python',
        'qsharp',
        'r',
        'razor',
        'redis',
        'redshift',
        'restructuredtext',
        'ruby',
        'rust',
        'sb',
        'scala',
        'scheme',
        'scss',
        'shell',
        'solidity',
        'sophia',
        'sparql',
        'sql',
        'st',
        'swift',
        'systemverilog',
        'tcl',
        'twig',
        'typescript',
        'vb',
        'wgsl',
        'xml',
        'yaml'
      ],
      globalAPI: true,
      filename: '[name].worker.[contenthash:6].js',
      customLanguages: [
        {
          label: 'yaml',
          entry: 'monaco-yaml',
          worker: {
            id: 'monaco-yaml/yamlWorker',
            entry: 'monaco-yaml/yaml.worker'
          }
        }
      ]
    })
  ]
}
