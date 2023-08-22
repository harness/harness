import React, { useState } from 'react'
import {
  Container,
  Layout,
  Text,
  Button,
  ButtonVariation,
  FormInput,
  Formik,
  useToaster,
  ButtonSize
} from '@harnessio/uicore'
import { Color, Intent } from '@harnessio/design-system'
import { useMutate } from 'restful-react'
import { ACCESS_MODES, permissionProps, voidFn } from 'utils/Utils'
import { useStrings } from 'framework/strings'
import type { TypesRepository } from 'services/code'
import { useAppContext } from 'AppContext'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import useDeleteRepoModal from './DeleteRepoModal/DeleteRepoModal'
import css from '../RepositorySettings.module.scss'

interface GeneralSettingsProps {
  repoMetadata: TypesRepository | undefined
  refetch: () => void
}

const GeneralSettingsContent = (props: GeneralSettingsProps) => {
  const { repoMetadata, refetch } = props
  const { openModal: openDeleteRepoModal } = useDeleteRepoModal()

  const [editDesc, setEditDesc] = useState(ACCESS_MODES.VIEW)
  const { showError, showSuccess } = useToaster()

  const space = useGetSpaceParam()
  const { standalone } = useAppContext()
  const { hooks } = useAppContext()
  const { getString } = useStrings()
  const { mutate } = useMutate({
    verb: 'PATCH',
    path: `/api/v1/repos/${repoMetadata?.path}/+/`
  })

  const permEditResult = hooks?.usePermissionTranslate?.(
    {
      resource: {
        resourceType: 'CODE_REPOSITORY'
      },
      permissions: ['code_repo_edit']
    },
    [space]
  )
  const permDeleteResult = hooks?.usePermissionTranslate?.(
    {
      resource: {
        resourceType: 'CODE_REPOSITORY'
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
                    openDeleteRepoModal()
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
