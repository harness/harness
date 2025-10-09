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

const { camel } = require("case");

module.exports = ({ componentName, verb, route, description, genericsTypes, paramsInPath, paramsTypes }, basePath) => {
    const propsType = type =>
        `${type}UsingFetchProps<${genericsTypes}>${paramsInPath.length ? ` & {${paramsTypes}}` : ""}`;

    if (verb === "get") {
        return `${description}export const ${camel(componentName)}Promise = (${
            paramsInPath.length ? `{${paramsInPath.join(", ")}, ...props}` : "props"
            }: ${propsType(
                "Get",
            )}, signal?: RequestInit["signal"]) => getUsingFetch<${genericsTypes}>(${basePath}, \`${route}\`, props, signal);\n\n`
    }
    else {
        return `${description}export const ${camel(componentName)}Promise = (${
            paramsInPath.length ? `{${paramsInPath.join(", ")}, ...props}` : "props"
            }: ${propsType(
                "Mutate",
            )}, signal?: RequestInit["signal"]) => mutateUsingFetch<${genericsTypes}>("${verb.toUpperCase()}", ${basePath}, \`${route}\`, props, signal);\n\n`;
    }
}
