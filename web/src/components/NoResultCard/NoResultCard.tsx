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
import { Button, Container, ButtonVariation, NoDataCard } from '@harnessio/uicore'
import type { IconName } from '@harnessio/icons'
import { noop } from 'lodash-es'
import { CodeIcon } from 'utils/GitUtils'
import { useStrings } from 'framework/strings'
import { Images } from 'images'
import css from './NoResultCard.module.scss'

interface NoResultCardProps {
  showWhen?: () => boolean
  forSearch: boolean
  title?: string
  message?: string
  emptySearchMessage?: string
  buttonText?: string
  buttonIcon?: IconName
  onButtonClick?: () => void
  permissionProp?: { disabled: boolean; tooltip: JSX.Element | string } | undefined
  standalone?: boolean
  forFilter?: boolean
  emptyFilterMessage?: string
}

export const NoResultCard: React.FC<NoResultCardProps> = ({
  showWhen = () => true,
  forSearch,
  forFilter = false,
  title,
  message,
  emptySearchMessage,
  emptyFilterMessage,
  buttonText = '',
  buttonIcon = CodeIcon.Add,
  onButtonClick = noop,
  permissionProp
}) => {
  const { getString } = useStrings()

  if (!showWhen()) {
    return null
  }

  return (
    <Container className={css.main}>
      <NoDataCard
        image={Images.EmptyState}
        messageTitle={forSearch || forFilter ? title || getString('noResultTitle') : undefined}
        message={
          forSearch
            ? emptySearchMessage || getString('noResultMessage')
            : forFilter
            ? emptyFilterMessage || getString('noFilterResultMessage')
            : message || getString('noResultMessage')
        }
        button={
          forSearch || forFilter ? undefined : (
            <Button
              variation={ButtonVariation.PRIMARY}
              text={buttonText}
              icon={buttonIcon as IconName}
              onClick={onButtonClick}
              {...permissionProp}
            />
          )
        }
      />
    </Container>
  )
}
