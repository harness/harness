import React from 'react'
import { Container, FormInput, Layout, Text } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import type { FormikProps } from 'formik'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import css from './InfraDetails.module.scss'

interface BasicDetailProps {
  formikProps: FormikProps<any>
}

const BasicDetails = ({ formikProps }: BasicDetailProps) => {
  const { getString } = useStrings()
  const { hooks, accountInfo, customComponents } = useAppContext()
  const { DelegateSelectorsV2 } = customComponents
  const queryParams = { accountId: accountInfo?.identifier }
  const { data: delegateData } = hooks.useGetDelegateSelectorsUpTheHierarchyV2({
    queryParams
  })

  const delegateHandler = (val: string[]) => {
    formikProps.setFieldValue('delegateSelector', val)
  }

  return (
    <Layout.Vertical spacing="medium" className={css.containerSpacing}>
      <Text className={css.basicDetailsHeading}>{getString('cde.configureInfra.basicDetails')}</Text>
      <Container className={css.basicDetailsBody}>
        <FormInput.InputWithIdentifier
          inputLabel={getString('cde.configureInfra.name')}
          inputName="name"
          isIdentifierEditable={true}
        />
        <FormInput.Text
          name="project"
          label={getString('cde.configureInfra.project')}
          placeholder={getString('cde.configureInfra.project')}
        />
        <FormInput.Text
          name="domain"
          className={css.inputWithNote}
          label={
            <Text
              rightIcon="info"
              className={css.inputLabel}
              rightIconProps={{ color: Color.PRIMARY_7, size: 12, margin: 5 }}>
              {getString('cde.configureInfra.domain')}
            </Text>
          }
          placeholder={getString('cde.configureInfra.domain')}
        />
        <Text className={css.noteText}>{getString('cde.configureInfra.basicNoteText')}</Text>
        <Container className={css.delegateContainer}>
          <Text className={css.delegateSelector}>{getString('cde.delegate.DelegateSelector')}</Text>
          <DelegateSelectorsV2
            data={delegateData?.resource ?? []}
            selectedItems={formikProps?.values?.delegateSelector}
            onTagInputChange={delegateHandler}
          />
        </Container>
      </Container>
    </Layout.Vertical>
  )
}

export default BasicDetails
