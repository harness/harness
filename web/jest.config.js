process.env.TZ = 'GMT'

const { compilerOptions } = require('./tsconfig')

module.exports = {
  globals: {
    'ts-jest': {
      isolatedModules: true,
      diagnostics: false
    },
    __DEV__: false
  },
  setupFilesAfterEnv: ['<rootDir>/scripts/jest/setup-file.js'],
  collectCoverageFrom: [
    'src/**/*.{ts,tsx}',
    '!src/index.tsx',
    '!src/App.tsx',
    '!src/bootstrap.tsx',
    '!src/framework/strings/**',
    '!src/services/**',
    '!src/**/*.d.ts',
    '!src/**/*.test.{ts,tsx}',
    '!src/**/*.stories.{ts,tsx}',
    '!src/**/__test__/**',
    '!src/**/__tests__/**',
    '!src/utils/test/**',
    '!src/AppUtils.ts'
  ],
  coverageReporters: ['lcov', 'json-summary'],
  transform: {
    '^.+\\.tsx?$': 'ts-jest',
    '^.+\\.js$': 'ts-jest',
    '^.+\\.ya?ml$': '<rootDir>/scripts/jest/yaml-transform.js',
    '^.+\\.gql$': '<rootDir>/scripts/jest/gql-loader.js'
  },
  moduleDirectories: ['node_modules', 'src'],
  testMatch: ['**/?(*.)+(spec|test).[jt]s?(x)'],
  moduleNameMapper: {
    '\\.s?css$': 'identity-obj-proxy',
    'monaco-editor': '<rootDir>/node_modules/react-monaco-editor',
    '\\.(jpg|jpeg|png|gif|svg|eot|otf|webp|ttf|woff|woff2|mp4|webm|wav|mp3|m4a|aac|oga)$':
      '<rootDir>/scripts/jest/file-mock.js'
  },
  coverageThreshold: {
    global: {
      statements: 60,
      branches: 40,
      functions: 40,
      lines: 60
    }
  },
  transformIgnorePatterns: ['node_modules/(?!(date-fns|lodash-es|p-debounce)/)'],
  testPathIgnorePatterns: ['<rootDir>/dist']
}
