import React, { useMemo } from 'react'
import {
  Button,
  ButtonVariation,
  Color,
  Dialog,
  FontVariation,
  FormikForm,
  FormInput,
  Layout,
  Text
} from '@harness/uicore'
import { useModalHook } from '@harness/use-modal'
import { Formik } from 'formik'
import { useMutate } from 'restful-react'
import moment from 'moment'
import * as Yup from 'yup'

import { REGEX_VALID_REPO_NAME } from 'utils/Utils'

import { useStrings } from 'framework/strings'

const useNewToken = ({ onClose }: { onClose: () => void }) => {
  const { getString } = useStrings()
  const { mutate } = useMutate({ path: '/api/v1/user/tokens', verb: 'POST' })

  const lifeTimeOptions = useMemo(
    () => [
      { label: getString('nDays', { number: 7 }), value: 604800000000000 },
      { label: getString('nDays', { number: 30 }), value: 2592000000000000 },
      { label: getString('nDays', { number: 60 }), value: 5184000000000000 },
      { label: getString('nDays', { number: 90 }), value: 7776000000000000 }
    ],
    [getString]
  )

  const [openModal, hideModal] = useModalHook(() => {
    return (
      <Dialog isOpen enforceFocus={false} onClose={hideModal} title={getString('createNewToken')}>
        <Formik
          initialValues={{
            uid: '',
            lifeTime: 0
          }}
          validationSchema={Yup.object().shape({
            uid: Yup.string()
              .required(getString('validation.nameIsRequired'))
              .matches(REGEX_VALID_REPO_NAME, getString('validation.nameInvalid')),
            lifeTime: Yup.number().required(getString('validation.expirationDateRequired'))
          })}
          onSubmit={async values => {
            await mutate(values)
            hideModal()
            onClose()
          }}>
          {formikProps => {
            const expiresAtString = moment(Date.now() + formikProps.values.lifeTime / 1000000).format(
              'dddd, MMMM DD YYYY'
            )

            return (
              <FormikForm>
                <FormInput.Text
                  name="uid"
                  label={getString('name')}
                  placeholder={getString('newToken.namePlaceholder')}
                />
                <FormInput.Select name="lifeTime" label={getString('expiration')} items={lifeTimeOptions} usePortal />
                {formikProps.values.lifeTime ? (
                  <Text font={{ variation: FontVariation.SMALL_SEMI }} color={Color.GREY_400}>
                    {getString('newToken.expireOn', { date: expiresAtString })}
                  </Text>
                ) : null}
                <Layout.Horizontal margin={{ top: 'xxxlarge' }} spacing="medium">
                  <Button
                    text={getString('newToken.generateToken')}
                    type="submit"
                    variation={ButtonVariation.PRIMARY}
                  />
                  <Button text={getString('cancel')} onClick={hideModal} variation={ButtonVariation.TERTIARY} />
                </Layout.Horizontal>
              </FormikForm>
            )
          }}
        </Formik>
      </Dialog>
    )
  }, [])

  return {
    openModal,
    hideModal
  }
}

export default useNewToken
