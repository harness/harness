const path = require('path')
const util = require('util')
const fs = require('fs')

require('dotenv').config()

const { merge } = require('webpack-merge')
const ForkTsCheckerWebpackPlugin = require('fork-ts-checker-webpack-plugin')
const MiniCssExtractPlugin = require('mini-css-extract-plugin')
const HTMLWebpackPlugin = require('html-webpack-plugin')
const {
  DefinePlugin,
  WatchIgnorePlugin,
  container: { ModuleFederationPlugin }
} = require('webpack')

const commonConfig = require('./webpack.common')
const API_URL = process.env.API_URL ?? 'http://localhost:3000'
const HOST = 'localhost'
const PORT = process.env.PORT ?? 3020
const STANDALONE = JSON.parse(process.env.STANDALONE ?? 'true')

console.info(`Starting development build... http://${HOST}:${PORT}`)
console.info('Environment variables:')
console.table({ STANDALONE, HOST, PORT, API_URL })

const devConfig = {
  mode: 'development',
  target: 'web',
  entry: './src/index.tsx',
  devtool: 'cheap-module-source-map',
  cache: { type: 'filesystem' },
  output: {
    filename: '[name].js',
    chunkFilename: '[name].[id].js',
    path: path.resolve(process.cwd(), 'dist'),
    pathinfo: false
  },
  ...(STANDALONE
    ? {
        optimization: {
          runtimeChunk: 'single'
        }
      }
    : {}),
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
  },
  plugins: [
    new MiniCssExtractPlugin({
      filename: '[name].css',
      chunkFilename: '[name].[id].css',
      ignoreOrder: true
    }),
    new HTMLWebpackPlugin({
      publicPath: '/',
      template: 'src/index.html',
      filename: 'index.html',
      favicon: 'src/favicon.svg',
      minify: false,
      templateParameters: {
        __DEV__: true
      }
    }),
    new DefinePlugin({
      'process.env': '{}', // required for @blueprintjs/core
      __DEV__: true,
      __ENABLE_CDN__: false
    })
  ],
  module: {
    rules: [
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
  }
}

module.exports = merge(commonConfig, devConfig)
