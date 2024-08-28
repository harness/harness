/*
 * Copyright 2023 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React, { useState } from 'react'
import {
  Button,
  ButtonVariation,
  Dialog,
  Layout,
  Text,
  Container,
  useToaster,
  Formik,
  FormInput,
  Popover,
  ButtonSize,
  FlexExpander,
  FormikForm,
  stringSubstitute
} from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { Color, FontVariation } from '@harnessio/design-system'
import { Menu, MenuItem, PopoverInteractionKind, Position, Spinner } from '@blueprintjs/core'
import * as Yup from 'yup'
import { FieldArray } from 'formik'
import { useGet, useMutate } from 'restful-react'
import { Render } from 'react-jsx-match'
import { useModalHook } from 'hooks/useModalHook'
import { useStrings } from 'framework/strings'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { CodeIcon } from 'utils/GitUtils'
import { colorsPanel, ColorName, ColorDetails, getErrorMessage, LabelType, LabelTypes, getScopeData } from 'utils/Utils'
import { Label } from 'components/Label/Label'
import type {
  EnumLabelColor,
  TypesLabel,
  TypesLabelValue,
  TypesSaveLabelInput,
  TypesSaveLabelValueInput
} from 'services/code'
import { getConfig } from 'services/config'
import { useAppContext } from 'AppContext'
import css from './LabelModal.module.scss'

const enum ModalMode {
  SAVE,
  UPDATE
}
interface ExtendedTypesLabelValue extends TypesLabelValue {
  color: ColorName
}

interface LabelModalProps {
  refetchlabelsList: () => Promise<void>
}

interface LabelFormData extends TypesLabel {
  labelName: string
  allowDynamicValues: boolean
  color: ColorName
  labelValues: ExtendedTypesLabelValue[]
}

const ColorSelectorDropdown = (props: {
  onClick: (color: ColorName) => void
  currentColorName: ColorName | undefined | false
  disabled?: boolean
}) => {
  const { currentColorName, onClick: onClickColorOption, disabled: disabledPopover } = props

  const colorNames: ColorName[] = Object.keys(colorsPanel) as ColorName[]
  const getColorsObj = (colorKey: ColorName): ColorDetails => {
    return colorsPanel[colorKey]
  }

  const currentColorObj = getColorsObj(currentColorName ? currentColorName : ColorName.Blue)

  return (
    <Popover
      minimal
      interactionKind={PopoverInteractionKind.CLICK}
      position={Position.BOTTOM}
      disabled={disabledPopover}
      popoverClassName={css.popover}
      content={
        <Menu style={{ margin: '1px' }} className={css.colorMenu}>
          {colorNames?.map(colorName => {
            const colorObj = getColorsObj(colorName)
            return (
              <MenuItem
                key={colorName}
                active={colorName === currentColorName}
                text={
                  <Text font={{ size: 'normal' }} style={{ color: `${colorObj.text}` }}>
                    {colorName}
                  </Text>
                }
                onClick={() => onClickColorOption(colorName)}
              />
            )
          })}
        </Menu>
      }>
      <Button
        className={css.selectColor}
        text={
          <Layout.Horizontal width={'97px'} flex={{ alignItems: 'center', justifyContent: 'space-between' }}>
            <Text
              font={{ size: 'medium' }}
              icon={'symbol-circle'}
              iconProps={{ size: 20 }}
              padding={{ right: 'xsmall' }}
              style={{
                color: `${currentColorObj.stroke}`
              }}
            />

            <Text
              font={{ size: 'normal' }}
              style={{
                color: `${currentColorObj.text}`,
                gap: '5px',
                alignItems: 'center'
              }}>
              {currentColorName}
            </Text>

            <FlexExpander />
            <Icon
              padding={{ right: 'small', top: '2px' }}
              name="chevron-down"
              font={{ size: 'normal' }}
              size={15}
              background={currentColorObj.text}
            />
          </Layout.Horizontal>
        }
      />
    </Popover>
  )
}

const useLabelModal = ({ refetchlabelsList }: LabelModalProps) => {
  const { repoMetadata, space } = useGetRepositoryMetadata()
  const { standalone } = useAppContext()
  const { getString } = useStrings()
  const { showSuccess, showError } = useToaster()
  const [modalMode, setModalMode] = useState<ModalMode>(ModalMode.SAVE)
  const [updateLabel, setUpdateLabel] = useState<LabelTypes>()

  const openUpdateLabelModal = (label: LabelTypes) => {
    setModalMode(ModalMode.UPDATE)
    setUpdateLabel(label)
    openModal()
  }

  const { scopeRef } = getScopeData(space, updateLabel?.scope ?? 1, standalone)

  const { mutate: createUpdateLabel } = useMutate({
    verb: 'PUT',
    base: getConfig('code/api/v1'),
    path: `/repos/${repoMetadata?.path as string}/+/labels`
  })

  const { mutate: createUpdateSpaceLabel } = useMutate({
    verb: 'PUT',
    base: getConfig('code/api/v1'),
    path: updateLabel?.scope ? `/spaces/${scopeRef as string}/+/labels` : `/spaces/${space as string}/+/labels`
  })

  const getPath = () =>
    updateLabel?.scope === 0 && repoMetadata
      ? `/repos/${encodeURIComponent(repoMetadata?.path as string)}/labels/${encodeURIComponent(
          updateLabel?.key ? updateLabel?.key : ''
        )}/values`
      : `/spaces/${encodeURIComponent(scopeRef)}/labels/${encodeURIComponent(
          updateLabel?.key ? updateLabel?.key : ''
        )}/values`

  const {
    data: initLabelValues,
    loading: initValueListLoading,
    refetch: refetchInitValuesList
  } = useGet<TypesLabelValue[]>({
    base: getConfig('code/api/v1'),
    path: getPath(),
    lazy: true
  })

  //ToDo : Remove getLabelValuesPromiseQuery component when Encoding is handled by BE for Harness

  const getPathHarness = () =>
    updateLabel?.scope === 0 && repoMetadata
      ? `/repos/${repoMetadata?.identifier}/labels/${encodeURIComponent(
          updateLabel?.key ? updateLabel?.key : ''
        )}/values`
      : `/labels/${encodeURIComponent(updateLabel?.key ? updateLabel?.key : '')}/values`

  const {
    data: initLabelValuesQuery,
    loading: initValueListLoadingQuery,
    refetch: refetchInitValuesListQuery
  } = useGet<TypesLabelValue[]>({
    base: getConfig('code/api/v1'),
    path: getPathHarness(),
    queryParams: {
      accountIdentifier: scopeRef?.split('/')[0],
      orgIdentifier: scopeRef?.split('/')[1],
      projectIdentifier: scopeRef?.split('/')[2]
    },
    lazy: true
  })

  //ToDo: Remove all references of suffix with Query when Encoding is handled by BE for Harness

  const [openModal, hideModal] = useModalHook(() => {
    const handleLabelSubmit = (formData: LabelFormData) => {
      const { labelName, color, labelValues, description, allowDynamicValues, id } = formData
      const createLabelPayload: { label: TypesSaveLabelInput; values: TypesSaveLabelValueInput[] } = {
        label: {
          color: color?.toLowerCase() as EnumLabelColor,
          description: description,
          id: id ?? 0,
          key: labelName,
          type: allowDynamicValues ? LabelType.DYNAMIC : LabelType.STATIC
        },
        values: labelValues?.length
          ? labelValues.map(value => {
              return {
                color: value.color?.toLowerCase() as EnumLabelColor,
                id: value.id ?? 0,
                value: value.value
              }
            })
          : []
      }
      if (
        (repoMetadata && modalMode === ModalMode.SAVE) ||
        (modalMode === ModalMode.UPDATE && updateLabel?.scope === 0 && repoMetadata)
      ) {
        try {
          createUpdateLabel(createLabelPayload)
            .then(() => {
              showSuccess(
                modalMode === ModalMode.SAVE ? getString('labels.labelCreated') : getString('labels.labelUpdated')
              )
              refetchlabelsList()
              onClose()
            })
            .catch(error => showError(getErrorMessage(error), 1200, getString('labels.labelCreationFailed')))
        } catch (exception) {
          showError(getErrorMessage(exception), 1200, getString('labels.labelCreationFailed'))
        }
      } else {
        try {
          createUpdateSpaceLabel(createLabelPayload)
            .then(() => {
              showSuccess(
                modalMode === ModalMode.SAVE ? getString('labels.labelCreated') : getString('labels.labelUpdated')
              )
              refetchlabelsList()
              onClose()
            })
            .catch(error => showError(getErrorMessage(error), 1200, getString('labels.labelUpdateFailed')))
        } catch (exception) {
          showError(getErrorMessage(exception), 1200, getString('labels.labelUpdateFailed'))
        }
      }
    }
    const onClose = () => {
      setModalMode(ModalMode.SAVE)
      setUpdateLabel({})
      hideModal()
    }

    const validationSchema = Yup.object({
      labelName: Yup.string()
        .max(
          50,
          stringSubstitute(getString('labels.stringMax'), {
            entity: 'Name'
          }) as string
        )
        .test(
          'no-newlines',
          stringSubstitute(getString('labels.noNewLine'), {
            entity: 'Name'
          }) as string,
          value => !/\r|\n/.test(value as string)
        )
        .required(getString('labels.labelNameReq')),
      labelValues: Yup.array().of(
        Yup.object({
          value: Yup.string()
            .max(
              50,
              stringSubstitute(getString('labels.stringMax'), {
                entity: 'Value'
              }) as string
            )
            .test(
              'no-newlines',
              stringSubstitute(getString('labels.noNewLine'), {
                entity: 'Name'
              }) as string,
              value => !/\r|\n/.test(value as string)
            )
            .required(getString('labels.labelValueReq')),
          color: Yup.string()
        })
      )
    })
    const handleKeyDown = (event: React.KeyboardEvent<HTMLFormElement>) => {
      if (event.key === 'Enter') {
        event.preventDefault()
      }
    }

    const getLabelValues = (values: TypesLabelValue[] = []) =>
      values.map(valueObj => ({
        id: valueObj.id,
        value: valueObj.value,
        color: valueObj.color as ColorName
      }))

    const initialFormValues = (() => {
      const baseValues = {
        color: updateLabel?.color as ColorName,
        description: updateLabel?.description ?? '',
        id: updateLabel?.id ?? 0,
        labelName: updateLabel?.key ?? '',
        allowDynamicValues: updateLabel?.type === LabelType.DYNAMIC,
        labelValues: [] as { id: number; value: string; color: ColorName }[]
      }

      if (modalMode === ModalMode.SAVE) {
        return { ...baseValues, color: ColorName.Blue }
      }

      if (modalMode === ModalMode.UPDATE && updateLabel?.value_count === 0) return baseValues

      if (standalone) {
        return { ...baseValues, labelValues: getLabelValues(initLabelValues ?? undefined) }
      }
      return { ...baseValues, labelValues: getLabelValues(initLabelValuesQuery ?? undefined) }
    })()

    return (
      <Dialog
        isOpen
        onOpening={() => {
          if (modalMode === ModalMode.UPDATE && updateLabel?.value_count !== 0) {
            standalone ? refetchInitValuesList() : refetchInitValuesListQuery()
          }
        }}
        enforceFocus={false}
        onClose={onClose}
        title={modalMode === ModalMode.SAVE ? getString('labels.createLabel') : getString('labels.updateLabel')}
        className={css.labelModal}>
        <Formik<LabelFormData>
          formName="labelModal"
          initialValues={initialFormValues}
          enableReinitialize={true}
          validationSchema={validationSchema}
          validateOnChange
          validateOnBlur
          onSubmit={handleLabelSubmit}>
          {formik => {
            return (
              <FormikForm onKeyDown={handleKeyDown}>
                <Render when={modalMode === ModalMode.UPDATE}>
                  <Container className={css.yellowContainer}>
                    <Text
                      icon="main-issue"
                      iconProps={{ size: 16, color: Color.ORANGE_700, margin: { right: 'small' } }}
                      padding={{ left: 'large', right: 'large', top: 'small', bottom: 'small' }}
                      color={Color.WARNING}>
                      {getString('labels.intentText', {
                        space: updateLabel?.key
                      })}
                    </Text>
                  </Container>
                </Render>
                <Layout.Horizontal spacing={'large'}>
                  <Layout.Vertical style={{ width: '55%' }}>
                    <Layout.Vertical spacing="large" className={css.modalForm}>
                      <Container margin={{ top: 'medium' }}>
                        <Text font={{ variation: FontVariation.BODY2 }}>{getString('labels.labelName')}</Text>
                        <Layout.Horizontal
                          flex={{ alignItems: formik.isValid ? 'center' : 'flex-start', justifyContent: 'flex-start' }}
                          style={{ gap: '4px', margin: '4px' }}>
                          <ColorSelectorDropdown
                            currentColorName={formik.values.color || ColorName.Blue}
                            onClick={(colorName: ColorName) => {
                              formik.setFieldValue('color', colorName)
                            }}
                          />
                          <FormInput.Text
                            key={'labelName'}
                            style={{ flexGrow: '1', margin: 0 }}
                            name="labelName"
                            placeholder={getString('labels.provideLabelName')}
                            tooltipProps={{
                              dataTooltipId: 'labels.newLabel'
                            }}
                            inputGroup={{ autoFocus: true }}
                          />
                        </Layout.Horizontal>
                      </Container>
                      <Container margin={{ top: 'medium' }} className={css.labelDescription}>
                        <Text font={{ variation: FontVariation.BODY2 }}>{getString('labels.descriptionOptional')}</Text>
                        <FormInput.Text name="description" placeholder={getString('labels.placeholderDescription')} />
                      </Container>
                      <Container margin={{ top: 'medium' }}>
                        <Text font={{ variation: FontVariation.BODY2 }}>{getString('labels.labelValuesOptional')}</Text>
                        <FieldArray
                          name="labelValues"
                          render={({ push, remove }) => {
                            if (
                              modalMode === ModalMode.UPDATE &&
                              updateLabel?.value_count !== 0 &&
                              (initValueListLoading || initValueListLoadingQuery)
                            )
                              return <Spinner size={20} />
                            else
                              return (
                                <Layout.Vertical>
                                  {formik.values.labelValues?.map((_, index) => (
                                    <Layout.Horizontal
                                      key={`labelValue + ${index}`}
                                      flex={{
                                        alignItems: formik.isValid ? 'center' : 'flex-start',
                                        justifyContent: 'flex-start'
                                      }}
                                      style={{ gap: '4px', margin: '4px' }}>
                                      <ColorSelectorDropdown
                                        key={`labelValueColor + ${index}`}
                                        currentColorName={
                                          formik.values.labelValues &&
                                          index !== undefined &&
                                          (formik.values.labelValues[index].color as ColorName)
                                        }
                                        onClick={(colorName: ColorName) => {
                                          formik.setFieldValue(
                                            'labelValues',
                                            formik.values.labelValues?.map((value, i) =>
                                              i === index ? { ...value, color: colorName } : value
                                            )
                                          )
                                        }}
                                      />
                                      <FormInput.Text
                                        key={`labelValueKey + ${index}`}
                                        style={{ flexGrow: '1', margin: 0 }}
                                        name={`${'labelValues'}[${index}].value`}
                                        placeholder={getString('labels.provideLabelValue')}
                                        tooltipProps={{
                                          dataTooltipId: 'labels.newLabel'
                                        }}
                                        inputGroup={{ autoFocus: true }}
                                      />
                                      <Button
                                        key={`removeValue + ${index}`}
                                        style={{ marginRight: 'auto', color: 'var(--grey-300)' }}
                                        variation={ButtonVariation.ICON}
                                        icon={'code-close'}
                                        onClick={() => {
                                          remove(index)
                                        }}
                                      />
                                    </Layout.Horizontal>
                                  ))}
                                  <Button
                                    style={{ marginRight: 'auto' }}
                                    variation={ButtonVariation.LINK}
                                    disabled={!formik.isValid || formik.values.labelName?.length === 0}
                                    text={getString('labels.addValue')}
                                    icon={CodeIcon.Add}
                                    onClick={() =>
                                      push({
                                        name: '',
                                        color: formik.values.color
                                      })
                                    }
                                  />
                                </Layout.Vertical>
                              )
                          }}
                        />
                      </Container>
                      <Container margin={{ top: 'medium' }} className={css.labelDescription}>
                        <FormInput.CheckBox label={getString('labels.allowDynamic')} name="allowDynamicValues" />
                      </Container>
                    </Layout.Vertical>
                    <Container margin={{ top: 'medium' }}>
                      <Layout.Horizontal flex={{ justifyContent: 'flex-start' }}>
                        <Button
                          margin={{ right: 'medium' }}
                          type="submit"
                          text={getString('save')}
                          variation={ButtonVariation.PRIMARY}
                          size={ButtonSize.MEDIUM}
                        />
                        <Button
                          text={getString('cancel')}
                          variation={ButtonVariation.TERTIARY}
                          size={ButtonSize.MEDIUM}
                          onClick={onClose}
                        />
                      </Layout.Horizontal>
                    </Container>
                  </Layout.Vertical>
                  <Layout.Vertical
                    style={{ width: '45%', padding: '25px 35px 25px 35px', borderLeft: '1px solid var(--grey-100)' }}>
                    <Text>{getString('labels.labelPreview')}</Text>
                    {modalMode === ModalMode.UPDATE &&
                    updateLabel?.value_count !== 0 &&
                    (initValueListLoading || initValueListLoadingQuery) ? (
                      <Spinner size={20} />
                    ) : (
                      <Layout.Vertical spacing={'medium'}>
                        {formik.values.labelValues?.length
                          ? formik.values.labelValues?.map((valueObj, i) => (
                              <Label
                                key={`label + ${i}`}
                                name={formik.values.labelName || getString('labels.labelName')}
                                label_color={formik.values.color}
                                label_value={
                                  valueObj.value?.length
                                    ? { name: valueObj.value, color: valueObj.color }
                                    : {
                                        name: getString('labels.labelValue'),
                                        color: valueObj.color || formik.values.color
                                      }
                                }
                              />
                            ))
                          : !formik.values.allowDynamicValues && (
                              <Label
                                name={formik.values.labelName || getString('labels.labelName')}
                                label_color={formik.values.color}
                              />
                            )}
                        {formik.values.allowDynamicValues && (
                          <Label
                            name={formik.values.labelName || getString('labels.labelName')}
                            label_color={formik.values.color}
                            label_value={{ name: getString('labels.canbeAddedByUsers') }}
                          />
                        )}
                      </Layout.Vertical>
                    )}
                  </Layout.Vertical>
                </Layout.Horizontal>
              </FormikForm>
            )
          }}
        </Formik>
      </Dialog>
    )
  }, [
    updateLabel,
    initLabelValues,
    initLabelValuesQuery,
    initValueListLoading,
    initValueListLoadingQuery,
    refetchlabelsList,
    refetchInitValuesListQuery,
    modalMode
  ])

  return {
    openModal,
    openUpdateLabelModal,
    hideModal
  }
}

export default useLabelModal
