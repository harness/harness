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
import { Avatar, Container, Layout, Text, Utils } from '@harnessio/uicore'
import { GitCommit, GitFork, Label } from 'iconoir-react'
import { Color } from '@harnessio/design-system'
import { PipeSeparator } from 'components/PipeSeparator/PipeSeparator'
import { useAppContext } from 'AppContext'
import type { EnumTriggerAction } from 'services/code'
import { CommitActions } from 'components/CommitActions/CommitActions'
import css from './ExecutionText.module.scss'

export enum ExecutionTrigger {
  CRON = 'cron',
  MANUAL = 'manual',
  PUSH = 'push',
  PULL = 'pull_request',
  TAG = 'tag'
}

enum PillType {
  BRANCH = 'branch',
  TAG = 'tag',
  COMMIT = 'commit'
}

interface PillProps {
  type: PillType
  text: string
}

interface ExecutionTextProps {
  authorName: string
  authorEmail: string
  repoPath: string
  commitRef: string
  event: ExecutionTrigger
  target: string
  afterRef: string
  source: string
  action: EnumTriggerAction | undefined
}

const Pill: React.FC<PillProps> = ({ type, text }) => {
  let Icon

  switch (type) {
    case PillType.BRANCH:
      Icon = GitFork
      break
    case PillType.TAG:
      Icon = Label
      break
    case PillType.COMMIT:
      Icon = GitCommit
      break
    default:
      Icon = GitCommit
  }

  return (
    <Layout.Horizontal
      spacing={'xsmall'}
      style={{ alignItems: 'center', borderRadius: '4px' }}
      className={css.pillContainer}>
      <Icon height={12} width={12} color={Utils.getRealCSSColor(Color.GREY_500)} />
      <Text className={css.pillText} font={{ size: 'xsmall' }}>
        {text}
      </Text>
    </Layout.Horizontal>
  )
}

export const ExecutionText: React.FC<ExecutionTextProps> = ({
  authorName,
  authorEmail,
  repoPath,
  commitRef,
  event,
  target,
  afterRef,
  source,
  action
}) => {
  const { routes } = useAppContext()

  let componentToRender

  switch (event) {
    case ExecutionTrigger.CRON:
      componentToRender = (
        <Text font={{ size: 'small' }} className={css.author}>
          Triggered by CRON job
        </Text>
      )
      break
    case ExecutionTrigger.MANUAL:
      componentToRender = (
        <Text font={{ size: 'small' }} className={css.author}>{`${authorName} triggered manually`}</Text>
      )
      break
    case ExecutionTrigger.PUSH:
      componentToRender = (
        <>
          <Text font={{ size: 'small' }} className={css.author}>{`${authorName} pushed`}</Text>
          <Pill type={PillType.COMMIT} text={afterRef?.slice(0, 6)} />
          <Text font={{ size: 'small' }} className={css.author}>
            to
          </Text>
          <Pill type={PillType.BRANCH} text={target} />
        </>
      )
      break
    case ExecutionTrigger.PULL:
      componentToRender = (
        <>
          <Text font={{ size: 'small' }} className={css.author}>{`${authorName} ${
            action === 'pullreq_reopened'
              ? 'reopened'
              : action === 'pullreq_branch_updated'
              ? 'updated'
              : action === 'pullreq_closed'
              ? 'closed'
              : action === 'pullreq_merged'
              ? 'merged'
              : 'created'
          } pull request`}</Text>
          <Pill type={PillType.BRANCH} text={source} />
          <Text font={{ size: 'small' }} className={css.author}>
            to
          </Text>
          <Pill type={PillType.BRANCH} text={target} />
        </>
      )
      break
    case ExecutionTrigger.TAG:
      componentToRender = (
        <>
          <Text font={{ size: 'small' }} className={css.author}>{`${authorName} ${
            action === 'branch_updated' ? 'updated' : 'created'
          }`}</Text>
          <Pill type={PillType.TAG} text={target.split('/').pop() as string} />
        </>
      )
      break
    default:
      componentToRender = (
        <Text font={{ size: 'small' }} className={css.author}>
          Unknown trigger
        </Text>
      )
  }

  return (
    <Layout.Horizontal spacing={'small'} style={{ alignItems: 'center', marginLeft: '1.2rem' }}>
      <Avatar email={authorEmail} name={authorName} size="small" hoverCard={false} />
      {componentToRender}
      <PipeSeparator height={7} />
      <Container onClick={Utils.stopEvent}>
        <CommitActions
          href={routes.toCODECommit({
            repoPath: repoPath,
            commitRef: commitRef
          })}
          sha={commitRef}
          enableCopy
        />
      </Container>
    </Layout.Horizontal>
  )
}
