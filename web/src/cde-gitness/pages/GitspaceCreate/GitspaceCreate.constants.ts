import type { EnumIDEType, OpenapiCreateGitspaceRequest } from 'cde-gitness/services'

export const gitnessFormInitialValues: OpenapiCreateGitspaceRequest = {
  branch: '',
  code_repo_url: '',
  identifier: '',
  ide: 'vs_code' as EnumIDEType,
  resource_identifier: 'default',
  name: ''
}
