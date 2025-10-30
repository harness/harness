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
const { merge } = require('webpack-merge')
const { DefinePlugin } = require('webpack')
const HTMLWebpackPlugin = require('html-webpack-plugin')
const MiniCssExtractPlugin = require('mini-css-extract-plugin')
const { RetryChunkLoadPlugin } = require('webpack-retry-chunk-load-plugin')
const BundleAnalyzerPlugin = require('webpack-bundle-analyzer').BundleAnalyzerPlugin

const enableBundleAnalyser = process.env.ENABLE_BUNDLE_ANALYSER === 'true'
const commonConfig = require('./webpack.common')
const CONTEXT = process.cwd()

const prodConfig = {
  mode: 'production',
  devtool: 'hidden-source-map',
  output: {
    publicPath: 'auto',
    pathinfo: false,
    filename: '[name].[contenthash:6].js',
    chunkFilename: '[name].[id].[contenthash:6].js'
  },
  plugins: [
    new DefinePlugin({
      __DEV__: false
    }),
    new MiniCssExtractPlugin({
      filename: '[name].[contenthash:6].css',
      chunkFilename: '[name].[id].[contenthash:6].css'
    }),
    new HTMLWebpackPlugin({
      template: 'src/ar/index.html',
      filename: '../index.html',
      minify: false,
      templateParameters: {
        __DEV__: false
      }
    }),
    new MiniCssExtractPlugin({
      filename: '[name].[contenthash:6].css',
      chunkFilename: '[name].[id].[contenthash:6].css'
    }),
    new RetryChunkLoadPlugin({
      maxRetries: 5
    })
  ]
}

if (enableBundleAnalyser) {
  prodConfig.plugins.push(new BundleAnalyzerPlugin())
}

module.exports = merge(commonConfig, prodConfig)
