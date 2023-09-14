import React, { useCallback, useEffect, useMemo, useState } from 'react'
import { useGet, useMutate } from 'restful-react'
import { Link, useParams } from 'react-router-dom'
import { get, isEmpty, isUndefined, set } from 'lodash-es'
import { stringify } from 'yaml'
import { Menu, PopoverPosition } from '@blueprintjs/core'
import {
  Container,
  PageHeader,
  PageBody,
  Layout,
  ButtonVariation,
  Text,
  useToaster,
  SplitButton,
  Button
} from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { Color, FontVariation } from '@harnessio/design-system'
import type { OpenapiCommitFilesRequest, RepoCommitFilesResponse, RepoFileContent, TypesPipeline } from 'services/code'
import { useStrings } from 'framework/strings'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useGetResourceContent } from 'hooks/useGetResourceContent'
import MonacoSourceCodeEditor from 'components/SourceCodeEditor/MonacoSourceCodeEditor'
import { PluginsPanel } from 'components/PluginsPanel/PluginsPanel'
import useRunPipelineModal from 'components/RunPipelineModal/RunPipelineModal'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { useAppContext } from 'AppContext'
import type { CODEProps } from 'RouteDefinitions'
import { getErrorMessage } from 'utils/Utils'
import { decodeGitContent } from 'utils/GitUtils'
import pipelineSchemaV1 from './schema/pipeline-schema-v1.json'
import pipelineSchemaV0 from './schema/pipeline-schema-v0.json'
import { DRONE_CONFIG_YAML_FILE_SUFFIXES, YamlVersion } from './Constants'

import css from './AddUpdatePipeline.module.scss'

const StarterPipelineV1: Record<string, any> = {
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
              type: 'script',
              spec: {
                image: 'golang',
                run: 'echo "hello world"'
              }
            }
          ]
        }
      }
    ]
  }
}

const StarterPipelineV0: Record<string, any> = {
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
  const [pipelineAsObj, setPipelineAsObj] = useState<Record<string, any>>({})
  const [pipelineAsYAML, setPipelineAsYaml] = useState<string>('')
  const { openModal: openRunPipelineModal } = useRunPipelineModal()
  const repoPath = useMemo(() => repoMetadata?.path || '', [repoMetadata])
  const [isExistingPipeline, setIsExistingPipeline] = useState<boolean>(false)
  const [isDirty, setIsDirty] = useState<boolean>(false)

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

  const { mutate, loading } = useMutate<RepoCommitFilesResponse>({
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
    gitRef: pipelineData?.default_branch || '',
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
      setPipelineAsYaml(originalPipelineYAMLFileContent)
    } else {
      // load with starter pipeline
      try {
        setPipelineAsYaml(stringify(yamlVersion === YamlVersion.V1 ? StarterPipelineV1 : StarterPipelineV0))
      } catch (ex) {
        // ignore exception
      }
    }
  }, [yamlVersion, isExistingPipeline, originalPipelineYAMLFileContent, pipelineAsObj])

  // find if editor content was modified
  useEffect(() => {
    setIsDirty(originalPipelineYAMLFileContent !== pipelineAsYAML)
  }, [originalPipelineYAMLFileContent, pipelineAsYAML])

  // set initial CTA title
  useEffect(() => {
    setSelectedOption(isDirty ? pipelineSaveAndRunOption : pipelineRunOption)
  }, [isDirty])

  const handleSaveAndRun = (option: PipelineSaveAndRunOption): void => {
    if ([PipelineSaveAndRunAction.SAVE_AND_RUN, PipelineSaveAndRunAction.SAVE].includes(option?.action)) {
      try {
        const data: OpenapiCommitFilesRequest = {
          actions: [
            {
              action: isExistingPipeline ? 'UPDATE' : 'CREATE',
              path: pipelineData?.config_path,
              payload: pipelineAsYAML,
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

  const updatePipeline = (payload: Record<string, any>): Record<string, any> => {
    const pipelineAsObjClone = { ...pipelineAsObj }
    const stepInsertPath = yamlVersion === YamlVersion.V1 ? 'spec.stages.0.spec.steps' : 'steps'
    let existingSteps: [unknown] = get(pipelineAsObjClone, stepInsertPath, [])
    if (existingSteps.length > 0) {
      existingSteps.push(payload)
    } else {
      existingSteps = [payload]
    }
    set(pipelineAsObjClone, stepInsertPath, existingSteps)
    return pipelineAsObjClone
  }

  const addUpdatePluginToPipelineYAML = (_isUpdate: boolean, pluginFormData: Record<string, any>): void => {
    try {
      const updatedPipelineAsObj = updatePipeline(pluginFormData)
      setPipelineAsObj(updatedPipelineAsObj)
      setPipelineAsYaml(stringify(updatedPipelineAsObj))
    } catch (ex) {
      // ignore exception
    }
  }

  const renderCTA = useCallback(() => {
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
            text={selectedOption?.title}
            disabled={loading || !isDirty}
            variation={ButtonVariation.PRIMARY}
            popoverProps={{
              interactionKind: 'click',
              usePortal: true,
              position: PopoverPosition.BOTTOM_RIGHT,
              transitionDuration: 1000
            }}
            intent="primary"
            onClick={() => handleSaveAndRun(selectedOption)}>
            {pipelineSaveAndRunOptions.map(option => {
              return (
                <Menu.Item
                  key={option.title}
                  text={
                    <Text color={Color.BLACK} font={{ variation: FontVariation.SMALL_BOLD }}>
                      {option.title}
                    </Text>
                  }
                  onClick={() => {
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
  }, [loading, fetchingPipeline, isDirty, repoMetadata, pipeline, selectedOption, isExistingPipeline, pipelineAsYAML])

  return (
    <>
      <Container className={css.main}>
        <PageHeader
          title={getString('pipelines.editPipeline', { pipeline })}
          breadcrumbs={
            <Container className={css.header}>
              <Layout.Horizontal spacing="small" className={css.breadcrumb}>
                <Link to={routes.toCODEPipelines({ repoPath })}>{getString('pageTitle.pipelines')}</Link>
                <Icon name="main-chevron-right" size={8} color={Color.GREY_500} />
                <Text font={{ size: 'small' }}>{pipeline}</Text>
              </Layout.Horizontal>
            </Container>
          }
          content={<Layout.Horizontal flex={{ justifyContent: 'space-between' }}>{renderCTA()}</Layout.Horizontal>}
        />
        <PageBody>
          <LoadingSpinner visible={fetchingPipeline || fetchingPipelineYAMLFileContent} />
          <Layout.Horizontal className={css.container}>
            <Container className={css.editorContainer}>
              <MonacoSourceCodeEditor
                language={'yaml'}
                schema={yamlVersion === YamlVersion.V1 ? pipelineSchemaV1 : pipelineSchemaV0}
                source={pipelineAsYAML}
                onChange={(value: string) => setPipelineAsYaml(value)}
              />
            </Container>
            <Container className={css.pluginsContainer}>
              <PluginsPanel onPluginAddUpdate={addUpdatePluginToPipelineYAML} version={yamlVersion} />
            </Container>
          </Layout.Horizontal>
        </PageBody>
      </Container>
    </>
  )
}

export default AddUpdatePipeline
