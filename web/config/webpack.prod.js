const { merge } = require('webpack-merge')
const HTMLWebpackPlugin = require('html-webpack-plugin')
const MiniCssExtractPlugin = require('mini-css-extract-plugin')
const CircularDependencyPlugin = require('circular-dependency-plugin')
const JSONGeneratorPlugin = require('@harness/jarvis/lib/webpack/json-generator-plugin').default
const { DefinePlugin } = require('webpack')

const commonConfig = require('./webpack.common')

const ON_PREM = `${process.env.ON_PREM}` === 'true'

const prodConfig = {
  mode: 'production',
  devtool: 'source-map',
  output: {
    filename: '[name].[contenthash:6].js',
    chunkFilename: '[name].[id].[contenthash:6].js'
  },
  plugins: [
    new MiniCssExtractPlugin({
      filename: '[name].[contenthash:6].css',
      chunkFilename: '[name].[id].[contenthash:6].css'
    }),
    new JSONGeneratorPlugin({
      content: {
        version: require('../package.json').version,
        gitCommit: process.env.GIT_COMMIT,
        gitBranch: process.env.GIT_BRANCH
      },
      filename: 'version.json'
    }),
    new CircularDependencyPlugin({
      exclude: /node_modules/,
      failOnError: true
    }),
    new HTMLWebpackPlugin({
      template: 'src/index.html',
      filename: 'index.html',
      minify: false,
      templateParameters: {
        __ON_PREM__: ON_PREM
      }
    })
  ]
}

module.exports = merge(commonConfig, prodConfig)
