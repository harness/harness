/* eslint-disable @typescript-eslint/no-var-requires, no-console  */
const packageJson = require('./package.json')
const deps = packageJson.dependencies
const { pick, omit, mapValues } = require('lodash')
const buildVersion = JSON.stringify(packageJson.version)
const webpack = require('webpack')
const path = require('path')
const fs = require('fs')
const devServerProxyConfig = require('./webpack.devServerProxy.config')

const ForkTsCheckerWebpackPlugin = require('fork-ts-checker-webpack-plugin')
const CircularDependencyPlugin = require('circular-dependency-plugin')
const TsconfigPathsPlugin = require('tsconfig-paths-webpack-plugin')
const MiniCssExtractPlugin = require('mini-css-extract-plugin')
const HTMLWebpackPlugin = require('html-webpack-plugin')
const BundleAnalyzerPlugin = require('webpack-bundle-analyzer').BundleAnalyzerPlugin
const JSONGeneratorPlugin = require('@harness/jarvis/lib/webpack/json-generator-plugin').default
const ModuleFederationPlugin = require('webpack/lib/container/ModuleFederationPlugin')
const MonacoWebpackPlugin = require('monaco-editor-webpack-plugin')
const GenerateStringTypesPlugin = require('./scripts/webpack/GenerateStringTypesPlugin').GenerateStringTypesPlugin

/**
 * These packages must be stricly shared with exact versions
 */
const ExactSharedPackages = [
  'formik',
  'react',
  'react-dom',
  'react-router-dom',
  '@harness/uicore',
  '@blueprintjs/core',
  '@blueprintjs/select',
  '@blueprintjs/datetime',
  'restful-react'
]

const DEV = process.env.NODE_ENV === 'development'
const SOURCE_MAP = process.env.SOURCE_MAP === 'true'
const ON_PREM = `${process.env.ON_PREM}` === 'true'
const CONTEXT = process.cwd()
const config = {
  context: CONTEXT,
  entry: './src/index.tsx',
  target: 'web',
  mode: DEV ? 'development' : 'production',
  output: {
    publicPath: 'auto',
    filename: DEV ? 'static/[name].js' : 'static/[name].[contenthash:6].js',
    chunkFilename: DEV ? 'static/[name].[id].js' : 'static/[name].[id].[contenthash:6].js',
    pathinfo: false
  },
  devtool: DEV || SOURCE_MAP ? 'cheap-module-source-map' : 'hidden-source-map',
  devServer: DEV
    ? {
        port: 3002,
        proxy: Object.fromEntries(
          Object.entries(devServerProxyConfig).map(([key, value]) => [
            key,
            Object.assign({ logLevel: 'debug', secure: false, changeOrigin: true }, value)
          ])
        ),
        client: {
          overlay: false
        }
      }
    : undefined,
  stats: {
    modules: false,
    children: false
  },
  cache: DEV ? { type: 'filesystem' } : false,
  module: {
    rules: [
      {
        test: /\.m?js$/,
        include: /node_modules/,
        type: 'javascript/auto'
      },
      {
        test: /\.(j|t)sx?$/,
        exclude: /node_modules/,
        use: [
          {
            loader: 'ts-loader',
            options: {
              transpileOnly: true
            }
          }
        ]
      },
      {
        test: /\.module\.scss$/,
        exclude: /node_modules/,
        use: [
          MiniCssExtractPlugin.loader,
          {
            loader: '@harness/css-types-loader',
            options: {
              prettierConfig: CONTEXT
            }
          },
          {
            loader: 'css-loader',
            options: {
              importLoaders: 1,
              modules: {
                mode: 'local',
                localIdentName: DEV ? '[name]_[local]_[hash:base64:6]' : '[hash:base64:6]',
                exportLocalsConvention: 'camelCaseOnly'
              }
            }
          },
          {
            loader: 'sass-loader',
            options: {
              sassOptions: {
                includePaths: [path.join(CONTEXT, 'src')]
              },
              sourceMap: false,
              implementation: require('sass')
            }
          }
        ]
      },
      {
        test: /(?<!\.module)\.scss$/,
        exclude: /node_modules/,
        use: [
          MiniCssExtractPlugin.loader,
          {
            loader: 'css-loader',
            options: {
              importLoaders: 1,
              modules: false
            }
          },
          {
            loader: 'sass-loader',
            options: {
              sassOptions: {
                includePaths: [path.join(CONTEXT, 'src')]
              },
              implementation: require('sass')
            }
          }
        ]
      },
      {
        test: /\.(jpg|jpeg|png|svg|gif)$/,
        use: [
          {
            loader: 'url-loader',
            options: {
              limit: 2000,
              fallback: 'file-loader'
            }
          }
        ]
      },
      {
        test: /\.css$/,
        use: ['style-loader', 'css-loader']
      },
      {
        test: /\.ttf$/,
        loader: 'file-loader'
      },
      {
        test: /\.ya?ml$/,
        type: 'json',
        use: [
          {
            loader: 'yaml-loader'
          }
        ]
      },
      {
        test: /\.gql$/,
        type: 'asset/source'
      },
      {
        test: /\.(mp4)$/,
        use: [
          {
            loader: 'file-loader'
          }
        ]
      }
    ]
  },
  resolve: {
    extensions: ['.mjs', '.js', '.ts', '.tsx', '.json', '.ttf', '.scss'],
    plugins: [new TsconfigPathsPlugin()]
  },
  optimization: {
    splitChunks: {
      chunks: 'all'
    }
  }
}

const commonPlugins = [
  new MiniCssExtractPlugin({
    filename: DEV ? 'static/[name].css' : 'static/[name].[contenthash:6].css',
    chunkFilename: DEV ? 'static/[name].[id].css' : 'static/[name].[id].[contenthash:6].css'
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
  new webpack.ContextReplacementPlugin(/moment[\/\\]locale$/, /en/),
  new webpack.DefinePlugin({
    'process.env': '{}', // required for @blueprintjs/core
    __DEV__: DEV,
    __ON_PREM__: ON_PREM
  }),
  new MonacoWebpackPlugin({
    // available options are documented at https://github.com/Microsoft/monaco-editor-webpack-plugin#options
    languages: ['yaml', 'json']
  }),
  new GenerateStringTypesPlugin(),

  new ModuleFederationPlugin({
    name: 'samplemodule',
    filename: 'remoteEntry.js',
    exposes: {
      './App': './src/App.tsx',
      './TestView': './src/views/TestView/TestView.tsx'
    },
    shared: Object.assign(
      {},
      mapValues(pick(deps, ExactSharedPackages), version => ({
        singleton: true,
        requiredVersion: version
      }))
    )
  })
]

const devOnlyPlugins = [
  new webpack.WatchIgnorePlugin({
    paths: [/node_modules(?!\/@harness)/, /\.d\.ts$/, /stringTypes\.ts/]
  }),
  new ForkTsCheckerWebpackPlugin()
]

const prodOnlyPlugins = [
  new JSONGeneratorPlugin({
    content: {
      version: packageJson.version,
      gitCommit: process.env.GIT_COMMIT,
      gitBranch: process.env.GIT_BRANCH
    },
    filename: 'static/version.json'
  }),
  new CircularDependencyPlugin({
    exclude: /node_modules/,
    failOnError: true
  })
]

config.plugins = commonPlugins.concat(DEV ? devOnlyPlugins : prodOnlyPlugins)

module.exports = config
