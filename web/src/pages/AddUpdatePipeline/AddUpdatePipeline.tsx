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
import { useGet, useMutate } from 'restful-react'
import { useParams } from 'react-router-dom'
import { get, isEmpty, isUndefined, set } from 'lodash-es'
import { parse, stringify } from 'yaml'
import cx from 'classnames'
import { Menu, PopoverPosition } from '@blueprintjs/core'
import { Container, PageBody, Layout, ButtonVariation, Text, useToaster, SplitButton, Button } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import type { OpenapiCommitFilesRequest, TypesListCommitResponse, RepoFileContent, TypesPipeline } from 'services/code'
import { useStrings } from 'framework/strings'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useGetResourceContent } from 'hooks/useGetResourceContent'
import MonacoSourceCodeEditor from 'components/SourceCodeEditor/MonacoSourceCodeEditor'
import { PluginForm, PluginsPanel } from 'components/PluginsPanel/PluginsPanel'
import useRunPipelineModal from 'components/RunPipelineModal/RunPipelineModal'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { useAppContext } from 'AppContext'
import type { CODEProps } from 'RouteDefinitions'
import { getErrorMessage } from 'utils/Utils'
import { normalizeGitRef, decodeGitContent } from 'utils/GitUtils'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import pipelineSchemaV1 from './schema/pipeline-schema-v1.json'
import pipelineSchemaV0 from './schema/pipeline-schema-v0.json'
import { DRONE_CONFIG_YAML_FILE_SUFFIXES, YamlVersion } from './Constants'

import css from './AddUpdatePipeline.module.scss'

const StarterPipelineV1: Record<string, unknown> = {
  version: 1,
  kind: 'pipeline',
  spec: {
    stages: [
      {
        name: 'build',
        type: 'ci',
        spec: {
          steps: [
            {
              name: 'build',
              type: 'run',
              spec: {
                container: 'alpine',
                script: 'echo "hello world"'
              }
            }
          ]
        }
      }
    ]
  }
}

const StarterPipelineV0: Record<string, unknown> = {
  kind: 'pipeline',
  type: 'docker',
  name: 'default',
  steps: [
    {
      name: 'test',
      image: 'alpine',
      commands: ['echo hello world']
    }
  ]
}

enum PipelineSaveAndRunAction {
  SAVE,
  RUN,
  SAVE_AND_RUN
}

interface PipelineSaveAndRunOption {
  title: string
  action: PipelineSaveAndRunAction
}

