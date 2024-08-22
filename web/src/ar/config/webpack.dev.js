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

const path = require('path')

require('dotenv').config()

const { mergeWithRules } = require('webpack-merge')
const ForkTsCheckerWebpackPlugin = require('fork-ts-checker-webpack-plugin')
const MiniCssExtractPlugin = require('mini-css-extract-plugin')
const HTMLWebpackPlugin = require('html-webpack-plugin')

const { DefinePlugin, WatchIgnorePlugin } = require('webpack')
const commonConfig = require('./webpack.common')

const devConfig = {
  mode: 'development',
  devtool: 'cheap-module-source-map',
  cache: { type: 'filesystem' },
  output: {
    filename: '[name].js',
    chunkFilename: '[name].[id].js'
  },
  devServer: {
    historyApiFallback: {
      /**
       * Added to support routes ending with patterns .x eg. myroute/test.route
       * discussion: https://stackoverflow.com/questions/38576356/attempting-to-route-a-url-with-a-dot-leads-to-404-with-webpack-dev-server
       */
      disableDotRule: true
    },
    port: 8191,
    hot: true,
    allowedHosts: 'all',
    proxy: {
      '/api': {
        target: process.env.API_URL || 'http://localhost:9091'
      }
    },
    static: [path.join(process.cwd(), 'src/static')]
  },
  module: {
    rules: [
      {
        test: /\.module\.scss$/,
        use: [
          {
            loader: 'css-loader',
            options: {
              modules: {
                mode: 'local',
                localIdentName: '[name]_[local]_[hash:base64:6]',
                exportLocalsConvention: 'camelCaseOnly'
              }
            }
          }
        ]
      }
    ]
  },
  plugins: [
    new MiniCssExtractPlugin({
      filename: '[name].css',
      chunkFilename: '[name].[id].css'
    }),
    new HTMLWebpackPlugin({
      template: 'src/ar/index.html',
      filename: 'index.html',
      publicPath: '/',
      minify: false,
      templateParameters: {
        __DEV__: true
      }
    }),
    new DefinePlugin({
      'process.env': '{}', // required for @blueprintjs/core
      __DEV__: true,
      __ENABLE_CDN__: false
    }),
    new WatchIgnorePlugin({
      paths: [/node_modules(?!\/@harness)/, /\.(d|test)\.tsx?$/, /types\.ts/, /\.snap$/]
    }),
    new ForkTsCheckerWebpackPlugin({
      typescript: {
        memoryLimit: 6144
      }
    })
  ]
}

let mergedConfig = mergeWithRules({
  module: {
    rules: {
      test: 'match',
      use: {
        loader: 'match',
        options: 'merge'
      }
    }
  }
})(commonConfig, devConfig)

module.exports = mergedConfig
