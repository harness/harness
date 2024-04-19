/*
 * Copyright 2023 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React from 'react'
import { ButtonVariation, ButtonSize, Container, Layout, PageBody, Text, Button } from '@harnessio/uicore'
import { FontVariation } from '@harnessio/design-system'
import { noop } from 'lodash-es'
import { useGet } from 'restful-react'
import { useHistory } from 'react-router-dom'
import { useStrings } from 'framework/strings'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { useAppContext } from 'AppContext'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { NewSpaceModalButton } from 'components/NewSpaceModalButton/NewSpaceModalButton'
import css from './Home.module.scss'

export default function Home() {
  const { getString } = useStrings()
  const history = useHistory()
  const { routes } = useAppContext()
  const { currentUser } = useAppContext()
  const { space } = useGetRepositoryMetadata()

  const { data: spaces } = useGet({
    path: '/api/v1/user/memberships',

    debounce: 500
  })

  const NewSpaceButton = (
    <NewSpaceModalButton
      size={ButtonSize.LARGE}
      className={css.bigButton}
      space={space}
      modalTitle={getString('createASpace')}
      text={getString('newSpace')}
      variation={ButtonVariation.PRIMARY}
      icon="plus"
      onRefetch={noop}
      handleNavigation={spaceName => {
        history.push(routes.toCODERepositories({ space: spaceName }))
      }}
      onSubmit={data => {
        history.push(routes.toCODERepositories({ space: data.path as string }))
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

        {spaces?.length === 0 ? (
          <Container className={css.container} flex={{ justifyContent: 'center', align: 'center-center' }}>
            <Layout.Vertical className={css.spaceContainer} spacing="small">
              <Text flex={{ justifyContent: 'center', align: 'center-center' }} font={{ variation: FontVariation.H2 }}>
                {getString('homepage.welcomeText', {
                  currentUser: currentUser?.display_name || currentUser?.email || ''
                })}
              </Text>
              <Text
                flex={{ justifyContent: 'center', align: 'center-center' }}
                font={{ variation: FontVariation.BODY1 }}>
                {getString('homepage.firstStep')}
              </Text>
              <Container className={css.buttonContainer} padding={{ top: 'large' }} flex={{ justifyContent: 'center' }}>
                {NewSpaceButton}
              </Container>
            </Layout.Vertical>
          </Container>
        ) : (
          <Container className={css.container} flex={{ justifyContent: 'center', align: 'center-center' }}>
            <Layout.Vertical className={css.spaceContainer} spacing="small">
              <Text flex={{ justifyContent: 'center', align: 'center-center' }} font={{ variation: FontVariation.H2 }}>
                {getString('homepage.selectSpaceTitle')}
              </Text>
              <Text font={{ variation: FontVariation.BODY1 }}>{getString('homepage.selectSpaceContent')}</Text>
              <Container className={css.buttonContainer} padding={{ top: 'large' }} flex={{ justifyContent: 'center' }}>
                <Button
                  text={getString('homepage.selectSpace')}
                  size={ButtonSize.LARGE}
                  variation={ButtonVariation.PRIMARY}
                  onClick={() => {
                    // TODO: create a space provider to trigger open modal of space selector
                    const button = document.body.querySelectorAll('.bp3-popover-target')[0] as HTMLElement
                    button.click()
                  }}
                />
              </Container>
            </Layout.Vertical>
          </Container>
        )}
      </PageBody>
    </Container>
  )
}
