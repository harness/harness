import * as yup from 'yup'
import { compact } from 'lodash-es'
import type { UseStringsReturn } from 'framework/strings'
import { GitProviders, getProviderTypeMapping } from 'utils/GitUtils'
import type { ThirdPartyRepoImportFormProps } from 'cde-gitness/components/ThirdPartyRepoImportForm/ThirdPartyRepoImportForm'

export const validateGitnessForm = (getString: UseStringsReturn['getString']) =>
  yup.object().shape({
    branch: yup.string().trim().required(getString('cde.branchValidationMessage')),
    code_repo_url: yup.string().trim().required(getString('cde.repoValidationMessage')),
    id: yup.string().trim().required(),
    ide: yup.string().trim().required(),
    infra_provider_resource_id: yup.string().trim().required(getString('cde.machineValidationMessage')),
    name: yup.string().trim().required()
  })

export const validationSchemaStepOne = (getString: UseStringsReturn['getString']) =>
  yup.object().shape({
    gitProvider: yup.string().required(),
    repo: yup
      .string()
      .trim()
      .when('gitProvider', {
        is: gitProvider => [GitProviders.GITHUB, GitProviders.GITLAB, GitProviders.BITBUCKET].includes(gitProvider),
        then: yup.string().required(getString('importSpace.orgRequired'))
      }),
    branch: yup.string().trim().required(getString('cde.branchValidationMessage')),
    hostUrl: yup
      .string()
      // .matches(MATCH_REPOURL_REGEX, getString('importSpace.invalidUrl'))
      .when('gitProvider', {
        is: gitProvider =>
          ![GitProviders.GITHUB, GitProviders.GITLAB, GitProviders.BITBUCKET, GitProviders.AZURE].includes(gitProvider),
        then: yup.string().required(getString('importRepo.required')),
        otherwise: yup.string().notRequired() // Optional based on your needs
      }),
    org: yup
      .string()
      .trim()
      .when('gitProvider', {
        is: GitProviders.AZURE,
        then: yup.string().required(getString('importSpace.orgRequired'))
      }),
    project: yup
      .string()
      .trim()
      .when('gitProvider', {
        is: GitProviders.AZURE,
        then: yup.string().required(getString('importSpace.spaceNameRequired'))
      }),
    name: yup.string().trim().required(getString('validation.nameIsRequired'))
  })

export const handleImportSubmit = (formData: ThirdPartyRepoImportFormProps) => {
  const type = getProviderTypeMapping(formData.gitProvider)

  const provider = {
    type,
    username: formData.username,
    password: formData.password,
    host: ''
  }

  if (
    ![GitProviders.GITHUB, GitProviders.GITLAB, GitProviders.BITBUCKET, GitProviders.AZURE].includes(
      formData.gitProvider
    )
  ) {
    provider.host = formData.hostUrl
  }

  const importPayload = {
    name: formData.repo || formData.name,
    description: formData.description || '',
    id: formData.repo,
    provider,
    ide: formData.ide,
    branch: formData.branch,
    infra_provider_resource_id: 'default',
    provider_repo: compact([
      formData.org,
      formData.gitProvider === GitProviders.AZURE ? formData.project : '',
      formData.repo
    ])
      .join('/')
      .replace(/\.git$/, '')
  }

  return importPayload
}
