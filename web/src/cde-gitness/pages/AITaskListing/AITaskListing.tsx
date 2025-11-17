import React from 'react'
import { Breadcrumbs, Card, Container, Heading, Layout, Page, Text } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
// import { useStrings } from 'framework/strings'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { useAppContext } from 'AppContext'
import { useGetCDEAPIParams } from 'cde-gitness/hooks/useGetCDEAPIParams'

const AITaskListing = () => {
  // const { getString } = useStrings()
  const space = useGetSpaceParam()
  const { standalone, accountInfo, routes } = useAppContext()
  const { orgIdentifier, projectIdentifier, accountIdentifier } = useGetCDEAPIParams()

  const getBreadcrumbLinks = () => {
    if (standalone) {
      return [{ url: routes.toCDEAITasks({ space }), label: 'Tasks' }]
    }

    return [
      {
        url: `/account/${accountIdentifier}/module/cde`,
        label: `Account: ${accountInfo?.name || accountIdentifier}`
      },
      {
        url: `/account/${accountIdentifier}/module/cde/orgs/${orgIdentifier}`,
        label: `Organization: ${orgIdentifier}`
      },
      {
        url: `/account/${accountIdentifier}/module/cde/orgs/${orgIdentifier}/projects/${projectIdentifier}`,
        label: `Project: ${projectIdentifier}`
      },
      {
        url: routes.toCDEAITasks({ space }),
        label: 'Tasks'
      }
    ]
  }

  return (
    <>
      <Page.Header title="Task Listing" breadcrumbs={<Breadcrumbs links={getBreadcrumbLinks()} />} />
      <Page.Body>
        <Container>
          <Card>
            <Container padding="large">
              <Layout.Vertical spacing="small">
                <Heading font={{ weight: 'bold' }} color={Color.BLACK} level={2}>
                  Task Listing page
                </Heading>
                <Text font={{ size: 'medium' }}>Coming soon</Text>
              </Layout.Vertical>
            </Container>
          </Card>
        </Container>
      </Page.Body>
    </>
  )
}

export default AITaskListing
