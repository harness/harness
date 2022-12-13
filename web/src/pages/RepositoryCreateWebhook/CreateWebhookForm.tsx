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
  FontVariation
} from '@harness/uicore'
import { FormGroup } from '@blueprintjs/core'
import React from 'react'
import { useStrings } from 'framework/strings'
import css from './RepositoryCreateWebhook.module.scss'

export default function CreateWehookForm() {
  const { getString } = useStrings()
  return (
    <Container margin={{ right: 'xxlarge' }} style={{ width: 700, maxHeight: '95vh', overflow: 'auto' }}>
      <Layout.Vertical padding={{ left: 'xxlarge', top: 'xxlarge' }} style={{ height: '100%' }} className={css.form}>
        <Formik
          initialValues={{
            payloadUrl: '',
            events: ''
          }}
          formName="create-webhook-form"
          enableReinitialize={true}
          validateOnChange
          validateOnBlur
          onSubmit={() => {
            console.log('here')
          }}>
          {formik => {
            const { values } = formik
            return (
              <FormikForm>
                <FormInput.Text
                  name="payloadUrl"
                  label={getString('payloadUrlLabel')}
                  placeholder={getString('samplePayloadUrl')}
                  tooltipProps={{
                    dataTooltipId: 'payloadUrl'
                  }}
                  inputGroup={{ autoFocus: true }}
                />

                <FormInput.Text
                  name="secret"
                  label={getString('secret')}
                  placeholder={getString('samplePayloadUrl')}
                  tooltipProps={{
                    dataTooltipId: 'secret'
                  }}
                />
                <FormGroup className={css.eventRadioGroup}>
                  <FormInput.RadioGroup
                    name="events"
                    className={css.eventRadioGroup}
                    label={getString('webhookEventsLabel')}
                    items={[
                      { label: getString('pushEvent'), value: 'singleEvent' },
                      { label: getString('allEvents'), value: 'all' },
                      { label: getString('individualEvents'), value: 'one' }
                    ]}
                  />
                  {values.events === 'one' ? (
                    <article
                      style={{ display: 'flex', gap: '6rem', flexWrap: 'wrap', marginLeft: '30px', marginTop: '10px' }}>
                      <section>
                        <FormInput.CheckBox
                          label={getString('branchTagCreation')}
                          name="branchTagCreation"
                          className={css.checkbox}
                        />
                        <FormInput.CheckBox
                          label={getString('branchProtectionRules')}
                          name="branchProtectionRules"
                          className={css.checkbox}
                        />
                        <FormInput.CheckBox
                          label={getString('checkSuites')}
                          name="checkSuites"
                          className={css.checkbox}
                        />
                        <FormInput.CheckBox label={getString('botAlerts')} name="botAlerts" className={css.checkbox} />
                      </section>
                      <section>
                        <FormInput.CheckBox
                          label={getString('branchTagDeletion')}
                          name="branchTagDeletion"
                          className={css.checkbox}
                        />

                        <FormInput.CheckBox label={getString('checkRuns')} name="checkRuns" className={css.checkbox} />
                        <FormInput.CheckBox
                          label={getString('scanAlerts')}
                          name="scanAlerts"
                          className={css.checkbox}
                        />
                        <FormInput.CheckBox
                          label={getString('deployKeys')}
                          name="deployKeys"
                          className={css.checkbox}
                        />
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

                    <FormInput.CheckBox label={getString('enableSSLVerification')} name="sslVerification" />
                  </div>
                </FormGroup>

                <Layout.Horizontal
                  spacing="small"
                  padding={{ right: 'xxlarge', top: 'xxxlarge', bottom: 'large' }}
                  style={{ alignItems: 'center' }}>
                  <Button type="submit" text={getString('createWebhook')} variation={ButtonVariation.PRIMARY} />

                  <Button text={getString('cancel')} variation={ButtonVariation.TERTIARY} />
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
