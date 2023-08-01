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
