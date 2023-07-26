import React, { useState } from 'react'
import {
  Button,
  Color,
  Text,
  Container,
  FormInput,
  Formik,
  Layout,
  Page,
  ButtonVariation,
  ButtonSize,
  Intent
} from '@harness/uicore'
import { useGetSpace } from 'services/code'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { ACCESS_MODES, voidFn } from 'utils/Utils'
import { useStrings } from 'framework/strings'
import css from './SpaceSettings.module.scss'

export default function SpaceSettings() {
  const { space } = useGetRepositoryMetadata()
  const { data } = useGetSpace({ space_ref: encodeURIComponent(space), lazy: !space })
  const [editName, setEditName] = useState(ACCESS_MODES.VIEW)

  const [editDesc, setEditDesc] = useState(ACCESS_MODES.VIEW)
  const { getString } = useStrings()

  return (
    <Container className={css.mainCtn}>
      <Page.Header title={getString('spaceSetting.setting')} />
      <Page.Body>
        <Container padding="xlarge">
          <Formik
            formName="spaceGeneralSettings"
            initialValues={{
              name: data?.uid,
              desc: data?.description
            }}
            onSubmit={voidFn(() => {})}>
            {formik => {
              return (
                <Layout.Vertical padding={{ top: 'medium' }}>
                  <Container padding="large" margin={{ bottom: 'medium' }} className={css.generalContainer}>
                    <Layout.Horizontal padding={{ bottom: 'medium' }}>
                      <Container className={css.label}>
                        <Text padding={{ top: 'small' }} color={Color.GREY_600} font={{ size: 'small' }}>
                          {getString('name')}
                        </Text>
                      </Container>
                      <Container className={css.content}>
                        {editName === ACCESS_MODES.EDIT ? (
                          <Layout.Horizontal>
                            <FormInput.Text name="name" className={css.textContainer} />
                            <Layout.Horizontal className={css.buttonContainer}>
                              <Button
                                className={css.saveBtn}
                                margin={{ right: 'medium' }}
                                type="submit"
                                text={getString('save')}
                                variation={ButtonVariation.SECONDARY}
                                size={ButtonSize.SMALL}
                                onClick={() => {
                                  // mutate({ description: formik.values?.name })
                                  //   .then(() => {
                                  //     showSuccess(getString('spaceUpdate'))
                                  //   })
                                  //   .catch(err => {
                                  //     showError(err)
                                  //   })
                                  setEditName(ACCESS_MODES.VIEW)
                                  // refetch()
                                }}
                              />
                              <Button
                                text={getString('cancel')}
                                variation={ButtonVariation.TERTIARY}
                                size={ButtonSize.SMALL}
                                onClick={() => {
                                  setEditName(ACCESS_MODES.VIEW)
                                }}
                              />
                            </Layout.Horizontal>
                          </Layout.Horizontal>
                        ) : (
                          <Text color={Color.GREY_800} font={{ size: 'small' }}>
                            {formik?.values?.name || data?.uid}
                            <Button
                              text={getString('edit')}
                              icon="Edit"
                              variation={ButtonVariation.LINK}
                              onClick={() => {
                                setEditName(ACCESS_MODES.EDIT)
                              }}
                              // {...permissionProps(permEditResult, standalone)}
                            />
                          </Text>
                        )}
                      </Container>
                    </Layout.Horizontal>
                    <Layout.Horizontal>
                      <Container className={css.label}>
                        <Text padding={{ top: 'small' }} color={Color.GREY_600} font={{ size: 'small' }}>
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
                                  // mutate({ description: formik.values?.desc })
                                  //   .then(() => {
                                  //     showSuccess(getString('repoUpdate'))
                                  //   })
                                  //   .catch(err => {
                                  //     showError(err)
                                  //   })
                                  setEditDesc(ACCESS_MODES.VIEW)
                                  // refetch()
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
                            {formik?.values?.desc || data?.description}
                            <Button
                              text={getString('edit')}
                              icon="Edit"
                              variation={ButtonVariation.LINK}
                              onClick={() => {
                                setEditDesc(ACCESS_MODES.EDIT)
                              }}
                              // {...permissionProps(permEditResult, standalone)}
                            />
                          </Text>
                        )}
                      </Container>
                    </Layout.Horizontal>
                  </Container>
                  <Container padding="medium" className={css.generalContainer}>
                    <Container className={css.deleteContainer}>
                      <Layout.Vertical className={css.verticalContainer}>
                        <Text icon="main-trash" color={Color.GREY_600} font={{ size: 'small' }}>
                          {getString('dangerDeleteRepo')}
                        </Text>
                        <Layout.Horizontal padding={{ top: 'medium' }} flex={{ justifyContent: 'space-between' }}>
                          <Container className={css.yellowContainer}>
                            <Text
                              icon="main-issue"
                              iconProps={{ size: 16, color: Color.ORANGE_700 }}
                              padding={{ left: 'large', right: 'large', top: 'small', bottom: 'small' }}
                              color={Color.WARNING}>
                              {getString('spaceSetting.intentText', {
                                space: data?.uid
                              })}
                            </Text>
                          </Container>
                          <Button
                            disabled={true} // TODO: Disable until backend has soft delete
                            intent={Intent.DANGER}
                            onClick={() => {
                              // confirmDeleteBranch()
                            }}
                            variation={ButtonVariation.SECONDARY}
                            text={getString('deleteSpace')}
                            // {...permissionProps(permDeleteResult, standalone)}
                          ></Button>
                        </Layout.Horizontal>
                      </Layout.Vertical>
                    </Container>
                  </Container>
                </Layout.Vertical>
              )
            }}
          </Formik>
        </Container>
      </Page.Body>
    </Container>
  )
}
