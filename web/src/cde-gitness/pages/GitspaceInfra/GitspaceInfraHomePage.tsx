import React from 'react'
import { Breadcrumbs, Button, ButtonVariation, Page } from '@harnessio/uicore'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import { routes } from 'cde-gitness/RouteDefinitions'
import NoDataCard from './NoDataCard'
import css from './GitspaceInfraHomePage.module.scss'

const GitspaceInfraHomePage = () => {
  const { getString } = useStrings()
  const { accountInfo } = useAppContext()
  const data = null

  return (
    <>
      <Page.Header
        title={getString('cde.gitspaceInfra')}
        content={
          data ? (
            <Button
              icon="Edit"
              iconProps={{ size: 12 }}
              variation={ButtonVariation.SECONDARY}
              text={getString('cde.edit')}
            />
          ) : (
            <></>
          )
        }
        breadcrumbs={
          <Breadcrumbs
            className={css.customBreadcumbStyles}
            links={[
              {
                url: routes.toModuleRoute({ accountId: accountInfo?.identifier }),
                label: `${getString('cde.account')}: ${accountInfo?.name}`
              },
              {
                url: routes.toCDEGitspaceInfra({ accountId: accountInfo?.identifier }),
                label: getString('cde.gitspaceInfra')
              }
            ]}
          />
        }
      />
      <Page.Body className={css.main}>{data ? <>Data</> : <NoDataCard />}</Page.Body>
    </>
  )
}

export default GitspaceInfraHomePage
