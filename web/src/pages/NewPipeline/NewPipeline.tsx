import React, { useState } from 'react'
import { useMutate } from 'restful-react'
import { Link, useParams } from 'react-router-dom'
import { Container, PageHeader, PageBody, Button, Layout, ButtonVariation, Text } from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { Color } from '@harnessio/design-system'
import type { OpenapiCommitFilesRequest, RepoCommitFilesResponse } from 'services/code'
import { useStrings } from 'framework/strings'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { SourceCodeEditor } from 'components/SourceCodeEditor/SourceCodeEditor'
import { useAppContext } from 'AppContext'
import type { CODEProps } from 'RouteDefinitions'

import css from './NewPipeline.module.scss'

const NewPipeline = (): JSX.Element => {
  const { routes } = useAppContext()
  const { getString } = useStrings()
  const { pipeline } = useParams<CODEProps>()
  const space = useGetSpaceParam()
  const [pipelineAsYAML, setPipelineAsYaml] = useState<string>('')

  const { mutate, loading } = useMutate<RepoCommitFilesResponse>({
    verb: 'POST',
    path: `/api/v1/repos/test-space/vb-repo/+/commits`
  })

  const handleSaveAndRun = (): void => {
    const data: OpenapiCommitFilesRequest = {
      actions: [{ action: 'CREATE', path: `sample_${new Date().getTime()}.txt`, payload: pipelineAsYAML }],
      branch: 'main',
      new_branch: '',
      title: `Create pipeline ${pipeline}`,
      message: ''
    }

    mutate(data)
      .then(response => console.log(response))
      .catch(error => console.log(error))
  }

  return (
    <Container className={css.main}>
      <PageHeader
        title={getString('pipelines.editPipeline', { pipeline })}
        breadcrumbs={
          <Container className={css.header}>
            <Layout.Horizontal spacing="small" className={css.breadcrumb}>
              <Link to={routes.toCODEPipelines({ space })}>{getString('pageTitle.pipelines')}</Link>
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
        <Container className={css.editorContainer}>
          <SourceCodeEditor
            language={'yaml'}
            source={
              'stages:\n- type: ci\n  spec:\n    steps:\n    - type: script\n      spec:\n        run: echo hello world'
            }
            onChange={(value: string) => setPipelineAsYaml(value)}
            autoHeight
          />
        </Container>
      </PageBody>
    </Container>
  )
}

export default NewPipeline