const AddUpdatePipeline = (): JSX.Element => {
  const { routes } = useAppContext()
  const { getString } = useStrings()
  const { pipeline } = useParams<CODEProps>()
  const { repoMetadata } = useGetRepositoryMetadata()
  const { showError, showSuccess, clear: clearToaster } = useToaster()
  const [yamlVersion, setYAMLVersion] = useState<YamlVersion>()
  const [pipelineYAML, setPipelineYAML] = useState<string>('')
  const { openModal: openRunPipelineModal } = useRunPipelineModal()
  const repoPath = useMemo(() => repoMetadata?.path || '', [repoMetadata])
  const [isExistingPipeline, setIsExistingPipeline] = useState<boolean>(false)
  const [isDirty, setIsDirty] = useState<boolean>(false)
  const [generatingPipeline, setGeneratingPipeline] = useState<boolean>(false)
  const editorYAMLRef = useRef<string>('')

  const pipelineSaveOption: PipelineSaveAndRunOption = {
    title: getString('save'),
    action: PipelineSaveAndRunAction.SAVE
  }

  const pipelineRunOption: PipelineSaveAndRunOption = {
    title: getString('run'),
    action: PipelineSaveAndRunAction.RUN
  }

  const pipelineSaveAndRunOption: PipelineSaveAndRunOption = {
    title: getString('pipelines.saveAndRun'),
    action: PipelineSaveAndRunAction.SAVE_AND_RUN
  }

  const pipelineSaveAndRunOptions: PipelineSaveAndRunOption[] = [pipelineSaveAndRunOption, pipelineSaveOption]

  const [selectedOption, setSelectedOption] = useState<PipelineSaveAndRunOption>()

  const { mutate, loading } = useMutate<TypesListCommitResponse>({
    verb: 'POST',
    path: `/api/v1/repos/${repoPath}/+/commits`
  })

  // Fetch pipeline metadata to fetch pipeline YAML file content
  const { data: pipelineData, loading: fetchingPipeline } = useGet<TypesPipeline>({
    path: `/api/v1/repos/${repoPath}/+/pipelines/${pipeline}`,
    lazy: !repoMetadata
  })

  const {
    data: pipelineYAMLFileContent,
    loading: fetchingPipelineYAMLFileContent,
    refetch: fetchPipelineYAMLFileContent
  } = useGetResourceContent({
    repoMetadata,
    gitRef: normalizeGitRef(pipelineData?.default_branch || '') as string,
    resourcePath: pipelineData?.config_path || ''
  })

  const originalPipelineYAMLFileContent = useMemo(
    (): string => decodeGitContent((pipelineYAMLFileContent?.content as RepoFileContent)?.data),
    [pipelineYAMLFileContent?.content]
  )

  // set YAML version for Pipeline setup
  useEffect(() => {
    setYAMLVersion(
      DRONE_CONFIG_YAML_FILE_SUFFIXES.find((suffix: string) => pipelineData?.config_path?.endsWith(suffix))
        ? YamlVersion.V0
        : YamlVersion.V1
    )
  }, [pipelineData])

  // check if file already exists and has some content
  useEffect(() => {
    setIsExistingPipeline(!isEmpty(originalPipelineYAMLFileContent) && !isUndefined(originalPipelineYAMLFileContent))
  }, [originalPipelineYAMLFileContent])

  // load initial content on the editor
  useEffect(() => {
    if (isExistingPipeline) {
      setPipelineYAML(originalPipelineYAMLFileContent)
    } else {
      // load with starter pipeline
      try {
        setPipelineYAML(stringify(yamlVersion === YamlVersion.V1 ? StarterPipelineV1 : StarterPipelineV0))
      } catch (ex) {
        // ignore exception
      }
    }
  }, [yamlVersion, isExistingPipeline, originalPipelineYAMLFileContent])

  // find if editor content was modified
  useEffect(() => {
    setIsDirty(originalPipelineYAMLFileContent !== pipelineYAML)
  }, [originalPipelineYAMLFileContent, pipelineYAML])

  // set initial CTA title
  useEffect(() => {
    setSelectedOption(isDirty ? pipelineSaveAndRunOption : pipelineRunOption)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isDirty])

  useEffect(() => {
    editorYAMLRef.current = pipelineYAML
  }, [pipelineYAML])

  const handleSaveAndRun = (option: PipelineSaveAndRunOption): void => {
    if ([PipelineSaveAndRunAction.SAVE_AND_RUN, PipelineSaveAndRunAction.SAVE].includes(option?.action)) {
      try {
        const data: OpenapiCommitFilesRequest = {
          actions: [
            {
              action: isExistingPipeline ? 'UPDATE' : 'CREATE',
              path: pipelineData?.config_path,
              payload: pipelineYAML,
              sha: isExistingPipeline ? pipelineYAMLFileContent?.sha : ''
            }
          ],
          branch: pipelineData?.default_branch || '',
          title: `${isExistingPipeline ? getString('updated') : getString('created')} pipeline ${pipeline}`,
          message: ''
        }

        mutate(data)
          .then(() => {
            fetchPipelineYAMLFileContent()
            clearToaster()
            showSuccess(getString(isExistingPipeline ? 'pipelines.updated' : 'pipelines.created'))
            if (option?.action === PipelineSaveAndRunAction.SAVE_AND_RUN && repoMetadata && pipeline) {
              openRunPipelineModal({ repoMetadata, pipeline })
            }
            setSelectedOption(pipelineRunOption)
          })
          .catch(error => {
            showError(getErrorMessage(error), 0, 'pipelines.failedToSavePipeline')
          })
      } catch (exception) {
        showError(getErrorMessage(exception), 0, 'pipelines.failedToSavePipeline')
      }
    }
  }

  const updatePipelineWithPluginData = (
    existingPipeline: Record<string, unknown>,
    payload: Record<string, unknown>
  ): Record<string, unknown> => {
    const pipelineAsObjClone = { ...existingPipeline }
    if (Object.keys(pipelineAsObjClone).length > 0) {
      const stepInsertPath = 'spec.stages.0.spec.steps'
      let existingSteps = (get(pipelineAsObjClone, stepInsertPath) as unknown[]) || []
      if (existingSteps.length > 0) {
        existingSteps.push(payload)
      } else {
        existingSteps = [payload]
      }
      set(pipelineAsObjClone, stepInsertPath, existingSteps)
      return pipelineAsObjClone
    }
    return existingPipeline
  }

  const handlePluginAddUpdateToPipeline = useCallback(
    ({
      pluginFormData,
      existingYAML
    }: {
      isUpdate: boolean
      pluginFormData: PluginForm
      existingYAML: string
    }): void => {
      try {
        const pipelineAsObj = parse(existingYAML)
        const updatedPipelineAsObj = updatePipelineWithPluginData(pipelineAsObj, pluginFormData)
        if (Object.keys(updatedPipelineAsObj).length > 0) {
          // avoid setting to empty pipeline in case pipeline update with plugin data fails
          const updatedPipelineAsYAML = stringify(updatedPipelineAsObj)
          setPipelineYAML(updatedPipelineAsYAML)
          editorYAMLRef.current = updatedPipelineAsYAML
        }
      } catch (ex) {
        // ignore exception
      }
    },
    []
  )

  const handleGeneratePipeline = useCallback(async (): Promise<void> => {
    try {
      const response = await fetch(`/api/v1/repos/${repoPath}/+/pipelines/generate`)
      if (response.ok && response.status === 200) {
        const responsePipelineAsYAML = await response.text()
        if (responsePipelineAsYAML) {
          setPipelineYAML(responsePipelineAsYAML)
        }
      }
      setGeneratingPipeline(false)
    } catch (exception) {
      showError(getErrorMessage(exception), 0, getString('pipelines.failedToGenerate'))
      setGeneratingPipeline(false)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [repoPath])

  const renderCTA = useCallback(() => {
    /* Do not render CTA till pipeline existence info is obtained */
    if (fetchingPipeline || !pipelineData) {
      return <></>
    }
    switch (selectedOption?.action) {
      case PipelineSaveAndRunAction.RUN:
        return (
          <Button
            variation={ButtonVariation.PRIMARY}
            text={getString('run')}
            onClick={() => {
              if (repoMetadata && pipeline) {
                openRunPipelineModal({ repoMetadata, pipeline })
              }
            }}
          />
        )
      case PipelineSaveAndRunAction.SAVE:
      case PipelineSaveAndRunAction.SAVE_AND_RUN:
        return isExistingPipeline ? (
          <Button
            variation={ButtonVariation.PRIMARY}
            text={getString('save')}
            onClick={() => {
              handleSaveAndRun(pipelineSaveOption)
            }}
            disabled={loading || !isDirty}
          />
        ) : (
          <SplitButton
            text={
              <Text color={Color.WHITE} font={{ variation: FontVariation.BODY2_SEMI, weight: 'bold' }}>
                {pipelineSaveAndRunOptions[0].title}
              </Text>
            }
            disabled={loading || !isDirty}
            variation={ButtonVariation.PRIMARY}
            popoverProps={{
              interactionKind: 'click',
              usePortal: true,
              position: PopoverPosition.BOTTOM_RIGHT,
              popoverClassName: css.popover
            }}
            intent="primary"
            onClick={() => handleSaveAndRun(pipelineSaveAndRunOptions[0])}>
            {[pipelineSaveAndRunOptions[1]].map(option => {
              return (
                <Menu.Item
                  className={css.menuItem}
                  key={option.title}
                  text={<Text font={{ variation: FontVariation.BODY2 }}>{option.title}</Text>}
                  onClick={() => {
                    handleSaveAndRun(option)
                    setSelectedOption(option)
                  }}
                />
              )
            })}
          </SplitButton>
        )
      default:
        return <></>
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [
    loading,
    fetchingPipeline,
    fetchingPipelineYAMLFileContent,
    isDirty,
    repoMetadata,
    pipeline,
    selectedOption,
    isExistingPipeline,
    pipelineYAML,
    pipelineData
  ])

  if (fetchingPipeline || fetchingPipelineYAMLFileContent) {
    return <LoadingSpinner visible={true} />
  }

  return (
    <>
      <Container className={css.main}>
        <RepositoryPageHeader
          repoMetadata={repoMetadata}
          title={pipeline as string}
          dataTooltipId="repositoryExecutions"
          extraBreadcrumbLinks={
            repoMetadata && [
              {
                label: getString('pageTitle.pipelines'),
                url: routes.toCODEPipelines({ repoPath: repoMetadata.path as string })
              },
              ...(pipeline
                ? [
                    {
                      label: pipeline,
                      url: ''
                    }
                  ]
                : [])
            ]
          }
          content={<Layout.Horizontal flex={{ justifyContent: 'space-between' }}>{renderCTA()}</Layout.Horizontal>}
        />
        <PageBody>
          <Layout.Vertical>
            {!isExistingPipeline && yamlVersion === YamlVersion.V1 && (
              <Layout.Horizontal
                padding={{ left: 'medium', bottom: 'medium', top: 'medium' }}
                className={css.generateHeader}
                spacing="large"
                flex={{ justifyContent: 'flex-start' }}>
                <Button
                  text={getString('generate')}
                  variation={ButtonVariation.SECONDARY}
                  className={css.generate}
                  onClick={handleGeneratePipeline}
                  disabled={generatingPipeline}
                />
                <Text font={{ weight: 'bold' }}>{getString('generateHelptext')}</Text>
              </Layout.Horizontal>
            )}
            <Layout.Horizontal className={css.layout}>
              <Container
                className={cx(css.editorContainer, {
                  [css.extendedHeight]: isExistingPipeline || yamlVersion === YamlVersion.V0
                })}>
                <MonacoSourceCodeEditor
                  language={'yaml'}
                  schema={yamlVersion === YamlVersion.V1 ? pipelineSchemaV1 : pipelineSchemaV0}
                  source={pipelineYAML}
                  onChange={(value: string) => setPipelineYAML(value)}
                />
              </Container>
              {yamlVersion === YamlVersion.V1 && (
                <Container className={cx(css.pluginsContainer, { [css.extendedHeight]: isExistingPipeline })}>
                  <PluginsPanel
                    onPluginAddUpdate={(isUpdate: boolean, pluginFormData: PluginForm) =>
                      handlePluginAddUpdateToPipeline({
                        isUpdate,
                        pluginFormData,
                        existingYAML: editorYAMLRef.current || pipelineYAML
                      })
                    }
                  />
                </Container>
              )}
            </Layout.Horizontal>
          </Layout.Vertical>
        </PageBody>
      </Container>
    </>
  )
}

export default AddUpdatePipeline
