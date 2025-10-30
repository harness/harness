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

require('dotenv').config()

const { merge } = require('webpack-merge')
const commonConfig = require('./webpack.common')

const API_URL = process.env.API_URL ?? 'http://localhost:3000'
const HOST = 'localhost'
const PORT = process.env.PORT ?? 3020
const STANDALONE = process.env.STANDALONE === 'true'
const CONTEXT = process.cwd()
const prodConfig = require('./webpack.prod')

console.info(`Starting development build... http://${HOST}:${PORT}`)
console.info('Environment variables:')
console.table({ STANDALONE, HOST, PORT, API_URL })

const devConfig = {
  mode: 'development',
  entry: path.resolve(CONTEXT, '/src/index.tsx'),
  devtool: 'cheap-module-source-map',
  cache: { type: 'filesystem' },
  output: {
    publicPath: STANDALONE ? '/' : 'auto'
  },
  optimization: STANDALONE
    ? {
        runtimeChunk: 'single',
        splitChunks: prodConfig.optimization.splitChunks
      }
    : {},
  devServer: {
    hot: true,
    host: HOST,
    historyApiFallback: {
      disableDotRule: true
    },
    port: PORT,
    proxy: {
      '/api': {
        target: API_URL,
        logLevel: 'debug',
        secure: false,
        changeOrigin: true
      }
    },
    compress: false
  }
}

module.exports = merge(commonConfig, devConfig)
