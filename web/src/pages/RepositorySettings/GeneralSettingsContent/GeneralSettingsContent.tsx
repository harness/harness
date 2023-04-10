import React, { useState } from 'react'
import {
  Color,
  Container,
  Layout,
  Text,
  Button,
  ButtonVariation,
  Intent,
  FormInput,
  Formik,
  useToaster,
  ButtonSize,
  StringSubstitute
} from '@harness/uicore'
import { useHistory } from 'react-router-dom'
import { useMutate } from 'restful-react'
import { getErrorMessage, permissionProps, voidFn } from 'utils/Utils'
import { useStrings } from 'framework/strings'
import type { TypesRepository } from 'services/code'
import { useConfirmAction } from 'hooks/useConfirmAction'
import { useAppContext } from 'AppContext'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import css from '../RepositorySettings.module.scss'

enum ACCESS_MODES {
  VIEW,
  EDIT
}

interface GeneralSettingsProps {
  repoMetadata: TypesRepository | undefined
  refetch: () => void
}

const GeneralSettingsContent = (props: GeneralSettingsProps) => {
  const { repoMetadata, refetch } = props
  const [editDesc, setEditDesc] = useState(ACCESS_MODES.VIEW)
  const { showError, showSuccess } = useToaster()
  const history = useHistory()
  const { routes } = useAppContext()
  const space = useGetSpaceParam()
  const { standalone } = useAppContext()
  const { hooks } = useAppContext()
  const { getString } = useStrings()
  const { mutate } = useMutate({
    verb: 'PATCH',
    path: `/api/v1/repos/${repoMetadata?.path}/+/`
  })
  const { mutate: deleteRepo } = useMutate({
    verb: 'DELETE',
    path: `/api/v1/repos/${repoMetadata?.path}/+/`
  })
  const contentText = () => (
    <Text>
      {' '}
      <StringSubstitute
        str={getString('deleteRepoText')}
        vars={{
          REPONAME: <strong>{repoMetadata?.uid}</strong>
        }}
      />
    </Text>
  )
  const confirmDeleteBranch = useConfirmAction({
    title: getString('deleteRepoTitle'),
    confirmText: getString('confirm'),
    intent: Intent.DANGER,
    message: contentText(),
    action: async () => {
      deleteRepo({})
        .then(() => {
          showSuccess(getString('repoDeleted', { repo: repoMetadata?.uid }), 5000)
          history.push(routes.toCODERepositories({ space }))
        })
        .catch((error: any) => {
          showError(getErrorMessage(error), 0, 'failedToDeleteBranch')
        })
    }
  })
  const permEditResult = hooks?.usePermissionTranslate?.(
    {
      resource: {
        resourceType: 'CODE_REPO'
      },
      permissions: ['code_repo_edit']
    },
    [space]
  )
  const permDeleteResult = hooks?.usePermissionTranslate?.(
    {
      resource: {
        resourceType: 'CODE_REPO'
      },
      permissions: ['code_repo_delete']
    },
    [space]
  )

  return (
    <Formik
      formName="repoGeneralSettings"
      initialValues={{
        name: repoMetadata?.uid,
        desc: repoMetadata?.description
      }}
      onSubmit={voidFn(mutate)}>
      {formik => {
        return (
          <Layout.Vertical padding={{ top: 'medium' }}>
            <Container padding="large" margin={{ bottom: 'medium' }} className={css.generalContainer}>
              <Layout.Horizontal padding={{ bottom: 'medium' }}>
                <Container className={css.label}>
                  <Text color={Color.GREY_600} font={{ size: 'small' }}>
                    {getString('repositoryName')}
                  </Text>
                </Container>
                <Container className={css.content}>
                  <Text color={Color.GREY_800} font={{ size: 'small' }}>
                    {repoMetadata?.uid}
                  </Text>
                </Container>
              </Layout.Horizontal>
              <Layout.Horizontal>
                <Container className={css.label}>
                  <Text color={Color.GREY_600} font={{ size: 'small' }}>
                    {getString('description')}
                  </Text>
                </Container>
                <Container className={css.content}>
                  {editDesc === ACCESS_MODES.EDIT ? (
                    <Layout.Horizontal>
                      <FormInput.Text name="desc" className={css.textContainer} />
                      <Layout.Horizontal className={css.buttonContainer}>
                        <Button
                          className={css.saveBtn}
                          margin={{ right: 'medium' }}
                          type="submit"
                          text={getString('save')}
                          variation={ButtonVariation.SECONDARY}
                          size={ButtonSize.SMALL}
                          onClick={() => {
                            mutate({ description: formik.values?.desc })
                              .then(() => {
                                showSuccess(getString('repoUpdate'))
                              })
                              .catch(err => {
                                showError(err)
                              })
                            setEditDesc(ACCESS_MODES.VIEW)
                            refetch()
                          }}
                        />
                        <Button
                          text={getString('cancel')}
                          variation={ButtonVariation.TERTIARY}
                          size={ButtonSize.SMALL}
                          onClick={() => {
                            setEditDesc(ACCESS_MODES.VIEW)
                          }}
                        />
                      </Layout.Horizontal>
                    </Layout.Horizontal>
                  ) : (
                    <Text color={Color.GREY_800} font={{ size: 'small' }}>
                      {formik?.values?.desc || repoMetadata?.description}
                      <Button
                        text={getString('edit')}
                        icon="Edit"
                        variation={ButtonVariation.LINK}
                        onClick={() => {
                          setEditDesc(ACCESS_MODES.EDIT)
                        }}
                        {...permissionProps(permEditResult, standalone)}
                      />
                    </Text>
                  )}
                </Container>
              </Layout.Horizontal>
            </Container>
            <Container padding="medium" className={css.generalContainer}>
              <Container className={css.deleteContainer}>
                <Text icon="main-trash" color={Color.GREY_600} font={{ size: 'small' }}>
                  {getString('dangerDeleteRepo')}
                </Text>
                <Button
                  intent={Intent.DANGER}
                  onClick={() => {
                    confirmDeleteBranch()
                  }}
                  variation={ButtonVariation.SECONDARY}
                  text={getString('delete')}
                  {...permissionProps(permDeleteResult, standalone)}></Button>
              </Container>
            </Container>
          </Layout.Vertical>
        )
      }}
    </Formik>
  )
}

export default GeneralSettingsContent
