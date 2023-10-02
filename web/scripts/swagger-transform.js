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

const fs = require('fs');
const path = require('path');
const _ = require('lodash');
const yaml = require('js-yaml');
const stringify = require('fast-json-stable-stringify');

// Define a mapping of valid transform modules
const transformModules = {
  'config1': require('./src/services/config1/transform.js'),
  'config2': require('./src/services/config2/transform.js'),
  'config3': require('./src/services/config3/transform.js'),
};

module.exports = inputSchema => {
  const argv = process.argv.slice(2);
  const validConfigs = Object.keys(transformModules); // Get valid config names from the mapping

  // Ensure that a valid config is provided as an argument
  const config = argv[0];
  if (!validConfigs.includes(config)) {
    console.error('Invalid or missing config argument.');
    process.exit(1); // Terminate the script if an invalid or missing config is provided
  }

  const overridesFile = path.join('src/services', config, 'overrides.yaml');

  let paths = inputSchema.paths;

  if (fs.existsSync(overridesFile)) {
    const data = fs.readFileSync(overridesFile, 'utf8');
    const { allowpaths, operationIdOverrides } = yaml.safeLoad(data);

    if (!allowpaths.includes('*')) {
      paths = _.pick(paths, ...allowpaths);
    }

    _.forIn(operationIdOverrides, (value, key) => {
      const [path, method] = key.split('.');

      if (path && method && _.has(paths, path) && _.has(paths[path], method)) {
        _.set(paths, [path, method, 'operationId'], value);
      }
    });
  }

  inputSchema.paths = paths;

  const transformModule = transformModules[config];
  if (transformModule) {
    inputSchema = transformModule(inputSchema);
  } else {
    console.error('Invalid transform module.');
    process.exit(1); // Terminate the script if an invalid transform module is provided
  }

  // stringify and parse JSON to get a stable object
  return JSON.parse(stringify(inputSchema));
};