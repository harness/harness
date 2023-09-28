import React, { useCallback, useEffect, useState } from 'react'
import { debounce, omit } from 'lodash'
import { FormikContextType, connect } from 'formik'
import { Layout, Text, FormInput, Button, ButtonVariation, ButtonSize, Container } from '@harnessio/uicore'
import { FontVariation } from '@harnessio/design-system'
import { Icon } from '@harnessio/icons'
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
  const [values, setValues] = useState<string[]>([])

  useEffect(() => {
    if (values.length > 0) {
      formik?.setFieldValue(name, values)
    } else {
      cleanupField()
    }
  }, [values])

  const getFieldName = useCallback(
    (index: number): string => {
      return `${name}-${index}`
    },
    [name]
  )

  const handleAddRowToList = useCallback((): void => {
    setRowCount((existingCount: number) => existingCount + 1)
  }, [])

  const handleRemoveRowFromList = useCallback((removedRowIdx: number): void => {
    setRowCount((existingCount: number) => existingCount - 1)
    removeItemFromList(removedRowIdx)
  }, [])

  const handleAddItemToList = useCallback((event: React.FormEvent<HTMLInputElement>): void => {
    const value = (event.target as HTMLInputElement).value
    setValues((existingValues: string[]) => [...existingValues, value])
  }, [])

  const removeItemFromList = useCallback((removedRowIdx: number): void => {
    setValues((existingValues: string[]) => {
      const existingValuesCopy = [...existingValues]
      if (removedRowIdx > -1) {
        existingValuesCopy.splice(removedRowIdx, 1)
        return existingValuesCopy
      }
      return existingValues
    })
  }, [])

  const cleanupField = useCallback((): void => {
    formik?.setValues(omit({ ...formik?.values }, name))
  }, [formik?.values])

  const debouncedAddItemToList = useCallback(debounce(handleAddItemToList, 500), [handleAddItemToList])

  const renderRow = useCallback((rowIndex: number): React.ReactElement => {
    return (
      <Layout.Horizontal margin={{ bottom: 'none' }} flex={{ justifyContent: 'space-between', alignItems: 'center' }}>
        <Container width="90%">
          <FormInput.Text name={getFieldName(rowIndex)} disabled={readOnly} onChange={debouncedAddItemToList} />
        </Container>
        <Icon
          name="code-delete"
          size={25}
          padding={{ bottom: 'medium' }}
          className={css.deleteRowBtn}
          onClick={event => {
            event.preventDefault()
            handleRemoveRowFromList(rowIndex)
          }}
        />
      </Layout.Horizontal>
    )
  }, [])

  const renderRows = useCallback((numOfRows: number): React.ReactElement => {
    const rows: React.ReactElement[] = []
    for (let idx = 0; idx < numOfRows; idx++) {
      rows.push(renderRow(idx))
    }
    return <Layout.Vertical>{rows}</Layout.Vertical>
  }, [])

  return (
    <Layout.Vertical spacing="small">
      <Layout.Vertical>
        <Text font={{ variation: FontVariation.FORM_LABEL }}>{label}</Text>
        {rowCount > 0 && <Container padding={{ top: 'small' }}>{renderRows(rowCount)}</Container>}
      </Layout.Vertical>
      <Button
        text={getString('addLabel')}
        rightIcon="plus"
        variation={ButtonVariation.PRIMARY}
        size={ButtonSize.SMALL}
        className={css.addBtn}
        onClick={handleAddRowToList}
      />
    </Layout.Vertical>
  )
}

export default connect(MultiList)
