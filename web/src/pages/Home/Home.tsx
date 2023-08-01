import React from 'react'
import { ButtonVariation, Container, FontVariation, Layout, PageBody, Text } from '@harness/uicore'
import { useGet } from 'restful-react'
// import type { TypesSpace } from 'services/code'
import { useStrings } from 'framework/strings'
// import { usePageIndex } from 'hooks/usePageIndex'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { useAppContext } from 'AppContext'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { NewSpaceModalButton } from 'components/NewSpaceModalButton/NewSpaceModalButton'
import css from './Home.module.scss'

export default function Home() {
  const { getString } = useStrings()
  // const [searchTerm, setSearchTerm] = useState('')
  // const [page, setPage] = usePageIndex(1)
  const { currentUser } = useAppContext()
  const { space } = useGetRepositoryMetadata()

  const spaces = []
  const { refetch } = useGet({
    path: '/api/v1/user/memberships'
  })
  const NewSpaceButton = (
    <NewSpaceModalButton
      space={space}
      modalTitle={getString('createSpace')}
      text={getString('createSpace')}
      variation={ButtonVariation.PRIMARY}
      icon="plus"
      width={173}
      height={48}
      onRefetch={refetch}
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
