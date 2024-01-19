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

const webpack = require('webpack')
const {
  container: { ModuleFederationPlugin },
  DefinePlugin
} = require('webpack')
const MiniCssExtractPlugin = require('mini-css-extract-plugin')
const TsconfigPathsPlugin = require('tsconfig-paths-webpack-plugin')
const GenerateStringTypesPlugin = require('../scripts/webpack/GenerateStringTypesPlugin').GenerateStringTypesPlugin
const { RetryChunkLoadPlugin } = require('webpack-retry-chunk-load-plugin')
const MonacoWebpackPlugin = require('monaco-editor-webpack-plugin')
const moduleFederationConfig = require('./moduleFederation.config')
const CONTEXT = process.cwd()
const DEV = process.env.NODE_ENV === 'development'

module.exports = {
  target: 'web',
  context: CONTEXT,
  stats: {
    modules: false,
    children: false
  },
  entry: {
    [moduleFederationConfig.name]: './src/public-path'
  },
  output: {
    publicPath: 'auto',
    filename: DEV ? 'static/[name].js' : 'static/[name].[contenthash:6].js',
    chunkFilename: DEV ? 'static/[name].[id].js' : 'static/[name].[id].[contenthash:6].js',
    pathinfo: false
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
        test: /\.(jpg|jpeg|png|svg|gif)$/,
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
      }
    ]
  },
  resolve: {
    extensions: ['.mjs', '.js', '.ts', '.tsx', '.json', '.ttf', '.scss'],
    plugins: [new TsconfigPathsPlugin()]
  },
  plugins: [
    new ModuleFederationPlugin(moduleFederationConfig),
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
        'cameligo',
        'clojure',
        'coffee',
        'cpp',
        'csharp',
        'csp',
        'css',
        'dockerfile',
        'fsharp',
        'go',
        'graphql',
        'handlebars',
        'html',
        'ini',
        'java',
        'javascript',
        'json',
        'kotlin',
        'less',
        'lua',
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
        'postiats',
        'powerquery',
        'powershell',
        'pug',
        'python',
        'r',
        'razor',
        'redis',
        'redshift',
        'restructuredtext',
        'ruby',
        'rust',
        'sb',
        'scheme',
        'scss',
        'shell',
        'solidity',
        'sophia',
        'sql',
        'st',
        'swift',
        'tcl',
        'twig',
        'typescript',
        'vb',
        'xml',
        'yaml',
        'hcl'
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
