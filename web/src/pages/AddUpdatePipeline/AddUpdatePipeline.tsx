import React, { useEffect, useMemo, useState } from 'react'
import { useGet, useMutate } from 'restful-react'
import { Link, useParams } from 'react-router-dom'
import { isEmpty, isUndefined } from 'lodash'
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
import pipelineSchema from './schema/pipeline-schema.json'

import css from './AddUpdatePipeline.module.scss'

const starterPipelineAsString =
  'version: 1\nstages:\n- type: ci\n  spec:\n    steps:\n    - type: script\n      spec:\n        run: echo hello world'

const AddUpdatePipeline = (): JSX.Element => {
  const { routes } = useAppContext()
  const { getString } = useStrings()
  const { pipeline } = useParams<CODEProps>()
  const { repoMetadata } = useGetRepositoryMetadata()
  const { showError } = useToaster()
  const [pipelineAsYAML, setPipelineAsYaml] = useState<string>('')
  const { openModal: openRunPipelineModal } = useRunPipelineModal()
  const repoPath = useMemo(() => repoMetadata?.path || '', [repoMetadata])
  const [isExistingPipeline, setIsExistingPipeline] = useState<boolean>(false)

  const { mutate, loading } = useMutate<RepoCommitFilesResponse>({
    verb: 'POST',
    path: `/api/v1/repos/${repoPath}/+/commits`
  })

  // Fetch pipeline metadata to fetch pipeline YAML file content
  const { data: pipelineData, loading: fetchingPipeline } = useGet<TypesPipeline>({
    path: `/api/v1/repos/${repoPath}/+/pipelines/${pipeline}`,
    lazy: !repoMetadata
  })

  const { data: pipelineYAMLFileContent, loading: resourceLoading } = useGetResourceContent({
    repoMetadata,
    gitRef: pipelineData?.default_branch || '',
    resourcePath: pipelineData?.config_path || ''
  })

  useEffect(() => {
    if (!resourceLoading) {
      setIsExistingPipeline(!isEmpty(pipelineYAMLFileContent) && !isUndefined(pipelineYAMLFileContent.content))
    }
  }, [pipelineYAMLFileContent, resourceLoading])

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
          if (repoMetadata && pipeline) {
            openRunPipelineModal({ repoMetadata, pipeline })
          }
        })
        .catch(error => {
          showError(getErrorMessage(error), 0, 'pipelines.failedToSavePipeline')
        })
    } catch (exception) {
      showError(getErrorMessage(exception), 0, 'pipelines.failedToCreatePipeline')
    }
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
                disabled={loading}
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
                schema={pipelineSchema}
                source={
                  decodeGitContent((pipelineYAMLFileContent?.content as RepoFileContent)?.data) ||
                  starterPipelineAsString
                }
                onChange={(value: string) => setPipelineAsYaml(value)}
              />
            </Container>
            <Container className={css.pluginsContainer}>
              <PluginsPanel />
            </Container>
          </Layout.Horizontal>
        </PageBody>
      </Container>
    </>
  )
}

export default AddUpdatePipeline
