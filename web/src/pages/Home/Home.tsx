import React from 'react'
import { ButtonVariation, Container, Layout, PageBody, Text } from '@harnessio/uicore'
import { FontVariation } from '@harnessio/design-system'
import { useGet } from 'restful-react'
import { useHistory } from 'react-router-dom'
// import type { TypesSpace } from 'services/code'
import { useStrings } from 'framework/strings'
// import { usePageIndex } from 'hooks/usePageIndex'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { useAppContext } from 'AppContext'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { voidFn } from 'utils/Utils'
import { NewSpaceModalButton } from 'components/NewSpaceModalButton/NewSpaceModalButton'
import css from './Home.module.scss'

export default function Home() {
  const { getString } = useStrings()
  const history = useHistory()
  const { routes } = useAppContext()
  // const [searchTerm, setSearchTerm] = useState('')
  // const [page, setPage] = usePageIndex(1)
  const { currentUser } = useAppContext()
  const { space } = useGetRepositoryMetadata()

  const spaces = []

  const NewSpaceButton = (
    <NewSpaceModalButton
      space={space}
      modalTitle={getString('createASpace')}
      text={getString('newSpace')}
      variation={ButtonVariation.PRIMARY}
      icon="plus"
      width={173}
      height={48}
      onRefetch={voidFn(() => {})}
      handleNavigation={spaceName => {
        history.push(routes.toCODERepositories({ space: spaceName }))
      }}
    />
  )
  return (
    <Container className={css.main}>
      <PageBody
        error={false}
        // retryOnError={voidFn(refetch)}
      >
        <LoadingSpinner visible={false} />

        {spaces.length == 0 ? (
          <Container className={css.container} flex={{ justifyContent: 'center', align: 'center-center' }}>
            <Layout.Vertical className={css.spaceContainer} spacing="small">
              <Text font={{ variation: FontVariation.H2 }}>
                {getString('homepage.welcomeText', {
                  currentUser: currentUser?.display_name
                })}
              </Text>
              <Text font={{ variation: FontVariation.BODY1 }}>{getString('homepage.firstStep')} </Text>
              <Container padding={{ top: 'large' }} flex={{ justifyContent: 'center' }}>
                {NewSpaceButton}
              </Container>
            </Layout.Vertical>
          </Container>
        ) : null}
      </PageBody>
    </Container>
  )
}
