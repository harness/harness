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

import React, { useCallback, useEffect, useRef, useState } from 'react'
import { Formik, FormikContextType } from 'formik'
import { parse } from 'yaml'
import cx from 'classnames'
import { capitalize, get, has, omit, pick, set } from 'lodash-es'
import { Classes, PopoverInteractionKind, PopoverPosition } from '@blueprintjs/core'
import { Color, FontVariation } from '@harnessio/design-system'
import { Icon, IconProps } from '@harnessio/icons'
import {
  Button,
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
import { PipelineEntity, CodeLensAction, CodeLensClickMetaData } from 'components/PipelineConfigPanel/types'
import { RunStep } from './Steps/HarnessSteps/RunStep/RunStep'

import css from './PluginsPanel.module.scss'

export interface EntityAddUpdateInterface extends Partial<CodeLensClickMetaData> {
  pathToField: string[]
  isUpdate: boolean
  formData: PluginFormDataInterface
}

export interface PluginsPanelInterface {
  pluginDataFromYAML: EntityAddUpdateInterface
  onPluginAddUpdate: (data: EntityAddUpdateInterface) => void
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

interface PluginInsertionTemplateInterface {
  name?: string
  type: PluginCategory.Drone
  spec: {
    name: string
    inputs: { [key: string]: string }
  }
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

const PluginInsertionTemplate: PluginInsertionTemplateInterface = {
  name: '<step-name>',
  type: PluginCategory.Drone,
  spec: {
    name: '<plugin-uid-from-database>',
    inputs: {
      '<param1>': '<value1>',
      '<param2>': '<value2>'
    }
  }
}

const PluginNameFieldPath = 'spec.name'
const PluginInputsFieldPath = 'spec.inputs'

const LIST_FETCHING_LIMIT = 100

const RunStepSpec: TypesPlugin = {
  uid: 'run'
}

export const PluginsPanel = (props: PluginsPanelInterface): JSX.Element => {
  const { pluginDataFromYAML, onPluginAddUpdate } = props
  const { getString } = useStrings()
  const [category, setCategory] = useState<PluginCategory>()
  const [panelView, setPanelView] = useState<PluginPanelView>(PluginPanelView.Category)
  const [plugin, setPlugin] = useState<TypesPlugin>()
  const [plugins, setPlugins] = useState<TypesPlugin[]>([])
  const [query, setQuery] = useState<string>('')
  const [loading, setLoading] = useState<boolean>(false)
  const formikRef = useRef<FormikContextType<PluginFormDataInterface>>()

  const PluginCategories: PluginCategoryInterface[] = [
    {
      category: PluginCategory.Harness,
      name: capitalize(getString('run')),
      description: getString('pluginsPanel.run.helptext'),
      icon: { name: 'run-step', size: 15 }
    },
    {
      category: PluginCategory.Drone,
      name: capitalize(getString('plugins.title')),
      description: getString('pluginsPanel.plugins.helptext'),
      icon: { name: 'plugin-ci-step', size: 18 }
    }
  ]

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

  useEffect(() => {
    const { entity, action } = pluginDataFromYAML
    if (entity === PipelineEntity.STEP) {
      switch (action) {
        case CodeLensAction.EDIT:
          handleIncomingPluginData(pluginDataFromYAML)
          break
        case CodeLensAction.ADD:
          setPanelView(PluginPanelView.Category)
          break
      }
    }
  }, [pluginDataFromYAML])

  useEffect(() => {
    if (category === PluginCategory.Drone) {
      fetchAllPlugins().then(response => setPlugins(response))
    }
  }, [category])

  useEffect(() => {
    if (panelView !== PluginPanelView.Listing) return

    if (query) {
      setPlugins(existingPlugins => existingPlugins.filter((item: TypesPlugin) => item.uid?.includes(query)))
    } else {
      fetchAllPlugins().then(response => setPlugins(response))
    }
  }, [query])

  const handleIncomingPluginData = useCallback(
    (data: EntityAddUpdateInterface) => {
      const { formData } = data
      const _category = get(formData, 'type') as PluginCategory
      if (_category === PluginCategory.Harness) {
        handlePluginCategoryClick(_category)
      } else {
        setCategory(PluginCategory.Drone)
        fetchAllPlugins().then(response => {
          const matchingPlugin = response?.find((_plugin: TypesPlugin) => _plugin?.uid === get(formData, 'spec.name'))
          if (matchingPlugin) {
            setPlugin(matchingPlugin)
            setPanelView(PluginPanelView.Configuration)
          }
        })
      }
    },
    [plugins]
  )

  const handlePluginCategoryClick = useCallback((selectedCategory: PluginCategory) => {
    setCategory(selectedCategory)
    if (selectedCategory === PluginCategory.Drone) {
      setPanelView(PluginPanelView.Listing)
    } else if (selectedCategory === PluginCategory.Harness) {
      setPlugin(RunStepSpec)
      setPanelView(PluginPanelView.Configuration)
    }
  }, [])

  const renderPluginCategories = (): JSX.Element => {
    return (
      <Layout.Vertical spacing="large">
        <Text font={{ variation: FontVariation.H4 }}>{getString('stepCategory.select')}</Text>
        <Layout.Vertical>
          {PluginCategories.map((item: PluginCategoryInterface) => {
            const { name, category: pluginCategory, description, icon } = item
            return (
              <Card
                className={cx(css.pluginCategoryCard, css.cursor)}
                key={pluginCategory}
                onClick={() => handlePluginCategoryClick(pluginCategory)}>
                <Layout.Horizontal flex={{ justifyContent: 'space-between' }}>
                  <Layout.Horizontal
                    onClick={() => handlePluginCategoryClick(pluginCategory)}
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
                      onClick={() => handlePluginCategoryClick(pluginCategory)}
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
  }

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
            const { uid, description } = pluginItem
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
                key={uid}
                width="100%">
                <Icon name={'plugin-ci-step'} size={25} />
                <Layout.Vertical padding={{ left: 'small' }} spacing="xsmall" className={css.pluginInfo}>
                  <Text
                    color={Color.GREY_900}
                    className={css.fontWeight600}
                    font={{ variation: FontVariation.BODY2_SEMI }}>
                    {uid}
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
  }, [loading, plugins, query])

  const generateFriendlyName = useCallback((pluginName: string): string => {
    return capitalize(pluginName.split('_').join(' '))
  }, [])

  const generateLabelForPluginField = useCallback(
    ({ name, properties }: { name: string; properties: PluginInput }): JSX.Element | string => {
      const { description } = properties
      return (
        <Layout.Horizontal spacing="small" flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
          {name && <Text font={{ variation: FontVariation.FORM_LABEL }}>{generateFriendlyName(name)}</Text>}
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
    []
  )

  const renderPluginFormField = ({ name, properties }: { name: string; properties: PluginInput }): JSX.Element => {
    const { type, options } = properties

    switch (type) {
      case ValueType.STRING: {
        const { isExtended } = options || {}
        const WrapperComponent = isExtended ? FormInput.TextArea : FormInput.Text
        return (
          <WrapperComponent
            name={name}
            label={generateLabelForPluginField({ name, properties })}
            style={{ width: '100%' }}
            key={name}
          />
        )
      }
      case ValueType.BOOLEAN:
        return (
          <Container className={css.toggle}>
            <FormInput.Toggle
              name={name}
              label={generateLabelForPluginField({ name, properties }) as string}
              style={{ width: '100%' }}
              key={name}
            />
          </Container>
        )
      case ValueType.ARRAY:
        return (
          <Container margin={{ bottom: 'large' }}>
            <MultiList
              name={name}
              label={generateLabelForPluginField({ name, properties }) as string}
              formik={formikRef.current}
            />
          </Container>
        )
      case ValueType.OBJECT:
        return (
          <Container margin={{ bottom: 'large' }}>
            <MultiMap
              name={name}
              label={generateLabelForPluginField({ name, properties }) as string}
              formik={formikRef.current}
            />
          </Container>
        )

      default:
        return <></>
    }
  }

  /* Ensures no junk/unrecognized form values are set in the YAML */
  const sanitizeFormData = useCallback(
    (existingFormData: PluginFormDataInterface, pluginInputs: PluginInputs): PluginFormDataInterface => {
      return pick(existingFormData, Object.keys(pluginInputs))
    },
    []
  )

  const constructPayloadForYAMLInsertion = (
    pluginFormData: PluginFormDataInterface,
    pluginMetadata?: TypesPlugin
  ): PluginFormDataInterface => {
    const { container = {}, name } = pluginFormData
    const formDataWithoutName = omit(pluginFormData, 'name')
    let payload = { ...PluginInsertionTemplate }
    switch (category) {
      case PluginCategory.Drone:
        /* Step name is optional, set only if specified by user */
        if (name) {
          set(payload, 'name', name)
        } else {
          payload = omit(payload, 'name')
        }
        set(payload, PluginNameFieldPath, pluginMetadata?.uid)
        set(payload, PluginInputsFieldPath, formDataWithoutName)
        return payload
      case PluginCategory.Harness:
        return {
          ...(name && { name }),
          type: PluginCategory.Harness,
          ...(Object.keys(container).length === 1 && has(container, 'image')
            ? { spec: { ...formDataWithoutName, container: get(container, 'image') } }
            : { spec: formDataWithoutName })
        }
      default:
        return {}
    }
  }

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
      return get(pluginSpecAsObj, 'spec.inputs', {})
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
    (_category: PluginCategory, formValues: PluginFormDataInterface): PluginInputs => {
      return Object.assign(
        { name: get(formValues, 'name') },
        formValues
          ? Object.entries(get(formValues, _category === PluginCategory.Harness ? 'spec' : 'spec.inputs', {})).reduce(
              (acc, [field, value]) => {
                set(acc, field, value)
                return acc
              },
              {} as PluginInputs
            )
          : {}
      )
    },
    []
  )

  const renderPluginConfigForm = useCallback((): JSX.Element => {
    const pluginInputs = getPluginInputsFromSpec(get(plugin, 'spec', '') as string) as PluginInputs
    const allPluginInputs = insertNameFieldToPluginInputs(pluginInputs)
    const { isUpdate, formData, ...rest } = pluginDataFromYAML
    return (
      <Layout.Vertical spacing="large" className={css.configForm}>
        <Layout.Horizontal spacing="small" flex={{ justifyContent: 'flex-start' }}>
          <Icon
            name="arrow-left"
            size={18}
            onClick={() => {
              setPlugin(undefined)
              if (category === PluginCategory.Drone) {
                setPanelView(PluginPanelView.Listing)
              } else if (category === PluginCategory.Harness) {
                setPanelView(PluginPanelView.Category)
              }
            }}
            className={css.arrow}
          />
          {plugin?.uid && (
            <Text font={{ variation: FontVariation.H4 }}>
              {getString(isUpdate ? 'updateLabel' : 'addLabel')} {plugin.uid} {getString('plugins.stepLabel')}
            </Text>
          )}
        </Layout.Horizontal>
        <Container className={css.form}>
          <Formik<PluginFormDataInterface>
            initialValues={
              isUpdate && category
                ? getInitialFormValuesFromYAML(category, formData)
                : getInitialFormValuesWithFieldDefaults(pluginInputs)
            }
            onSubmit={(values: PluginFormDataInterface) => {
              onPluginAddUpdate({
                isUpdate,
                formData: constructPayloadForYAMLInsertion(
                  category === PluginCategory.Drone ? sanitizeFormData(values, allPluginInputs) : values,
                  plugin
                ),
                ...rest
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
                      {category === PluginCategory.Harness ? (
                        <RunStep />
                      ) : Object.keys(pluginInputs).length > 0 ? (
                        <Layout.Vertical width="inherit">
                          {Object.keys(allPluginInputs).map((field: string) => {
                            return renderPluginFormField({ name: field, properties: get(allPluginInputs, field) })
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
  }, [plugin, category, pluginDataFromYAML])

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
  }, [loading, plugins, panelView, category, pluginDataFromYAML])

  return (
    <Layout.Vertical height="inherit">
      <Container height="inherit">{renderPluginsPanel()}</Container>
    </Layout.Vertical>
  )
}
