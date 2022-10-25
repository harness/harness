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
const MonacoWebpackPlugin = require('monaco-editor-webpack-plugin')
const commonConfig = require('./webpack.common')

const baseUrl = process.env.BASE_URL ?? 'https://qa.harness.io/gateway'
const targetLocalHost = JSON.parse(process.env.TARGET_LOCALHOST || 'true')

const ON_PREM = `${process.env.ON_PREM}` === 'true'
const DEV = process.env.NODE_ENV === 'development'

const devConfig = {
  mode: 'development',
  entry: './src/index.tsx',
  devtool: 'cheap-module-source-map',
  cache: { type: 'filesystem' },
  output: {
    filename: '[name].js',
    chunkFilename: '[name].[id].js'
  },
  devServer: {
    hot: true,
    host: 'localhost',
    historyApiFallback: true,
    port: 3020,
    proxy: {
      '/api': {
        target: targetLocalHost ? 'http://localhost:3000' : baseUrl,
        logLevel: 'debug',
        secure: false,
        changeOrigin: true
      }
    }
  },
  plugins: [
    new MiniCssExtractPlugin({
      filename: '[name].css',
      chunkFilename: '[name].[id].css'
    }),
    new HTMLWebpackPlugin({
      template: 'src/index.html',
      filename: 'index.html',
      minify: false,
      templateParameters: {
        __DEV__: DEV,
        __ON_PREM__: ON_PREM
      }
    }),
    new DefinePlugin({
      'process.env': '{}', // required for @blueprintjs/core
      __DEV__: DEV
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
        'yaml'
      ]
    })
    // new ForkTsCheckerWebpackPlugin()
    // new WatchIgnorePlugin({
    //   paths: [/node_modules(?!\/@wings-software)/, /\.d\.ts$/]
    // }),
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

console.table({ baseUrl, targetLocalHost })

module.exports = merge(commonConfig, devConfig)
