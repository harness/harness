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
import { capitalize, get, has, omit, set } from 'lodash-es'
import { Classes, PopoverInteractionKind, PopoverPosition } from '@blueprintjs/core'
import { Color, FontVariation } from '@harnessio/design-system'
import { Icon, IconProps } from '@harnessio/icons'
import {
  Accordion,
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

import css from './PluginsPanel.module.scss'

export interface PluginsPanelInterface {
  onPluginAddUpdate: (isUpdate: boolean, pluginFormData: PluginForm) => void
}

export interface PluginForm {
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
  type: 'plugin'
  spec: {
    name: string
    inputs: { [key: string]: string }
  }
}

enum PluginCategory {
  Harness,
  Drone
}

enum PluginPanelView {
  Category,
  Listing,
  Configuration
}

const PluginInsertionTemplate: PluginInsertionTemplateInterface = {
  name: '<step-name>',
  type: 'plugin',
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

const RunStep: TypesPlugin = {
  uid: 'run'
}

const DockerTemp: TypesPlugin = {
  uid: 'docker',
  description: 'The Docker plugin can be used to build and publish images to the Docker\nregistry.\n',
  logo: '',
  spec: '{\n  "kind": "plugin",\n  "type": "step",\n  "name": "docker",\n  "spec": {\n    "name": "Docker",\n    "description": "The Docker plugin can be used to build and publish images to the Docker\\nregistry.\\n",\n    "step": {\n      "type": "script",\n      "spec": {\n        "image": "plugins/docker",\n        "envs": {\n          "PLUGIN_ADD_HOST": "${{ inputs.add_host }}",\n          "PLUGIN_AUTO_TAG": "${{ inputs.auto_tag }}",\n          "PLUGIN_AUTO_TAG_SUFFIX": "${{ inputs.auto_tag_suffix }}",\n          "PLUGIN_BIP": "${{ inputs.bip }}",\n          "PLUGIN_BUILD_ARGS": "${{ inputs.build_args }}",\n          "PLUGIN_BUILD_ARGS_FROM_ENV": "${{ inputs.build_args_from_env }}",\n          "PLUGIN_CACHE_FROM": "${{ inputs.cache_from }}",\n          "PLUGIN_COMPRESS": "${{ inputs.compress }}",\n          "PLUGIN_CONTEXT": "${{ inputs.context }}",\n          "PLUGIN_CUSTOM_DNS": "${{ inputs.custom_dns }}",\n          "PLUGIN_CUSTOM_DNS_SEARCH": "${{ inputs.custom_dns_search }}",\n          "PLUGIN_CUSTOM_LABELS": "${{ inputs.custom_labels }}",\n          "PLUGIN_DAEMON_OFF": "${{ inputs.daemon_off }}",\n          "PLUGIN_DEBUG": "${{ inputs.debug }}",\n          "PLUGIN_DOCKERFILE": "${{ inputs.dockerfile }}",\n          "PLUGIN_DRY_RUN": "${{ inputs.dry_run }}",\n          "PLUGIN_EMAIL": "${{ inputs.email }}",\n          "PLUGIN_EXPERIMENTAL": "${{ inputs.experimental }}",\n          "PLUGIN_FORCE_TAG": "${{ inputs.force_tag }}",\n          "PLUGIN_INSECURE": "${{ inputs.insecure }}",\n          "PLUGIN_IPV6": "${{ inputs.ipv6 }}",\n          "PLUGIN_LABEL_SCHEMA": "${{ inputs.label_schema }}",\n          "PLUGIN_LAUNCH_DEBUG": "${{ inputs.launch_debug }}",\n          "PLUGIN_MIRROR": "${{ inputs.mirror }}",\n          "PLUGIN_MTU": "${{ inputs.mtu }}",\n          "PLUGIN_NO_CACHE": "${{ inputs.no_cache }}",\n          "PLUGIN_PASSWORD": "${{ inputs.password }}",\n          "PLUGIN_PLATFORM": "${{ inputs.platform }}",\n          "PLUGIN_PULL_IMAGE": "${{ inputs.pull_image }}",\n          "PLUGIN_PURGE": "${{ inputs.purge }}",\n          "PLUGIN_REGISTRY": "${{ inputs.registry }}",\n          "PLUGIN_REPO": "${{ inputs.repo }}",\n          "PLUGIN_SECRET": "${{ inputs.secret }}",\n          "PLUGIN_SQUASH": "${{ inputs.squash }}",\n          "PLUGIN_SSH-AGENT-KEY": "${{ inputs.ssh-agent-key }}",\n          "PLUGIN_STORAGE_DRIVER": "${{ inputs.storage_driver }}",\n          "PLUGIN_STORAGE_PATH": "${{ inputs.storage_path }}",\n          "PLUGIN_TAGS": "${{ inputs.tags }}",\n          "PLUGIN_TARGET": "${{ inputs.target }}",\n          "PLUGIN_USERNAME": "${{ inputs.username }}"\n        }\n      }\n    },\n    "inputs": {\n      "add_host": {\n        "type": "string",\n        "description": "additional host:IP mapping"\n      },\n      "auto_tag": {\n        "type": "boolean",\n        "description": "generate tag names automatically based on git branch and git tag"\n      },\n      "auto_tag_suffix": {\n        "type": "string",\n        "description": "generate tag names with this suffix"\n      },\n      "bip": {\n        "type": "boolean",\n        "description": "use for pass bridge ip"\n      },\n      "build_args": {\n        "type": "string",\n        "description": "pass custom arguments to docker build"\n      },\n      "build_args_from_env": {\n        "type": "string",\n        "description": "pass the envvars as custom arguments to docker build"\n      },\n      "cache_from": {\n        "type": "string",\n        "description": "images to consider as cache sources"\n      },\n      "compress": {\n        "type": "boolean",\n        "description": "compress the build context using gzip"\n      },\n      "context": {\n        "type": "string",\n        "description": "the context path to use, defaults to root of the git repo"\n      },\n      "custom_dns": {\n        "type": "string",\n        "description": "set custom dns servers for the container"\n      },\n      "custom_dns_search": {\n        "type": "string",\n        "description": "docker daemon dns search domains"\n      },\n      "custom_labels": {\n        "type": "string",\n        "description": "additional k=v labels"\n      },\n      "daemon_off": {\n        "type": "boolean",\n        "description": "don\'t start the docker daemon"\n      },\n      "debug": {\n        "type": "boolean",\n        "description": "launch the docker daemon in verbose debug mode"\n      },\n      "dockerfile": {\n        "type": "string",\n        "description": "dockerfile to be used",\n        "default": "Dockerfile"\n      },\n      "dry_run": {\n        "type": "string",\n        "description": "boolean if the docker image should not be pushed at the end"\n      },\n      "email": {\n        "type": "string",\n        "description": "docker email"\n      },\n      "experimental": {\n        "type": "boolean",\n        "description": "docker daemon Experimental mode"\n      },\n      "force_tag": {\n        "type": "boolean",\n        "description": "replace existing matched image tags"\n      },\n      "insecure": {\n        "type": "boolean",\n        "description": "enable insecure communication to this registry"\n      },\n      "ipv6": {\n        "type": "string",\n        "description": "docker daemon IPv6 networking"\n      },\n      "label_schema": {\n        "type": "string",\n        "description": "label-schema labels"\n      },\n      "launch_debug": {\n        "type": "boolean",\n        "description": "launch the docker daemon in verbose debug mode"\n      },\n      "mirror": {\n        "type": "string",\n        "description": "use a mirror registry instead of pulling images directly from the central Hub"\n      },\n      "mtu": {\n        "type": "string",\n        "description": "docker daemon custom mtu setting"\n      },\n      "no_cache": {\n        "type": "string",\n        "description": "do not use cached intermediate containers"\n      },\n      "password": {\n        "type": "string",\n        "description": "authenticates with this password",\n        "mask": true\n      },\n      "platform": {\n        "type": "string",\n        "description": "specify the target platform for the build output, (for example, linux/amd64, linux/arm64, or darwin/amd64)."\n      },\n      "pull_image": {\n        "type": "boolean",\n        "description": "force pull base image at build time"\n      },\n      "purge": {\n        "type": "boolean",\n        "description": "boolean if cleanup of the docker image should be done at the end",\n        "default": "true"\n      },\n      "registry": {\n        "type": "string",\n        "description": "authenticates to this registry"\n      },\n      "repo": {\n        "type": "string",\n        "description": "repository name for the image"\n      },\n      "secret": {\n        "type": "string",\n        "description": "Use buildkit to pass secrets to the dockerbuild. Eg `id=mysecret,src=secret-file`"\n      },\n      "squash": {\n        "type": "boolean",\n        "description": "squash the layers at build time"\n      },\n      "ssh-agent-key": {\n        "type": "string",\n        "description": "private key to use for ssh passthrough, see https://docs.docker.com/engine/reference/commandline/buildx_build/#ssh\\n",\n        "mask": true\n      },\n      "storage_driver": {\n        "type": "string",\n        "description": "supports `aufs`, `overlay` or `vfs` drivers"\n      },\n      "storage_path": {\n        "type": "string",\n        "description": "docker daemon storage path"\n      },\n      "tags": {\n        "type": "array",\n        "description": "repository tag(s) for the image"\n      },\n      "target": {\n        "type": "string",\n        "description": "the build target to use, must be defined in the docker file"\n      },\n      "username": {\n        "type": "string",\n        "description": "authenticates with this username"\n      }\n    }\n  }\n}\n'
}

export const PluginsPanel = ({ onPluginAddUpdate }: PluginsPanelInterface): JSX.Element => {
  const { getString } = useStrings()
  const [category, setCategory] = useState<PluginCategory>(PluginCategory.Drone)
  const [panelView, setPanelView] = useState<PluginPanelView>(PluginPanelView.Configuration)
  const [plugin, setPlugin] = useState<TypesPlugin>(DockerTemp)
  const [plugins, setPlugins] = useState<TypesPlugin[]>([])
  const [query, setQuery] = useState<string>('')
  const [loading, setLoading] = useState<boolean>(false)
  const formikRef = useRef<FormikContextType<PluginForm>>()

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

  const fetchAllPlugins = useCallback(async (): Promise<void> => {
    try {
      setLoading(true)
      const pluginsPage1 = await fetchPlugins(1)
      const pluginsPage2 = await fetchPlugins(2)
      setPlugins([...pluginsPage1, ...pluginsPage2])
    } catch (ex) {
      /* ignore exception */
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (category === PluginCategory.Drone) {
      fetchAllPlugins()
    }
  }, [category])

  useEffect(() => {
    if (panelView !== PluginPanelView.Listing) return

    if (query) {
      setPlugins(existingPlugins => existingPlugins.filter((item: TypesPlugin) => item.uid?.includes(query)))
    } else {
      fetchAllPlugins()
    }
  }, [query])

  const handlePluginCategoryClick = useCallback((selectedCategory: PluginCategory) => {
    setCategory(selectedCategory)
    if (selectedCategory === PluginCategory.Drone) {
      setPanelView(PluginPanelView.Listing)
    } else if (selectedCategory === PluginCategory.Harness) {
      setPlugin(RunStep)
      setPanelView(PluginPanelView.Configuration)
    }
  }, [])

  const renderPluginCategories = (): JSX.Element => {
    return (
      <>
        {PluginCategories.map((item: PluginCategoryInterface) => {
          const { name, category: pluginCategory, description, icon } = item
          return (
            <Card className={css.pluginCard} key={pluginCategory}>
              <Layout.Horizontal flex={{ justifyContent: 'space-between' }}>
                <Layout.Horizontal
                  onClick={() => handlePluginCategoryClick(pluginCategory)}
                  flex={{ justifyContent: 'flex-start' }}
                  className={css.plugin}>
                  <Container className={css.pluginIcon}>
                    <Icon {...icon} />
                  </Container>
                  <Layout.Vertical padding={{ left: 'medium' }} spacing="xsmall">
                    <Text
                      color={Color.GREY_900}
                      className={css.pluginCategory}
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
                    className={css.plugin}
                  />
                </Container>
              </Layout.Horizontal>
            </Card>
          )
        })}
      </>
    )
  }

  const renderPlugins = useCallback((): JSX.Element => {
    return loading ? (
      <Container flex={{ justifyContent: 'center' }} padding="large">
        <Icon name="steps-spinner" color={Color.PRIMARY_7} size={25} />
      </Container>
    ) : (
      <Layout.Vertical spacing="small" padding={{ top: 'small' }}>
        <Layout.Horizontal flex={{ justifyContent: 'space-between' }} padding={{ left: 'small', right: 'xlarge' }}>
          <Layout.Horizontal
            flex={{ justifyContent: 'flex-start', alignItems: 'center' }}
            spacing="small"
            padding={{ top: 'medium', bottom: 'medium', left: 'medium' }}>
            <Icon
              name="arrow-left"
              size={18}
              onClick={() => {
                setPanelView(PluginPanelView.Category)
              }}
              className={css.arrow}
            />
            <Text font={{ variation: FontVariation.H5 }}>{getString('plugins.select')}</Text>
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
                padding={{ left: 'large', top: 'medium', bottom: 'medium', right: 'large' }}
                className={css.plugin}
                onClick={() => {
                  setPanelView(PluginPanelView.Configuration)
                  setPlugin(pluginItem)
                }}
                key={uid}>
                <Icon name={'plugin-ci-step'} size={25} />
                <Layout.Vertical padding={{ left: 'small' }} spacing="xsmall">
                  <Text
                    color={Color.GREY_900}
                    className={css.pluginCategory}
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

      default:
        return <></>
    }
  }

  const constructPayloadForYAMLInsertion = (pluginFormData: PluginForm, pluginMetadata?: TypesPlugin): PluginForm => {
    const { name, container = {} } = pluginFormData
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
        set(payload, PluginInputsFieldPath, omit(pluginFormData, 'name'))
        return payload
      case PluginCategory.Harness:
        return {
          ...(name && { name }),
          type: 'run',
          ...(Object.keys(container).length === 1 && has(container, 'image')
            ? { spec: { ...pluginFormData, container: get(container, 'image') } }
            : { spec: pluginFormData })
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

  const getInitialFormValues = useCallback((pluginInputs: PluginInputs): PluginInputs => {
    return Object.entries(pluginInputs).reduce((acc, [field, inputObj]) => {
      if (inputObj?.default) {
        set(acc, field, inputObj.default)
      }
      return acc
    }, {} as PluginInputs)
  }, [])

  const renderPluginConfigForm = useCallback((): JSX.Element => {
    const pluginInputs = getPluginInputsFromSpec(get(plugin, 'spec', '') as string) as PluginInputs
    const allPluginInputs = insertNameFieldToPluginInputs(pluginInputs)
    return (
      <Layout.Vertical spacing="large" className={css.configForm}>
        <Layout.Horizontal spacing="small" flex={{ justifyContent: 'flex-start' }}>
          <Icon
            name="arrow-left"
            size={18}
            onClick={() => {
              // setPlugin(undefined)
              if (category === PluginCategory.Drone) {
                setPanelView(PluginPanelView.Listing)
              } else if (category === PluginCategory.Harness) {
                setPanelView(PluginPanelView.Category)
              }
            }}
            className={css.arrow}
          />
          {plugin?.uid && (
            <Text font={{ variation: FontVariation.H5 }}>
              {getString('addLabel')} {plugin.uid} {getString('plugins.stepLabel')}
            </Text>
          )}
        </Layout.Horizontal>
        <Container className={css.form}>
          <Formik<PluginForm>
            initialValues={getInitialFormValues(pluginInputs)}
            onSubmit={(formData: PluginForm) => {
              onPluginAddUpdate?.(false, constructPayloadForYAMLInsertion(formData, plugin))
            }}
            validate={(formData: PluginForm) => console.log(formData)}>
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
                        <Layout.Vertical width="inherit">
                          <FormInput.Text
                            name={'name'}
                            label={getString('name')}
                            style={{ width: '100%' }}
                            key={'name'}
                          />
                          <FormInput.TextArea
                            name={'script'}
                            label={getString('pluginsPanel.run.script')}
                            style={{ width: '100%' }}
                            key={'script'}
                          />
                          <FormInput.Select
                            name={'shell'}
                            label={getString('pluginsPanel.run.shell')}
                            style={{ width: '100%' }}
                            key={'shell'}
                            items={[
                              { label: getString('pluginsPanel.run.sh'), value: 'sh' },
                              { label: getString('pluginsPanel.run.bash'), value: 'bash' },
                              { label: getString('pluginsPanel.run.powershell'), value: 'powershell' },
                              { label: getString('pluginsPanel.run.pwsh'), value: 'pwsh' }
                            ]}
                          />
                          <Accordion activeId="">
                            <Accordion.Panel
                              id="container"
                              summary="Container"
                              details={
                                <Layout.Vertical className={css.indent}>
                                  <FormInput.Text
                                    name={'container.image'}
                                    label={getString('pluginsPanel.run.image')}
                                    style={{ width: '100%' }}
                                    key={'container.image'}
                                  />
                                  <FormInput.Select
                                    name={'container.pull'}
                                    label={getString('pluginsPanel.run.pull')}
                                    style={{ width: '100%' }}
                                    key={'container.pull'}
                                    items={[
                                      { label: getString('pluginsPanel.run.always'), value: 'always' },
                                      { label: getString('pluginsPanel.run.never'), value: 'never' },
                                      { label: getString('pluginsPanel.run.ifNotExists'), value: 'if-not-exists' }
                                    ]}
                                  />
                                  <FormInput.Text
                                    name={'container.entrypoint'}
                                    label={getString('pluginsPanel.run.entrypoint')}
                                    style={{ width: '100%' }}
                                    key={'container.entrypoint'}
                                  />
                                  <FormInput.Text
                                    name={'container.network'}
                                    label={getString('pluginsPanel.run.network')}
                                    style={{ width: '100%' }}
                                    key={'container.network'}
                                  />
                                  <FormInput.Text
                                    name={'container.networkMode'}
                                    label={getString('pluginsPanel.run.networkMode')}
                                    style={{ width: '100%' }}
                                    key={'container.networkMode'}
                                  />
                                  <FormInput.Toggle
                                    name={'container.privileged'}
                                    label={getString('pluginsPanel.run.privileged')}
                                    style={{ width: '100%' }}
                                    key={'container.privileged'}
                                  />
                                  <FormInput.Text
                                    name={'container.user'}
                                    label={getString('user')}
                                    style={{ width: '100%' }}
                                    key={'container.user'}
                                  />
                                  <Accordion activeId="">
                                    <Accordion.Panel
                                      id="container.credentials"
                                      summary={getString('pluginsPanel.run.credentials')}
                                      details={
                                        <Layout.Vertical className={css.indent}>
                                          <FormInput.Text
                                            name={'container.credentials.username'}
                                            label={getString('pluginsPanel.run.username')}
                                            style={{ width: '100%' }}
                                            key={'container.credentials.username'}
                                          />
                                          <FormInput.Text
                                            name={'container.credentials.password'}
                                            label={getString('pluginsPanel.run.password')}
                                            style={{ width: '100%' }}
                                            key={'container.credentials.password'}
                                          />
                                        </Layout.Vertical>
                                      }
                                    />
                                  </Accordion>
                                </Layout.Vertical>
                              }
                            />
                            <Accordion.Panel
                              id="mount"
                              summary="Mount"
                              details={
                                <Layout.Vertical className={css.indent}>
                                  <FormInput.Text
                                    name={'mount.name'}
                                    label={getString('name')}
                                    style={{ width: '100%' }}
                                    key={'mount.name'}
                                  />
                                  <FormInput.Text
                                    name={'mount.path'}
                                    label={getString('pluginsPanel.run.path')}
                                    style={{ width: '100%' }}
                                    key={'mount.path'}
                                  />
                                </Layout.Vertical>
                              }
                            />
                          </Accordion>
                        </Layout.Vertical>
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
                      <Button variation={ButtonVariation.PRIMARY} text={getString('addLabel')} type="submit" />
                    </Container>
                  </Layout.Vertical>
                </FormikForm>
              )
            }}
          </Formik>
        </Container>
      </Layout.Vertical>
    )
  }, [plugin, category])

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
  }, [loading, plugins, panelView, category])

  return (
    <Layout.Vertical height="100%">
      {panelView === PluginPanelView.Category ? (
        <Container padding={{ top: 'medium', bottom: 'medium', left: 'medium' }}>
          <Text font={{ variation: FontVariation.H5 }}>{getString('step.select')}</Text>
        </Container>
      ) : (
        <></>
      )}
      {renderPluginsPanel()}
    </Layout.Vertical>
  )
}
