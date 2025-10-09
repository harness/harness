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

import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Formik, FormikContextType } from 'formik'
import { parse } from 'yaml'
import cx from 'classnames'
import { capitalize, get, has, isEmpty, isUndefined, pick, set } from 'lodash-es'
import type { IRange } from 'monaco-editor'
import { Classes, PopoverInteractionKind, PopoverPosition } from '@blueprintjs/core'
import { Color, FontVariation } from '@harnessio/design-system'
import { Icon, IconName, IconProps } from '@harnessio/icons'
import {
  Button,
  ButtonSize,
  ButtonVariation,
  Card,
  Container,
  ExpandingSearchInput,
  FormInput,
  FormikForm,
  Layout,
  Popover,
  Text
} from '@harnessio/uicore'
import type { TypesPlugin } from 'services/code'
import { useStrings } from 'framework/strings'
import { MultiList } from 'components/MultiList/MultiList'
import MultiMap from 'components/MultiMap/MultiMap'
import { PipelineEntity, Action, CodeLensClickMetaData } from 'components/PipelineConfigPanel/types'
import { generateDefaultStepInsertionPath } from 'components/SourceCodeEditor/EditorUtils'
import { usePublicResourceConfig } from 'hooks/usePublicResourceConfig'
import { RunStep } from './Steps/HarnessSteps/RunStep/RunStep'
import css from './PluginsPanel.module.scss'

export interface EntityAddUpdateInterface extends Partial<CodeLensClickMetaData> {
  pathToField: string[]
  range?: IRange
  isUpdate: boolean
  formData: PluginFormDataInterface
}

export interface PluginsPanelInterface {
  pluginDataFromYAML: EntityAddUpdateInterface
  onPluginAddUpdate: (data: EntityAddUpdateInterface) => void
  pluginFieldUpdateData: Partial<EntityAddUpdateInterface>
}

export interface PluginFormDataInterface {
  [key: string]: string | boolean | object
}

enum ValueType {
  STRING = 'string',
  BOOLEAN = 'boolean',
  NUMBER = 'number',
  ARRAY = 'array',
  OBJECT = 'object'
}

interface PluginInput {
  type: ValueType
  description?: string
  default?: string
  options?: { isExtended?: boolean }
}

interface PluginInputs {
  [key: string]: PluginInput
}

interface PluginCategoryInterface {
  category: PluginCategory
  name: string
  description: string
  icon: IconProps
}

enum PluginCategory {
  Harness = 'run',
  Drone = 'plugin'
}

enum PluginPanelView {
  Category,
  Listing,
  Configuration
}

const PluginsInputPath = 'inputs'
const PluginSpecPath = 'spec'
const PluginSpecInputPath = `${PluginSpecPath}.${PluginsInputPath}`

const LIST_FETCHING_LIMIT = 100

const RunStepSpec: TypesPlugin = {
  identifier: 'run'
}

