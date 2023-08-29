import React from 'react'
import { Container, PageHeader, PageBody } from '@harnessio/uicore'
import { Button, Layout, ButtonVariation } from '@harnessio/uicore'
import { useStrings } from 'framework/strings'
import { SourceCodeEditor } from 'components/SourceCodeEditor/SourceCodeEditor'

import css from './NewPipeline.module.scss'

const NewPipeline = (): JSX.Element => {
  const { getString } = useStrings()
  return (
    <Container className={css.main}>
      <PageHeader
        title={getString('pipelines.newPipelineButton')}
        content={
          <Layout.Horizontal flex={{ justifyContent: 'space-between' }}>
            <Button variation={ButtonVariation.PRIMARY} text={getString('save')} onClick={() => {}} />
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
