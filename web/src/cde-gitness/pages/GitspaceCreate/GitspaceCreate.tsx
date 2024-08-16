import React from 'react'
import { Breadcrumbs, Card, Container, Heading, Layout, Page, Text } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import { useStrings } from 'framework/strings'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { useAppContext } from 'AppContext'
import { GitnessCreateGitspace } from './GitnessCreateGitspace'
import { CDECreateGitspace } from './CDECreateGitspace'
import css from './GitspaceCreate.module.scss'

const GitspaceCreate = () => {
  const { getString } = useStrings()
  const space = useGetSpaceParam()
  const { standalone, routes } = useAppContext()

  return (
    <>
      <Page.Header
        title={getString('cde.createGitspace')}
        breadcrumbs={
          <Breadcrumbs
            links={[
              { url: routes.toCDEGitspaces({ space }), label: getString('cde.gitspaces') },
              { url: routes.toCDEGitspacesCreate({ space }), label: getString('cde.createGitspace') }
            ]}
          />
        }
      />
      <Page.Body className={css.main}>
        <Container className={css.titleContainer}>
          <Layout.Vertical spacing="small" margin={{ bottom: 'medium' }}>
            <Heading font={{ weight: 'bold' }} color={Color.BLACK} level={2}>
              {getString('cde.createGitspace')}
            </Heading>
            <Text font={{ size: 'medium' }}>{getString('cde.create.subtext')}</Text>
          </Layout.Vertical>
        </Container>
        <Card className={css.cardMain}>
          <Container className={css.subContainers}>
            {standalone ? <GitnessCreateGitspace /> : <CDECreateGitspace />}
          </Container>
        </Card>
      </Page.Body>
    </>
  )
}

export default GitspaceCreate
