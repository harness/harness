import React from 'react'
import {
  Button,
  Page,
  ButtonVariation,
  Breadcrumbs,
  HarnessDocTooltip,
  Container,
  Layout,
  Text
} from '@harnessio/uicore'
import { FontVariation } from '@harnessio/design-system'
import { useHistory } from 'react-router-dom'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { PageBrowserProps, getErrorMessage } from 'utils/Utils'
import noSpace from 'cde-gitness/assests/no-gitspace.svg?url'
import { ResourceListingPagination } from 'components/ResourceListingPagination/ResourceListingPagination'
import { useQueryParams } from 'hooks/useQueryParams'
import { usePageIndex } from 'hooks/usePageIndex'
import { ListGitspaces } from 'cde-gitness/components/GitspaceListing/ListGitspaces'
import CDEHomePage from 'cde-gitness/components/CDEHomePage/CDEHomePage'
import { useLisitngApi } from '../../hooks/useLisitngApi'
import css from './GitspacesListing.module.scss'
import zeroDayCss from 'cde-gitness/components/CDEHomePage/CDEHomePage.module.scss'

const GitspaceListing = () => {
  const space = useGetSpaceParam()
  const history = useHistory()
  const { getString } = useStrings()
  const { routes } = useAppContext()
  const pageBrowser = useQueryParams<PageBrowserProps>()
  const pageInit = pageBrowser.page ? parseInt(pageBrowser.page) : 1
  const [page, setPage] = usePageIndex(pageInit)

  const { data = '', loading = false, error = undefined, refetch, response } = useLisitngApi({ page })

  // useEffect(() => {
  //   if (!data && !loading) {
  //     history.push(routes.toCDEGitspacesCreate({ space }))
  //   }
  // }, [data, loading])

  return (
    <>
      {data?.length !== 0 && (
        <Page.Header
          title={
            <div className="ng-tooltip-native">
              <h2 data-tooltip-id="artifactListPageHeading"> {getString('cde.gitspaces')}</h2>
              <HarnessDocTooltip tooltipId="GitSpaceListPageHeading" useStandAlone={true} />
            </div>
          }
          breadcrumbs={
            <Breadcrumbs links={[{ url: routes.toCDEGitspaces({ space }), label: getString('cde.gitspaces') }]} />
          }
          toolbar={
            <Button
              onClick={() => history.push(routes.toCDEGitspacesCreate({ space }))}
              variation={ButtonVariation.PRIMARY}>
              {getString('cde.newGitspace')}
            </Button>
          }
        />
      )}
      <Container className={data?.length === 0 ? zeroDayCss.background : css.main}>
        <Layout.Vertical spacing={'large'}>
          {data && data?.length === 0 ? (
            <CDEHomePage />
          ) : (
            <Page.Body
              loading={loading}
              error={
                error ? (
                  <Layout.Vertical spacing={'large'}>
                    <Text font={{ variation: FontVariation.FORM_MESSAGE_DANGER }}>{getErrorMessage(error)}</Text>
                    <Button
                      onClick={() => refetch?.()}
                      variation={ButtonVariation.PRIMARY}
                      text={getString('cde.retry')}
                    />
                  </Layout.Vertical>
                ) : null
              }
              noData={{
                when: () => data?.length === 0,
                image: noSpace,
                message: getString('cde.noGitspaces')
              }}>
              {data?.length && (
                <>
                  <ListGitspaces data={data || []} refreshList={refetch} />
                  <ResourceListingPagination response={response} page={page} setPage={setPage} />
                </>
              )}
            </Page.Body>
          )}
        </Layout.Vertical>
      </Container>
    </>
  )
}

export default GitspaceListing
