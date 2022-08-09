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