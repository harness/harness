import { FormInput, FormikForm, Layout } from '@harnessio/uicore'
import React, { useState } from 'react'
import { useFormikContext } from 'formik'
import { useStrings } from 'framework/strings'
import { GitProviders, ImportFormData, getOrgLabel, getOrgPlaceholder, getProviders } from 'utils/GitUtils'
import css from './ThirdPartyRepoImportForm.module.scss'

export interface ThirdPartyRepoImportFormProps extends ImportFormData {
  branch: string
  ide: string
  id: string
}

export const ThirdPartyRepoImportForm = () => {
  const [auth, setAuth] = useState(false)
  const { getString } = useStrings()
  const { values, setFieldValue, validateField } = useFormikContext<ThirdPartyRepoImportFormProps>()
  return (
    <FormikForm>
      <FormInput.Select name={'gitProvider'} label={getString('importSpace.gitProvider')} items={getProviders()} />
      {![GitProviders.GITHUB, GitProviders.GITLAB, GitProviders.BITBUCKET, GitProviders.AZURE].includes(
        values.gitProvider
      ) && (
        <FormInput.Text
          className={css.hideContainer}
          name="hostUrl"
          label={getString('importRepo.url')}
          placeholder={getString('importRepo.urlPlaceholder')}
          tooltipProps={{
            dataTooltipId: 'repositoryURLTextField'
          }}
        />
      )}
      <FormInput.Text
        className={css.hideContainer}
        name="org"
        label={getString(getOrgLabel(values.gitProvider))}
        placeholder={getString(getOrgPlaceholder(values.gitProvider))}
      />
      {values.gitProvider === GitProviders.AZURE && (
        <FormInput.Text
          className={css.hideContainer}
          name="project"
          label={getString('importRepo.project')}
          placeholder={getString('importRepo.projectPlaceholder')}
        />
      )}
      <Layout.Horizontal spacing="medium" flex={{ justifyContent: 'space-between', alignItems: 'baseline' }}>
        <Layout.Vertical className={css.repoAndBranch} spacing="small">
          <FormInput.Text
            className={css.hideContainer}
            name="repo"
            label={getString('importRepo.repo')}
            placeholder={getString('importRepo.repoPlaceholder')}
            onChange={event => {
              const target = event.target as HTMLInputElement
              setFieldValue('repo', target.value)
              if (target.value) {
                setFieldValue('name', target.value)
                validateField('repo')
              }
            }}
          />
        </Layout.Vertical>
        <Layout.Vertical className={css.repoAndBranch} spacing="small">
          <FormInput.Text
            className={css.hideContainer}
            name="branch"
            label={getString('branch')}
            placeholder={getString('cde.create.branchPlaceholder')}
            onChange={event => {
              const target = event.target as HTMLInputElement
              setFieldValue('branch', target.value)
            }}
          />
        </Layout.Vertical>
      </Layout.Horizontal>
      <Layout.Horizontal spacing="medium">
        <FormInput.CheckBox
          name="authorization"
          label={getString('importRepo.reqAuth')}
          tooltipProps={{
            dataTooltipId: 'authorization'
          }}
          onClick={() => {
            setAuth(!auth)
          }}
          style={auth ? {} : { margin: 0 }}
        />
      </Layout.Horizontal>

      {auth ? (
        <>
          {[GitProviders.BITBUCKET, GitProviders.AZURE].includes(values.gitProvider) && (
            <FormInput.Text
              name="username"
              label={getString('userName')}
              placeholder={getString('importRepo.userPlaceholder')}
              tooltipProps={{
                dataTooltipId: 'repositoryUserTextField'
              }}
            />
          )}
          <FormInput.Text
            inputGroup={{ type: 'password' }}
            name="password"
            label={
              [GitProviders.BITBUCKET, GitProviders.AZURE].includes(values.gitProvider)
                ? getString('importRepo.appPassword')
                : getString('importRepo.passToken')
            }
            placeholder={
              [GitProviders.BITBUCKET, GitProviders.AZURE].includes(values.gitProvider)
                ? getString('importRepo.appPasswordPlaceholder')
                : getString('importRepo.passTokenPlaceholder')
            }
            tooltipProps={{
              dataTooltipId: 'repositoryPasswordTextField'
            }}
          />
        </>
      ) : null}
    </FormikForm>
  )
}
