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

import React, { useState } from 'react'
import cx from 'classnames'
import { useHistory } from 'react-router-dom'
import { defaultTo } from 'lodash-es'
import {
  Button,
  ButtonSize,
  ButtonVariation,
  Container,
  FlexExpander,
  Layout,
  StringSubstitute,
  Text
} from '@harnessio/uicore'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import { GitRefLink } from 'components/GitRefLink/GitRefLink'
import { TimePopoverWithLocal } from 'utils/timePopoverLocal/TimePopoverWithLocal'
import { makeDiffRefs } from 'utils/GitUtils'
import type { RepoRepositoryOutput, TypesBranchTable } from 'services/code'
import Branches from '../../icons/Branches.svg?url'
import css from './PRBanner.module.scss'

export const PRBanner = ({
  candidateBranch,
  repoMetadata
}: {
  candidateBranch: TypesBranchTable
  repoMetadata: RepoRepositoryOutput
}) => {
  const { getString } = useStrings()
  const [showPRBanner, setShowPRBanner] = useState(true)
  const { routes } = useAppContext()
  const history = useHistory()

  if (!showPRBanner) return null

  return (
    <Container className={cx(css.main, css.banner)}>
      <Layout.Horizontal spacing="small" flex={{ alignItems: 'center' }} className={css.layout}>
        <img src={Branches} width={20} height={20} />
        <Text flex={{ alignItems: 'center' }} className={css.message}>
          <StringSubstitute
            str={getString('pr.createPRBannerInfo')}
            vars={{
              branch: (
                <Container padding={{ right: 'xsmall' }}>
                  <GitRefLink
                    text={candidateBranch.name as string}
                    url={routes.toCODERepository({
                      repoPath: repoMetadata.path as string,
                      gitRef: candidateBranch.name
                    })}
                    showCopy
                    className={css.link}
                  />
                </Container>
              ),
              time: (
                <TimePopoverWithLocal
                  className={css.dateText}
                  time={defaultTo(candidateBranch.updated, 0)}
                  inline={false}
                />
              )
            }}
          />
        </Text>
        <FlexExpander />
        <Button
          variation={ButtonVariation.SECONDARY}
          text={getString('compareAndPullRequest')}
          onClick={() =>
            history.push(
              routes.toCODECompare({
                repoPath: repoMetadata.path as string,
                diffRefs: makeDiffRefs(repoMetadata.default_branch as string, candidateBranch.name as string)
              })
            )
          }
        />
        <Button
          variation={ButtonVariation.ICON}
          minimal
          icon="main-close"
          role="close"
          iconProps={{ size: 10 }}
          size={ButtonSize.SMALL}
          onClick={() => {
            setShowPRBanner(false)
          }}
        />
      </Layout.Horizontal>
    </Container>
  )
}
