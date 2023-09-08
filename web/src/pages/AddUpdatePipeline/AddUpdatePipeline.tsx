import React, { useEffect, useMemo, useState } from 'react'
import { useGet, useMutate } from 'restful-react'
import { Link, useParams } from 'react-router-dom'
import { get, isEmpty, isUndefined, set } from 'lodash'
import { stringify } from 'yaml'
import { Container, PageHeader, PageBody, Button, Layout, ButtonVariation, Text, useToaster } from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { Color } from '@harnessio/design-system'
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
import { YamlVersion } from './Constants'

import css from './AddUpdatePipeline.module.scss'

const StarterPipelineV1: Record<string, any> = {
  version: 1,
  stages: [
    {
      type: 'ci',
      spec: {
        steps: [
          {
            type: 'script',
            spec: {
              run: 'echo hello world'
            }
          }
        ]
      }
    }
  ]
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

const AddUpdatePipeline = (): JSX.Element => {
  const version = YamlVersion.V0
  const { routes } = useAppContext()
  const { getString } = useStrings()
  const { pipeline } = useParams<CODEProps>()
  const { repoMetadata } = useGetRepositoryMetadata()
  const { showError, showSuccess } = useToaster()
  const [pipelineAsObj, setPipelineAsObj] = useState<Record<string, any>>(
    version === YamlVersion.V0 ? StarterPipelineV0 : StarterPipelineV1
  )
  const [pipelineAsYAML, setPipelineAsYaml] = useState<string>('')
  const { openModal: openRunPipelineModal } = useRunPipelineModal()
  const repoPath = useMemo(() => repoMetadata?.path || '', [repoMetadata])
  const [isExistingPipeline, setIsExistingPipeline] = useState<boolean>(false)
  const [isDirty, setIsDirty] = useState<boolean>(false)

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
    loading: resourceLoading,
    refetch: fetchPipelineYAMLFileContent
  } = useGetResourceContent({
    repoMetadata,
    gitRef: pipelineData?.default_branch || '',
    resourcePath: pipelineData?.config_path || ''
  })

  // check if file exists and has some content
  useEffect(() => {
    if (!resourceLoading) {
      setIsExistingPipeline(!isEmpty(pipelineYAMLFileContent) && !isUndefined(pipelineYAMLFileContent.content))
    }
  }, [pipelineYAMLFileContent, resourceLoading])

  // to load initial content on the editor
  useEffect(() => {
    if (isExistingPipeline) {
      setPipelineAsYaml(decodeGitContent((pipelineYAMLFileContent?.content as RepoFileContent)?.data))
    } else {
      try {
        setPipelineAsYaml(stringify(pipelineAsObj))
      } catch (ex) {}
    }
  }, [isExistingPipeline, pipelineYAMLFileContent])

  // find if editor content was modified
  useEffect(() => {
    if (isExistingPipeline) {
      const originalContent = decodeGitContent((pipelineYAMLFileContent?.content as RepoFileContent)?.data)
      setIsDirty(originalContent !== pipelineAsYAML)
    } else {
      setIsDirty(true)
    }
  }, [isExistingPipeline, pipelineAsYAML, pipelineYAMLFileContent])

  const handleSaveAndRun = (): void => {
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
        branch: repoMetadata?.default_branch,
        new_branch: '',
        title: `${isExistingPipeline ? getString('updated') : getString('created')} pipeline ${pipeline}`,
        message: ''
      }

      mutate(data)
        .then(() => {
          fetchPipelineYAMLFileContent()
          showSuccess(getString(isExistingPipeline ? 'pipelines.updated' : 'pipelines.created'))
          if (repoMetadata && pipeline) {
            openRunPipelineModal({ repoMetadata, pipeline })
          }
        })
        .catch(error => {
          showError(getErrorMessage(error), 0, 'pipelines.failedToSavePipeline')
        })
    } catch (exception) {
      showError(getErrorMessage(exception), 0, 'pipelines.failedToSavePipeline')
    }
  }

  const updatePipeline = (payload: Record<string, any>): Record<string, any> => {
    const pipelineAsObjClone = { ...pipelineAsObj }
    const stepInsertPath = version === YamlVersion.V0 ? 'steps' : 'stages.0.spec.steps'
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
    } catch (ex) {}
  }

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
          content={
            <Layout.Horizontal flex={{ justifyContent: 'space-between' }}>
              <Button
                variation={ButtonVariation.PRIMARY}
                text={getString('pipelines.saveAndRun')}
                onClick={handleSaveAndRun}
                disabled={loading || !isDirty}
              />
            </Layout.Horizontal>
          }
        />
        <PageBody>
          <LoadingSpinner visible={fetchingPipeline} />
          <Layout.Horizontal>
            <Container className={css.editorContainer}>
              <MonacoSourceCodeEditor
                language={'yaml'}
                schema={version === YamlVersion.V0 ? pipelineSchemaV0 : pipelineSchemaV1}
                source={pipelineAsYAML}
                onChange={(value: string) => setPipelineAsYaml(value)}
              />
            </Container>
            <Container className={css.pluginsContainer}>
              <PluginsPanel onPluginAddUpdate={addUpdatePluginToPipelineYAML} />
            </Container>
          </Layout.Horizontal>
        </PageBody>
      </Container>
    </>
  )
}

export default AddUpdatePipeline
