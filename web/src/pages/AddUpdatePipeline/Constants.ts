export enum YamlVersion {
  V0,
  V1
}

export const DEFAULT_YAML_PATH_PREFIX = '.harness/'
export const DEFAULT_YAML_PATH_SUFFIX = '.yaml'

export const DRONE_CONFIG_YAML_FILE_SUFFIXES = ['.drone.yml', '.drone.yaml']

export const V1_SCHEMA_YAML_FILE_REGEX = /^(.*v1\.ya?ml)$/i
