import React, { useCallback, useEffect, useState } from 'react'
import { get } from 'lodash'
import { FormikContextType, connect } from 'formik'
import { Layout, Text, FormInput, Button, ButtonVariation, ButtonSize } from '@harnessio/uicore'
import { FontVariation } from '@harnessio/design-system'
import { useStrings } from 'framework/strings'

import css from './MultiList.module.scss'

interface MultiListConnectedProps extends MultiListProps {
  formik?: FormikContextType<any>
}

interface MultiListProps {
  name: string
  label: string
  readOnly?: boolean
}

export const MultiList = ({ name, label, readOnly, formik }: MultiListConnectedProps): JSX.Element => {
  const { getString } = useStrings()
  const [rowCount, setRowCount] = useState<number>(0)

  useEffect(() => {
    formik?.setFieldValue(name, getListValues())
  }, [rowCount])

  const getListValues = useCallback((): string[] => {
    const existingValues = []
    for (let fieldIdx = 0; fieldIdx < rowCount; fieldIdx++) {
      const fieldName = getFieldName(fieldIdx)
      const fieldValue = get(formik?.values, fieldName)
      if (fieldValue) {
        existingValues.push(get(formik?.values, getFieldName(fieldIdx)))
      }
    }
    return existingValues
  }, [rowCount])

  const getFieldName = useCallback(
    (index: number): string => {
      return `${name}-${index}`
    },
    [name]
  )

  const handleAdd = (): void => {
    setRowCount((existingCount: number) => existingCount + 1)
  }

  const renderRow = useCallback((rowIndex: number): React.ReactElement => {
    return <FormInput.Text name={getFieldName(rowIndex)} disabled={readOnly} />
  }, [])

  const renderRows = useCallback((numOfRows: number): React.ReactElement => {
    const rows: React.ReactElement[] = []
    for (let idx = 0; idx < numOfRows; idx++) {
      rows.push(renderRow(idx))
    }
    return <Layout.Vertical spacing="small">{rows}</Layout.Vertical>
  }, [])

  return (
    <Layout.Vertical spacing="xsmall">
      <Text font={{ variation: FontVariation.FORM_LABEL }}>{label}</Text>
      {renderRows(rowCount)}
      <Button
        text={getString('addLabel')}
        rightIcon="plus"
        variation={ButtonVariation.PRIMARY}
        size={ButtonSize.SMALL}
        className={css.addBtn}
        onClick={handleAdd}
      />
    </Layout.Vertical>
  )
}

export default connect(MultiList)
