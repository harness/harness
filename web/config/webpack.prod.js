const { merge } = require('webpack-merge')
const HTMLWebpackPlugin = require('html-webpack-plugin')
const MiniCssExtractPlugin = require('mini-css-extract-plugin')
const CircularDependencyPlugin = require('circular-dependency-plugin')
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
      filename: '[name].[contenthash:6].css',
      chunkFilename: '[name].[id].[contenthash:6].css'
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