export const PluginsPanel = (props: PluginsPanelInterface): JSX.Element => {
  const { pluginDataFromYAML, onPluginAddUpdate, pluginFieldUpdateData } = props
  const { getString } = useStrings()
  const [pluginCategory, setPluginCategory] = useState<PluginCategory>()
  const [panelView, setPanelView] = useState<PluginPanelView>(PluginPanelView.Category)
  const [plugin, setPlugin] = useState<TypesPlugin>()
  const [plugins, setPlugins] = useState<TypesPlugin[]>([])
  const [query, setQuery] = useState<string>('')
  const [loading, setLoading] = useState<boolean>(false)
  const formikRef = useRef<FormikContextType<PluginFormDataInterface>>()
  const [showSyncToolbar, setShowSyncToolbar] = useState<boolean>(false)
  const [formInitialValues, setFormInitialValues] = useState<PluginFormDataInterface>()
  const { UIFlags } = usePublicResourceConfig()

  const PluginCategories: PluginCategoryInterface[] = useMemo(
    () => [
      {
        category: PluginCategory.Harness,
        name: capitalize(getString('run')),
        description: getString('pluginsPanel.run.helptext'),
        icon: { name: 'run-step', size: 15 }
      },
      ...(UIFlags.show_plugin
        ? [
            {
              category: PluginCategory.Drone,
              name: capitalize(getString('plugins.title')),
              description: getString('pluginsPanel.plugins.helptext'),
              icon: { name: 'plugin-ci-step' as IconName, size: 18 }
            }
          ]
        : [])
    ],
    [UIFlags]
  )

  useEffect(() => {
    const { entity, action } = pluginDataFromYAML
    if (entity === PipelineEntity.STEP) {
      switch (action) {
        case Action.EDIT:
          handleIncomingPluginData(pluginDataFromYAML)
          break
        case Action.ADD:
          setPanelView(PluginPanelView.Category)
          break
      }
    }
  }, [pluginDataFromYAML]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    const { isUpdate, formData, pathToField } = pluginDataFromYAML
    setFormInitialValues(
      isUpdate
        ? getInitialFormValuesFromYAML({
            pathToField,
            formData
          })
        : getInitialFormValuesWithFieldDefaults(
            getPluginInputsFromSpec(get(plugin, PluginSpecPath, '') as string) as PluginInputs
          )
    )
  }, [plugin, pluginDataFromYAML]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    setShowSyncToolbar(!isEmpty(pluginFieldUpdateData.pathToField)) // check with actual formik value as well
  }, [pluginFieldUpdateData])

  useEffect(() => {
    if (pluginCategory === PluginCategory.Drone) {
      fetchAllPlugins().then(response => setPlugins(response))
    }
  }, [pluginCategory]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (panelView !== PluginPanelView.Listing) return

    if (query) {
      setPlugins(existingPlugins => existingPlugins.filter((item: TypesPlugin) => item.identifier?.includes(query)))
    } else {
      fetchAllPlugins().then(response => setPlugins(response))
    }
  }, [panelView, query]) // eslint-disable-line react-hooks/exhaustive-deps

  const fetchPlugins = async (page: number): Promise<TypesPlugin[]> => {
    const response = await fetch(`/api/v1/plugins?page=${page}&limit=${LIST_FETCHING_LIMIT}`)
    if (!response.ok) throw new Error('Failed to fetch plugins')
    return response.json()
  }

  const fetchAllPlugins = useCallback(async (): Promise<TypesPlugin[]> => {
    try {
      setLoading(true)
      const pluginsPage1 = await fetchPlugins(1)
      const pluginsPage2 = await fetchPlugins(2)
      return [...pluginsPage1, ...pluginsPage2]
    } catch (ex) {
      /* ignore exception */
    } finally {
      setLoading(false)
    }
    return []
  }, [])

  const handleIncomingPluginData = useCallback(
    (data: EntityAddUpdateInterface) => {
      const { formData } = data
      const _category = get(formData, 'type') as PluginCategory
      if (_category === PluginCategory.Harness) {
        handlePluginCategoryClick(PluginCategory.Harness)
      } else {
        setPluginCategory(PluginCategory.Drone)
        fetchAllPlugins().then(response => {
          const matchingPlugin = response?.find(
            (_plugin: TypesPlugin) => _plugin?.identifier === get(formData, 'spec.name')
          )
          if (matchingPlugin) {
            setPlugin(matchingPlugin)
            setPanelView(PluginPanelView.Configuration)
          }
        })
      }
    },
    [fetchAllPlugins] // eslint-disable-line react-hooks/exhaustive-deps
  )

  const handlePluginCategoryClick = useCallback((selectedCategory: PluginCategory) => {
    setPluginCategory(selectedCategory)
    if (selectedCategory === PluginCategory.Drone) {
      setPanelView(PluginPanelView.Listing)
    } else if (selectedCategory === PluginCategory.Harness) {
      setPlugin(RunStepSpec)
      setPanelView(PluginPanelView.Configuration)
    }
  }, [])

  const renderPluginCategories = useCallback((): JSX.Element => {
    return (
      <Layout.Vertical spacing="large">
        <Text font={{ variation: FontVariation.H4 }}>{getString('stepCategory.select')}</Text>
        <Layout.Vertical>
          {PluginCategories.map((item: PluginCategoryInterface) => {
            const { name, category, description, icon } = item
            return (
              <Card
                className={cx(css.pluginCategoryCard, css.cursor)}
                key={category}
                onClick={() => handlePluginCategoryClick(category)}>
                <Layout.Horizontal flex={{ justifyContent: 'space-between' }}>
                  <Layout.Horizontal
                    onClick={() => handlePluginCategoryClick(category)}
                    flex={{ justifyContent: 'flex-start' }}
                    className={css.cursor}>
                    <Container className={css.pluginIcon}>
                      <Icon {...icon} />
                    </Container>
                    <Layout.Vertical padding={{ left: 'medium' }} spacing="xsmall">
                      <Text
                        color={Color.GREY_900}
                        className={css.fontWeight600}
                        font={{ variation: FontVariation.BODY2_SEMI }}>
                        {name}
                      </Text>
                      <Text color={Color.GREY_500} font={{ variation: FontVariation.SMALL }}>
                        {description}
                      </Text>
                    </Layout.Vertical>
                  </Layout.Horizontal>
                  <Container>
                    <Icon
                      name="arrow-right"
                      size={24}
                      onClick={() => handlePluginCategoryClick(category)}
                      className={css.cursor}
                    />
                  </Container>
                </Layout.Horizontal>
              </Card>
            )
          })}
        </Layout.Vertical>
      </Layout.Vertical>
    )
  }, [PluginCategories]) // eslint-disable-line react-hooks/exhaustive-deps

  const renderPlugins = useCallback((): JSX.Element => {
    return loading ? (
      <Container flex={{ justifyContent: 'center' }} padding="large">
        <Icon name="steps-spinner" color={Color.PRIMARY_7} size={25} />
      </Container>
    ) : (
      <Layout.Vertical spacing="large">
        <Layout.Horizontal flex={{ justifyContent: 'space-between' }}>
          <Layout.Horizontal flex={{ justifyContent: 'flex-start', alignItems: 'center' }} spacing="small">
            <Icon
              name="arrow-left"
              size={18}
              onClick={() => {
                setPanelView(PluginPanelView.Category)
              }}
              className={css.arrow}
            />
            <Text font={{ variation: FontVariation.H4 }}>{getString('plugins.select')}</Text>
          </Layout.Horizontal>
          <ExpandingSearchInput
            autoFocus={true}
            alwaysExpanded={true}
            defaultValue={query}
            onChange={setQuery}
            className={css.search}
          />
        </Layout.Horizontal>
        <Container className={css.plugins}>
          {plugins?.map((pluginItem: TypesPlugin) => {
            const { identifier, description } = pluginItem
            return (
              <Layout.Horizontal
                flex={{ justifyContent: 'flex-start' }}
                margin={{
                  bottom: 'large'
                }}
                className={css.cursor}
                onClick={() => {
                  setPanelView(PluginPanelView.Configuration)
                  setPlugin(pluginItem)
                }}
                key={identifier}
                width="100%">
                <Icon name={'plugin-ci-step'} size={25} />
                <Layout.Vertical padding={{ left: 'small' }} spacing="xsmall" className={css.pluginInfo}>
                  <Text
                    color={Color.GREY_900}
                    className={css.fontWeight600}
                    font={{ variation: FontVariation.BODY2_SEMI }}>
                    {identifier}
                  </Text>
                  <Text color={Color.GREY_500} font={{ variation: FontVariation.SMALL }} className={css.pluginDesc}>
                    {description}
                  </Text>
                </Layout.Vertical>
              </Layout.Horizontal>
            )
          })}
        </Container>
      </Layout.Vertical>
    )
  }, [loading, plugins, query]) // eslint-disable-line react-hooks/exhaustive-deps

  const generateFriendlyName = (pluginName: string): string => {
    return capitalize(pluginName.split('_').join(' '))
  }

  const generateLabelForPluginField = useCallback(
    ({ label, properties }: { label: string; properties: PluginInput }): JSX.Element | string => {
      const { description } = properties
      return (
        <Layout.Horizontal spacing="small" flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
          {label && <Text font={{ variation: FontVariation.FORM_LABEL }}>{generateFriendlyName(label)}</Text>}
          {description && (
            <Popover
              interactionKind={PopoverInteractionKind.HOVER}
              boundary="viewport"
              position={PopoverPosition.RIGHT}
              popoverClassName={Classes.DARK}
              content={
                <Container padding="medium">
                  <Text font={{ variation: FontVariation.SMALL }} color={Color.WHITE}>
                    {description}
                  </Text>
                </Container>
              }>
              <Icon name="info" color={Color.PRIMARY_7} size={10} padding={{ bottom: 'small' }} />
            </Popover>
          )}
        </Layout.Horizontal>
      )
    },
    [] // eslint-disable-line react-hooks/exhaustive-deps
  )

  const renderPluginFormField = useCallback(
    ({
      label,
      identifier,
      name,
      properties
    }: {
      label: string
      identifier: string
      name: string
      properties: PluginInput
    }): JSX.Element => {
      const { type, options } = properties

      switch (type) {
        case ValueType.STRING: {
          const { isExtended } = options || {}
          const WrapperComponent = isExtended ? FormInput.TextArea : FormInput.Text
          return (
            <WrapperComponent
              name={name}
              key={name}
              label={generateLabelForPluginField({ label, properties })}
              style={{ width: '100%' }}
            />
          )
        }
        case ValueType.BOOLEAN:
          return (
            <Container className={css.toggle}>
              <FormInput.Toggle
                name={name}
                key={name}
                label={generateLabelForPluginField({ label, properties }) as string}
                style={{ width: '100%' }}
              />
            </Container>
          )
        case ValueType.ARRAY:
          return (
            <Container margin={{ bottom: 'large' }}>
              <MultiList
                identifier={identifier}
                name={name}
                key={name}
                label={generateLabelForPluginField({ label, properties }) as string}
                formik={formikRef.current}
              />
            </Container>
          )
        case ValueType.OBJECT:
          return (
            <Container margin={{ bottom: 'large' }}>
              <MultiMap
                identifier={identifier}
                name={name}
                key={name}
                label={generateLabelForPluginField({ label, properties }) as string}
                formik={formikRef.current}
              />
            </Container>
          )

        default:
          return <></>
      }
    },
    [] // eslint-disable-line react-hooks/exhaustive-deps
  )

  const insertNameFieldToPluginInputs = (existingInputs: {
    [key: string]: PluginInput
  }): { [key: string]: PluginInput } => {
    const inputsClone = Object.assign(
      {
        name: {
          type: 'string',
          description: 'Name of the step'
        }
      },
      existingInputs
    )
    return inputsClone
  }

  const getPluginInputsFromSpec = useCallback((pluginSpec: string): PluginInputs => {
    if (!pluginSpec) {
      return {}
    }
    try {
      const pluginSpecAsObj = parse(pluginSpec)
      return get(pluginSpecAsObj, PluginSpecInputPath, {})
    } catch (ex) {
      /* ignore error */
    }
    return {}
  }, [])

  const getInitialFormValuesWithFieldDefaults = useCallback((pluginInputs: PluginInputs): PluginInputs => {
    return Object.entries(pluginInputs).reduce((acc, [field, inputObj]) => {
      if (inputObj?.default) {
        set(acc, field, inputObj.default)
      }
      return acc
    }, {} as PluginInputs)
  }, [])

  const getInitialFormValuesFromYAML = useCallback(
    ({ pathToField, formData }: { pathToField: string[]; formData: PluginFormDataInterface }): PluginInputs => {
      let pluginInputsWithYAMLValues: PluginInputs = {}
      const fieldFormikPathPrefix = pathToField.join('.')
      if (!isEmpty(formData)) {
        const _category_ = get(formData, 'type') as PluginCategory
        pluginInputsWithYAMLValues = Object.entries(
          get(formData, _category_ === PluginCategory.Harness ? PluginSpecPath : PluginSpecInputPath, {})
        ).reduce((acc, [field, value]) => {
          const formikFieldName = getFormikFieldName({
            fieldName: field,
            fieldFormikPathPrefix,
            fieldFormikPathPrefixWithSpec: `${fieldFormikPathPrefix}.spec`,
            category: _category_
          })
          set(acc, formikFieldName, value)
          return acc
        }, {} as PluginInputs)
      }
      if (has(formData, 'name')) {
        set(
          pluginInputsWithYAMLValues,
          fieldFormikPathPrefix ? `${fieldFormikPathPrefix}.name` : 'name',
          get(formData, 'name')
        )
      }
      return pluginInputsWithYAMLValues
    },
    []
  )

  /**
   * Toolbar to sync updates from YAML into UI
   */
  const renderFieldSyncToolbar = useCallback((): JSX.Element => {
    return (
      <Layout.Vertical spacing="medium">
        <Layout.Horizontal spacing="small" flex={{ justifyContent: 'flex-start' }}>
          <Icon color={Color.ORANGE_500} name="warning-sign" />
          <Text color={Color.ORANGE_500}>{getString('pipelineConfig.yamlUpdated')}</Text>
        </Layout.Horizontal>
        <Layout.Horizontal spacing="medium" flex={{ justifyContent: 'flex-start' }}>
          <Button
            size={ButtonSize.SMALL}
            text={getString('refresh')}
            variation={ButtonVariation.PRIMARY}
            onClick={() => {
              const { pathToField = [], formData } = pluginFieldUpdateData
              setFormInitialValues((initialValues?: PluginFormDataInterface) => {
                if (!isUndefined(initialValues) && !isEmpty(pathToField)) {
                  const valueCopy = { ...initialValues }
                  set(valueCopy, pathToField.join('.'), formData)
                  return valueCopy
                }
                return initialValues
              })
              setShowSyncToolbar(false)
            }}
          />
          <Button
            size={ButtonSize.SMALL}
            text={getString('discard')}
            variation={ButtonVariation.SECONDARY}
            onClick={() => setShowSyncToolbar(false)}
          />
        </Layout.Horizontal>
      </Layout.Vertical>
    )
  }, [pluginFieldUpdateData]) // eslint-disable-line react-hooks/exhaustive-deps

  const getFormikFieldName = useCallback(
    ({
      fieldName,
      fieldFormikPathPrefix,
      fieldFormikPathPrefixWithSpec,
      category
    }: {
      fieldName: string
      fieldFormikPathPrefix: string
      fieldFormikPathPrefixWithSpec: string
      category?: PluginCategory
    }): string => {
      if (!category) {
        return ''
      }
      if (fieldName === 'name') {
        return `${fieldFormikPathPrefix}.name`
      } else {
        if (category === PluginCategory.Drone) {
          return `${fieldFormikPathPrefixWithSpec}.${PluginsInputPath}.${fieldName}`
        }
        return `${fieldFormikPathPrefixWithSpec}.${fieldName}`
      }
    },
    []
  )

  const sanitizePluginYAMLPayload = useCallback(
    (existingPayload: Record<string, any>, validKeys: string[]): Record<string, any> => {
      /* Ensure only keys in a plugin's input are added to the actual YAML, everything else should get removed */
      return pick(get(existingPayload, PluginSpecInputPath), validKeys)
    },
    []
  )

  const renderPluginConfigForm = useCallback((): JSX.Element => {
    const pluginInputs = getPluginInputsFromSpec(get(plugin, PluginSpecPath, '') as string) as PluginInputs
    const allPluginInputs = insertNameFieldToPluginInputs(pluginInputs)
    const { isUpdate } = pluginDataFromYAML
    const { pathToField } = isUpdate
      ? pluginDataFromYAML
      : { pathToField: generateDefaultStepInsertionPath().split('.') }
    const fieldFormikPathPrefix = pathToField.join('.')
    const fieldFormikPathPrefixWithSpec = `${fieldFormikPathPrefix}.${PluginSpecPath}`
    return (
      <Layout.Vertical
        spacing="large"
        className={cx(css.configForm, { [css.configFormWithSyncToolbar]: showSyncToolbar })}>
        <Layout.Horizontal spacing="small" flex={{ justifyContent: 'flex-start' }}>
          <Icon
            name="arrow-left"
            size={18}
            onClick={() => {
              setPlugin(undefined)
              if (pluginCategory === PluginCategory.Drone) {
                setPanelView(PluginPanelView.Listing)
              } else if (pluginCategory === PluginCategory.Harness) {
                setPanelView(PluginPanelView.Category)
              }
            }}
            className={css.arrow}
          />
          {plugin?.identifier && (
            <Text font={{ variation: FontVariation.H4 }}>
              {getString(isUpdate ? 'updateLabel' : 'addLabel')} {plugin.identifier} {getString('plugins.stepLabel')}
            </Text>
          )}
        </Layout.Horizontal>
        {showSyncToolbar && <Container>{renderFieldSyncToolbar()}</Container>}
        <Container className={cx(css.form, { [css.formHeightWithSyncToolbar]: showSyncToolbar })}>
          <Formik<PluginFormDataInterface>
            initialValues={formInitialValues || {}}
            onSubmit={(values: PluginFormDataInterface) => {
              let payloadForYAMLUpdate = get(values, pathToField, {})
              if (isEmpty(payloadForYAMLUpdate)) {
                return
              }
              if (pluginCategory === PluginCategory.Drone) {
                payloadForYAMLUpdate = sanitizePluginYAMLPayload(payloadForYAMLUpdate, Object.keys(allPluginInputs))
              }
              const updatedYAMLPayload = set({}, PluginSpecInputPath, payloadForYAMLUpdate)
              set(updatedYAMLPayload, 'type', pluginCategory)
              set(updatedYAMLPayload, `${PluginSpecPath}.name`, plugin?.identifier)
              onPluginAddUpdate({
                pathToField,
                isUpdate,
                formData: set({}, pathToField, updatedYAMLPayload)
              })
            }}
            enableReinitialize>
            {formik => {
              formikRef.current = formik
              return (
                <FormikForm height="100%" flex={{ justifyContent: 'space-between', alignItems: 'baseline' }}>
                  <Layout.Vertical flex={{ alignItems: 'flex-start' }} height="inherit" spacing="medium">
                    <Layout.Vertical
                      width="100%"
                      className={css.formFields}
                      spacing="xsmall"
                      flex={{ justifyContent: 'space-between' }}>
                      {pluginCategory === PluginCategory.Harness ? (
                        <RunStep prefix={fieldFormikPathPrefix} />
                      ) : Object.keys(pluginInputs).length > 0 ? (
                        <Layout.Vertical width="inherit">
                          {Object.keys(allPluginInputs).map((field: string) => {
                            return renderPluginFormField({
                              label: field,
                              identifier: field,
                              /* "name" gets rendered at outside step's spec */
                              name: getFormikFieldName({
                                fieldName: field,
                                fieldFormikPathPrefix,
                                fieldFormikPathPrefixWithSpec,
                                category: PluginCategory.Drone
                              }),
                              properties: get(allPluginInputs, field)
                            })
                          })}
                        </Layout.Vertical>
                      ) : (
                        <></>
                      )}
                    </Layout.Vertical>
                    <Container margin={{ top: 'small', bottom: 'small' }}>
                      <Button
                        variation={ButtonVariation.PRIMARY}
                        text={getString(isUpdate ? 'updateLabel' : 'addLabel')}
                        type="submit"
                        disabled={!formik.dirty}
                      />
                    </Container>
                  </Layout.Vertical>
                </FormikForm>
              )
            }}
          </Formik>
        </Container>
      </Layout.Vertical>
    )
  }, [plugin, pluginCategory, pluginDataFromYAML, pluginFieldUpdateData, showSyncToolbar, formInitialValues]) // eslint-disable-line react-hooks/exhaustive-deps

  const renderPluginsPanel = useCallback((): JSX.Element => {
    switch (panelView) {
      case PluginPanelView.Category:
        return renderPluginCategories()
      case PluginPanelView.Listing:
        return renderPlugins()
      case PluginPanelView.Configuration:
        return renderPluginConfigForm()
      default:
        return <></>
    }
  }, [panelView, renderPluginCategories, renderPluginConfigForm, renderPlugins])

  return (
    <Layout.Vertical height="inherit">
      <Container height="inherit">{renderPluginsPanel()}</Container>
    </Layout.Vertical>
  )
}
