import {
  ButtonVariation,
  Container,
  FlexExpander,
  Formik,
  FormikForm,
  Button,
  FormInput,
  Layout,
  Text,
  FontVariation,
  useToaster
} from '@harness/uicore'
import { useMutate } from 'restful-react'
import { FormGroup } from '@blueprintjs/core'
import { useHistory } from 'react-router-dom'
import * as yup from 'yup'
import React from 'react'
import type { OpenapiUpdateWebhookRequest, OpenapiWebhookTrigger, OpenapiWebhookType } from 'services/code'
import { getErrorMessage } from 'utils/Utils'
import { useStrings } from 'framework/strings'
import type { GitInfoProps } from 'utils/GitUtils'
import { useAppContext } from 'AppContext'
import css from './WehookForm.module.scss'

enum WebhookEventType {
  PUSH = 'push',
  ALL = 'all',
  INDIVIDUAL = 'individual'
}

enum WebhookIndividualEvent {
  BRANCH_PUSHED = 'branch_pushed',
  BRANCH_DELETED = 'branch_deleted'
}

interface FormData {
  url: string
  secret: string
  enabled: boolean
  secure: boolean
  events: WebhookEventType
  branchPush: boolean
  branchDeletion: boolean
}

interface WebHookFormProps extends Pick<GitInfoProps, 'repoMetadata'> {
  isEdit?: boolean
  webhook?: OpenapiWebhookType
}

export function WehookForm({ repoMetadata, isEdit, webhook }: WebHookFormProps) {
  const history = useHistory()
  const { getString } = useStrings()
  const { showError } = useToaster()
  const { routes } = useAppContext()
  const { mutate, loading } = useMutate<OpenapiWebhookType>({
    verb: isEdit ? 'PATCH' : 'POST',
    path: `/api/v1/repos/${repoMetadata.path}/+/webhooks${isEdit ? `/${webhook?.id}` : ''}`
  })

  return (
    <Container padding="xxlarge">
      <Layout.Vertical className={css.form}>
        <Formik<FormData>
          initialValues={{
            url: webhook?.url || '',
            secret: isEdit ? '***' : '',
            enabled: webhook ? (webhook?.enabled as boolean) : true,
            secure: webhook?.insecure === false || false,
            branchPush: webhook?.triggers?.includes(WebhookIndividualEvent.BRANCH_PUSHED) || false,
            branchDeletion: webhook?.triggers?.includes(WebhookIndividualEvent.BRANCH_DELETED) || false,
            events: WebhookEventType.INDIVIDUAL
          }}
          formName="create-webhook-form"
          enableReinitialize={true}
          validateOnChange
          validateOnBlur
          validationSchema={yup.object().shape({
            url: yup.string().required().url(),
            secret: yup.string().required()
          })}
          onSubmit={formData => {
            const triggers: OpenapiWebhookTrigger[] = []

            if (formData.branchPush) {
              triggers.push(WebhookIndividualEvent.BRANCH_PUSHED)
            }

            if (formData.branchDeletion) {
              triggers.push(WebhookIndividualEvent.BRANCH_DELETED)
            }

            if (!triggers.length) {
              return showError('At least one event must be selected')
            }

            const data: OpenapiUpdateWebhookRequest = {
              url: formData.url,
              secret: formData.secret,
              enabled: formData.enabled,
              insecure: !formData.secure,
              triggers
            }

            mutate(data)
              .then((response: OpenapiWebhookType) => {
                history.push(
                  routes.toCODEWebhookDetails({
                    repoPath: repoMetadata.path as string,
                    webhookId: String(response.id)
                  })
                )
              })
              .catch(exception => {
                showError(getErrorMessage(exception))
              })
          }}>
          {formik => {
            const { values } = formik

            return (
              <FormikForm>
                <FormInput.Text
                  name="url"
                  label={getString('payloadUrlLabel')}
                  placeholder={getString('samplePayloadUrl')}
                  tooltipProps={{ dataTooltipId: 'payloadUrl' }}
                  inputGroup={{ autoFocus: true }}
                />

                <FormInput.Text
                  name="secret"
                  label={getString('secret')}
                  placeholder={getString('enterSecret')}
                  tooltipProps={{ dataTooltipId: 'secret' }}
                  inputGroup={{ type: 'password' }}
                />

                <FormGroup className={css.eventRadioGroup}>
                  <FormInput.RadioGroup
                    name="events"
                    className={css.eventRadioGroup}
                    label={getString('webhookEventsLabel')}
                    items={[
                      { label: getString('pushEvent'), value: WebhookEventType.PUSH, disabled: true },
                      { label: getString('allEvents'), value: WebhookEventType.ALL, disabled: true },
                      { label: getString('individualEvents'), value: WebhookEventType.INDIVIDUAL }
                    ]}
                  />
                  {values.events === WebhookEventType.INDIVIDUAL ? (
                    <article
                      style={{ display: 'flex', gap: '6rem', flexWrap: 'wrap', marginLeft: '30px', marginTop: '10px' }}>
                      <section>
                        <FormInput.CheckBox label={'Branch push'} name="branchPush" className={css.checkbox} />
                        <FormInput.CheckBox label={'Branch deletion'} name="branchDeletion" className={css.checkbox} />
                      </section>
                    </article>
                  ) : null}
                </FormGroup>

                <FormGroup>
                  <div className={css.sslVerificationLabel}>
                    <Text
                      font={{ variation: FontVariation.FORM_LABEL, weight: 'bold' }}
                      padding={{ bottom: 10 }}
                      className="bp3-label">
                      {getString('sslVerificationLabel')}
                    </Text>
                    <FormInput.CheckBox label={getString('enableSSLVerification')} name="secure" />
                  </div>
                </FormGroup>

                <FormGroup>
                  <div className={css.sslVerificationLabel}>
                    <Text
                      font={{ variation: FontVariation.FORM_LABEL, weight: 'bold' }}
                      padding={{ bottom: 10 }}
                      className="bp3-label">
                      {getString('enabled')}
                    </Text>
                    <FormInput.CheckBox label={''} name="enabled" />
                  </div>
                </FormGroup>

                <Layout.Horizontal
                  spacing="small"
                  padding={{ right: 'xxlarge', top: 'xxxlarge', bottom: 'large' }}
                  style={{ alignItems: 'center' }}>
                  <Button
                    type="submit"
                    text={getString(isEdit ? 'updateWebhook' : 'createWebhook')}
                    variation={ButtonVariation.PRIMARY}
                    disabled={loading}
                  />

                  <Button
                    text={getString('cancel')}
                    variation={ButtonVariation.TERTIARY}
                    onClick={() =>
                      history.push(
                        routes.toCODEWebhooks({
                          repoPath: repoMetadata.path as string
                        })
                      )
                    }
                  />
                  <FlexExpander />
                </Layout.Horizontal>
              </FormikForm>
            )
          }}
        </Formik>
      </Layout.Vertical>
    </Container>
  )
}
