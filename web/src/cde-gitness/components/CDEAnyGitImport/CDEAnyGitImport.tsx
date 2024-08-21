import React, { useCallback, useEffect, useState } from 'react'
import cx from 'classnames'
import { Container, FormikForm, FormInput, Layout } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import { Icon } from '@harnessio/icons'
import { debounce, get } from 'lodash-es'
import { useFormikContext } from 'formik'
import { Repository } from 'iconoir-react'
import { useStrings } from 'framework/strings'
import type { OpenapiCreateGitspaceRequest } from 'cde-gitness/services'
import { BranchInput } from 'cde-gitness/components/BranchInput/BranchInput'
import { useRepoLookupForGitspace } from 'services/cde'
import { useGetCDEAPIParams } from 'cde-gitness/hooks/useGetCDEAPIParams'
import { getRepoIdFromURL, getRepoNameFromURL, isValidUrl } from './CDEAnyGitImport.utils'
import css from './CDEAnyGitImport.module.scss'

enum RepoCheckStatus {
  Valid = 'valid',
  InValid = 'InValid'
}

export const CDEAnyGitImport = () => {
  const { getString } = useStrings()
  const { setValues, setFieldError, values } = useFormikContext<OpenapiCreateGitspaceRequest>()
  const { accountIdentifier = '', orgIdentifier = '', projectIdentifier = '' } = useGetCDEAPIParams()

  const { mutate, loading } = useRepoLookupForGitspace({
    accountIdentifier,
    orgIdentifier,
    projectIdentifier
  })

  const [repoCheckState, setRepoCheckState] = useState<RepoCheckStatus | undefined>()

  useEffect(() => {
    if (values?.code_repo_type) {
      setRepoCheckState(undefined)
    }
  }, [values?.code_repo_type])

  const onChange = useCallback(
    debounce(async (url: string) => {
      let errorMessage = ''
      try {
        if (isValidUrl(url)) {
          const response = (await mutate({ url, repo_type: values?.code_repo_type })) as {
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
                code_repo_type: values?.code_repo_type
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
    [repoCheckState, values?.code_repo_type]
  )

  return (
    <FormikForm>
      <Layout.Horizontal spacing="medium">
        <Container width="63%" className={css.formFields}>
          <FormInput.Text
            name="code_repo_url"
            inputGroup={{
              leftElement: (
                <Container flex={{ alignItems: 'center' }}>
                  <Repository height={32} width={32} />
                </Container>
              ),
              className: css.leftElementClassName,
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
