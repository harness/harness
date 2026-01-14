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

const { merge } = require('webpack-merge')
const commonConfig = require('./webpack.common')
const CONTEXT = process.cwd()
const path = require('path')

const prodConfig = {
  mode: 'production',
  entry: path.resolve(CONTEXT, '/src/index.tsx'),
  devtool: process.env.ENABLE_SOURCE_MAP ? 'source-map' : false,
  optimization: {
    splitChunks: {
      minSize: 20_480,
      automaticNameDelimiter: '-',

      cacheGroups: {
        common: {
          test: /[\\/]node_modules[\\/]/,
          priority: -5,
          reuseExistingChunk: true,
          chunks: 'initial',
          name: 'vendor-common',
          minSize: 20_480,
          maxSize: 1_024_000
        },

        default: {
          minChunks: 2,
          priority: -10,
          reuseExistingChunk: true,
          minSize: 20_480,
          maxSize: 1_024_000
        },

        // Opting out of defaultVendors, so rest of the node modules will be part of default cacheGroup
        defaultVendors: false,

        react: {
          test: /[\\/]node_modules[\\/](react)[\\/]/,
          name: 'vendor-react',
          chunks: 'all',
          priority: 50,
          minSize: 0
        },

        reactdom: {
          test: /[\\/]node_modules[\\/](react-dom)[\\/]/,
          name: 'vendor-react-dom',
          chunks: 'all',
          priority: 40
        },

        reactrouterdom: {
          test: /[\\/]node_modules[\\/](react-router-dom)[\\/]/,
          name: 'vendor-react-router-dom',
          chunks: 'all',
          priority: 30,
          minSize: 0
        },

        blueprintjs: {
          test: /[\\/]node_modules[\\/](@blueprintjs)[\\/]/,
          name: 'vendor-blueprintjs',
          chunks: 'all',
          priority: 20
        },

        restfulreact: {
          test: /[\\/]node_modules[\\/](restful-react)[\\/]/,
          name: 'vendor-restful-react',
          chunks: 'all',
          priority: 10
        },

        designsystem: {
          test: /[\\/]node_modules[\\/](@harnessio\/design-system)[\\/]/,
          name: 'vendor-harnessio-design-system',
          chunks: 'all',
          priority: 5
        },

        icons: {
          test: /[\\/]node_modules[\\/](@harnessio\/icons)[\\/]/,
          name: 'vendor-harnessio-icons',
          chunks: 'all',
          priority: 1
        },

        uicore: {
          test: /[\\/]node_modules[\\/](@harnessio\/uicore)[\\/]/,
          name: 'vendor-harnessio-uicore',
          chunks: 'all',
          priority: 1
        }
      }
    }
  }
}

module.exports = merge(commonConfig, prodConfig)
