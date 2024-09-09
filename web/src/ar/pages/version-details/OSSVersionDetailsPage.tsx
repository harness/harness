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

import React, { useEffect } from 'react'
import { useHistory } from 'react-router-dom'
import { Button } from '@harnessio/uicore'
import { Drawer, Position } from '@blueprintjs/core'

import { useDecodedParams, useParentHooks, useRoutes } from '@ar/hooks'
import type { VersionDetailsPathParams } from '@ar/routes/types'
import VersionProvider from './context/VersionProvider'
import OSSVersionDetails from './OSSVersionDetails'

import css from './VersionDetails.module.scss'

export default function OSSVersionDetailsPage() {
  const { useModalHook } = useParentHooks()
  const routes = useRoutes()
  const pathParams = useDecodedParams<VersionDetailsPathParams>()
  const history = useHistory()

  const handleCloseModal = () => {
    history.push(
      routes.toARArtifactDetails({
        repositoryIdentifier: pathParams.repositoryIdentifier,
        artifactIdentifier: pathParams.artifactIdentifier
      })
    )
  }

  const [showModal, hideModal] = useModalHook(() => {
    return (
      <Drawer
        className="arApp"
        position={Position.RIGHT}
        isOpen={true}
        isCloseButtonShown={false}
        size={'70%'}
        onClose={() => {
          hideModal()
          handleCloseModal()
        }}>
        <VersionProvider
          repoKey={pathParams.repositoryIdentifier}
          artifactKey={pathParams.artifactIdentifier}
          versionKey={pathParams.versionIdentifier}
          className={css.ossVersionDetailsModal}>
          <Button minimal className={css.closeBtn} icon="cross" withoutBoxShadow onClick={hideModal} />
          <OSSVersionDetails />
        </VersionProvider>
      </Drawer>
    )
  }, [pathParams.versionIdentifier])

  useEffect(() => {
    showModal()
  }, [])
  return <></>
}
