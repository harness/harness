import type { ThirdPartyRepoImportFormProps } from 'cde-gitness/components/ThirdPartyRepoImportForm/ThirdPartyRepoImportForm'
import type { EnumIDEType, OpenapiCreateGitspaceRequest } from 'cde-gitness/services'
import { GitProviders } from 'utils/GitUtils'

export const gitnessFormInitialValues: OpenapiCreateGitspaceRequest = {
  branch: '',
  code_repo_url: '',
  devcontainer_path: '',
  id: '',
  ide: 'vsCode' as EnumIDEType,
  infra_provider_resource_id: 'default',
  name: ''
}
export const thirdPartyformInitialValues: ThirdPartyRepoImportFormProps = {
  gitProvider: GitProviders.GITHUB,
  hostUrl: '',
  org: '',
  project: '',
  repo: '',
  username: '',
  password: '',
  name: '',
  description: '',
  branch: '',
  ide: 'vsCode' as EnumIDEType,
  id: '',
  importPipelineLabel: false
}
