/*
 * Copyright 2024 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React from 'react'
import { useHistory, useParams } from 'react-router-dom'
import type { Webhook } from '@harnessio/react-har-service-client'
import { Button, ButtonVariation, useToggleOpen } from '@harnessio/uicore'

import { useRoutes } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import type { RepositoryDetailsTabPathParams } from '@ar/routes/types'

import CreateWebhookModal from './CreateWebhookModal'

export default function CreateWebhookButton() {
  const { isOpen, close, open } = useToggleOpen()
  const { getString } = useStrings()
  const { repositoryIdentifier } = useParams<RepositoryDetailsTabPathParams>()

  const routes = useRoutes()
  const history = useHistory()

  const handleAfterCreateWebhook = (data: Webhook) => {
    history.push(
      routes.toARRepositoryWebhookDetails({
        repositoryIdentifier,
        webhookIdentifier: data.identifier
      })
    )
  }
  return (
    <>
      <Button variation={ButtonVariation.PRIMARY} icon="plus" iconProps={{ size: 10 }} onClick={open}>
        {getString('webhookList.newWebhook')}
      </Button>
      <CreateWebhookModal isOpen={isOpen} onClose={close} onSubmit={handleAfterCreateWebhook} />
    </>
  )
}
