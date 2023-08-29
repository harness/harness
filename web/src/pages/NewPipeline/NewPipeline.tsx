import React, { useCallback } from 'react'
import { useMutate } from 'restful-react'
import { Container, PageHeader, PageBody } from '@harnessio/uicore'
import { Button, Layout, ButtonVariation } from '@harnessio/uicore'
import type { TypesPipeline, OpenapiCreatePipelineRequest } from 'services/code'
import { useStrings } from 'framework/strings'
import { SourceCodeEditor } from 'components/SourceCodeEditor/SourceCodeEditor'

import css from './NewPipeline.module.scss'

const NewPipeline = (): JSX.Element => {
  const { mutate: savePipeline } = useMutate<TypesPipeline>({
    verb: 'POST',
    path: `/api/v1/pipelines`
  })

  const handleSavePipeline = useCallback(async (): Promise<void> => {
    const payload: OpenapiCreatePipelineRequest = {
      config_path: 'config_path_4',
      default_branch: 'main',
      space_ref: 'test-space',
      repo_ref: 'test-space/vb-repo',
      repo_type: 'GITNESS',
      uid: 'pipeline_uid_4'
    }
    const response = await savePipeline(payload)
    console.log(response)
  }, [])

  const { getString } = useStrings()
  return (
    <Container className={css.main}>
      <PageHeader
        title={getString('pipelines.newPipelineButton')}
        content={
          <Layout.Horizontal flex={{ justifyContent: 'space-between' }}>
            <Button variation={ButtonVariation.PRIMARY} text={getString('save')} onClick={handleSavePipeline} />
          </Layout.Horizontal>
        }></PageHeader>
      <PageBody>
        <Container className={css.editorContainer}>
          <SourceCodeEditor language={'yaml'} source={''} onChange={() => {}} autoHeight />
        </Container>
      </PageBody>
    </Container>
  )
}

export default NewPipeline
