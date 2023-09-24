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
const path = require('path')
const HTMLWebpackPlugin = require('html-webpack-plugin')
const MiniCssExtractPlugin = require('mini-css-extract-plugin')
const { DefinePlugin } = require('webpack')

const commonConfig = require('./webpack.common')
const CONTEXT = process.cwd()

const prodConfig = {
  context: CONTEXT,
  entry: path.resolve(CONTEXT, '/src/index.tsx'),
  mode: 'production',
  devtool: process.env.ENABLE_SOURCE_MAP ? 'source-map' : false,
  output: {
    filename: '[name].[contenthash:6].js',
    chunkFilename: '[name].[id].[contenthash:6].js'
  },
  optimization: {
    splitChunks: {
      chunks: 'all',
      minSize: 51200,
      cacheGroups: {
        commons: {
          test: /[\\/]node_modules[\\/]/,
          name: 'vendors',
          chunks: 'all',
          maxSize: 1e7
        },
        blueprintjs: {
          test: /[\\/]node_modules[\\/](@blueprintjs)[\\/]/,
          name: 'vendor-blueprintjs',
          chunks: 'all',
          priority: 10
        }
      }
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
    })
  ]
}

module.exports = merge(commonConfig, prodConfig)
