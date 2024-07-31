import React, { useCallback, useState } from 'react'
import { get, debounce } from 'lodash-es'
import cx from 'classnames'
import { FormikForm, Layout, FormInput, Container, Text } from '@harnessio/uicore'
import { useFormikContext } from 'formik'
import { Color } from '@harnessio/design-system'
import { useHistory } from 'react-router-dom'
import { Icon } from '@harnessio/icons'
import { useStrings } from 'framework/strings'
import {
  getRepoIdFromURL,
  getRepoNameFromURL,
  isValidUrl
} from 'cde/components/CreateGitspace/components/SelectRepository/SelectRepository.utils'
import { BranchInput } from 'cde/components/CreateGitspace/components/BranchInput/BranchInput'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import NewRepoModalButton from 'components/NewRepoModalButton/NewRepoModalButton'
import { RepoCreationType } from 'utils/GitUtils'
import { useAppContext } from 'AppContext'
import { OpenapiCreateGitspaceRequest, useGitspacelookup } from 'cde-gitness/services'
import { EnumGitspaceCodeRepoType } from 'cde-gitness/constants'
import css from './ThirdPartyRepoImportForm.module.scss'

enum RepoCheckStatus {
  Valid = 'valid',
  InValid = 'InValid'
}

export const ThirdPartyRepoImportForm = () => {
  const { getString } = useStrings()
  const history = useHistory()
  const space = useGetSpaceParam()
  const { routes } = useAppContext()
  const { setValues, setFieldError } = useFormikContext<OpenapiCreateGitspaceRequest>()

  const { mutate, loading } = useGitspacelookup({})

  const [repoCheckState, setRepoCheckState] = useState<RepoCheckStatus | undefined>()

  const onChange = useCallback(
    debounce(async (url: string) => {
      let errorMessage = ''
      try {
        if (isValidUrl(url)) {
          const response = (await mutate({ space_ref: space, url })) as {
            is_private?: boolean
            branch: string
            url: string
          }
          if (response?.is_private) {
            errorMessage = getString('cde.repository.privateRepoWarning')
            setRepoCheckState(RepoCheckStatus.InValid)
          } else {
            setValues((prvValues: any) => {
              return {
                ...prvValues,
                code_repo_url: response.url,
                branch: response.branch,
                identifier: getRepoIdFromURL(response.url),
                name: getRepoNameFromURL(response.url),
                code_repo_type: EnumGitspaceCodeRepoType.UNKNOWN
              }
            })
            setRepoCheckState(RepoCheckStatus.Valid)
          }
        } else {
          if (url?.trim()?.length) {
            errorMessage = 'Invalid URL Format'
            setRepoCheckState(RepoCheckStatus.InValid)
          } else {
            if (repoCheckState) {
              setRepoCheckState(undefined)
            }
          }
        }
      } catch (err) {
        errorMessage = get(err, 'message') || ''
      }
      setFieldError('code_repo_url', errorMessage)
    }, 1000),
    [repoCheckState]
  )

  return (
    <FormikForm>
      <Layout.Horizontal spacing="small">
        <Text
          icon="warning-icon"
          font={{ size: 'small' }}
          margin={{ bottom: 'medium' }}
          iconProps={{ size: 20, color: Color.ORANGE_500 }}
          background={Color.ORANGE_50}
          padding="small">
          {getString('cde.create.importWarning')}
          {
            <NewRepoModalButton
              space={space}
              repoCreationType={RepoCreationType.IMPORT}
              customRenderer={fn => (
                <Text className={css.importForm} color={Color.PRIMARY_7} onClick={fn}>
                  {getString('cde.importInto')}
                </Text>
              )}
              modalTitle={getString('importGitRepo')}
              onSubmit={() => {
                history.push(routes.toCDEGitspacesCreate({ space }))
              }}
            />
          }
        </Text>
      </Layout.Horizontal>
      <Layout.Horizontal spacing="medium">
        <Container width="63%" className={css.formFields}>
          <FormInput.Text
            name="code_repo_url"
            inputGroup={{
              leftIcon: 'git-repo',
              color: Color.GREY_500,
              rightElement: (
                <Container height={50} width={25} flex={{ alignItems: 'center' }}>
                  {loading ? (
                    <Icon name="loading" />
                  ) : repoCheckState ? (
                    repoCheckState === RepoCheckStatus.Valid ? (
                      <Icon name="tick-circle" color={Color.GREEN_450} />
                    ) : (
                      <Icon name="warning-sign" color={Color.ERROR} />
                    )
                  ) : undefined}
                </Container>
              )
            }}
            placeholder={getString('cde.repository.repositoryURL')}
            className={cx(css.repoInput)}
            onChange={async event => {
              const target = event.target as HTMLInputElement
              await onChange(target.value)
            }}
          />
        </Container>
        <Container width="35%" className={css.formFields}>
          <BranchInput />
        </Container>
      </Layout.Horizontal>
    </FormikForm>
  )
}
