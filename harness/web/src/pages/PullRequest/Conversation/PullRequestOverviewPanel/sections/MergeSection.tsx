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
import React, { useMemo } from 'react'
import { Color, FontVariation } from '@harnessio/design-system'
import cx from 'classnames'
import {
  Button,
  ButtonSize,
  ButtonVariation,
  Container,
  Layout,
  StringSubstitute,
  TableV2,
  Text,
  useToggle
} from '@harnessio/uicore'
import { Render } from 'react-jsx-match'
import type { CellProps, Column } from 'react-table'
import { Images } from 'images'
import type { TypesPullReq } from 'services/code'
import { useStrings } from 'framework/strings'
import Success from '../../../../../icons/code-success.svg?url'
import Fail from '../../../../../icons/code-fail.svg?url'
import CommandLineInfo from '../CommandLineInfo'
import css from '../PullRequestOverviewPanel.module.scss'

interface MergeSectionProps {
  mergeable: boolean
  unchecked: boolean
  pullReqMetadata: TypesPullReq
  conflictingFiles: string[] | undefined
}

interface ConflictingFilesInterface {
  name: string
}
const MergeSection = (props: MergeSectionProps) => {
  const { mergeable, unchecked, pullReqMetadata, conflictingFiles } = props
  const { getString } = useStrings()
  const [isExpanded, toggleExpanded] = useToggle(false)
  const [showCommandLineInfo, toggleShowCommandLineInfo] = useToggle(false)
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const columns: Column<any>[] = useMemo(
    () => [
      {
        id: 'conflictingFiles',
        width: '45%',
        sort: true,
        Header: `Conflicting Files (${conflictingFiles?.length})`,
        accessor: 'conflictingFiles',
        Cell: ({ row }: CellProps<ConflictingFilesInterface>) => {
          return (
            <Text
              lineClamp={1}
              className={css.conflictingFileName}
              padding={{ left: 'small', right: 'small' }}
              color={Color.BLACK}>
              {row.original}
            </Text>
          )
        }
      }
    ], // eslint-disable-next-line react-hooks/exhaustive-deps
    [conflictingFiles]
  )
  return (
    <>
      <Container className={cx(css.sectionContainer, css.borderContainer)}>
        <Layout.Horizontal flex={{ justifyContent: 'space-between' }}>
          <Layout.Horizontal flex={{ alignItems: 'center', justifyContent: 'start' }}>
            {(unchecked && <img src={Images.PrUnchecked} width={25} height={25} />) || (
              <>
                {mergeable ? (
                  <img alt={getString('success')} width={26} height={26} src={Success} />
                ) : (
                  <img alt={getString('failed')} width={26} height={26} src={Fail} />
                )}
              </>
            )}

            {(unchecked && (
              <Layout.Vertical padding={{ left: 'medium' }}>
                <Text padding={{ bottom: 'xsmall' }} className={css.sectionTitle}>
                  {getString('mergeCheckInProgress')}
                </Text>
                <Text className={css.sectionSubheader}> {getString('pr.checkingToMerge')}</Text>
              </Layout.Vertical>
            )) || (
              <Layout.Vertical>
                {!mergeable && (
                  <Text
                    className={css.sectionTitle}
                    color={Color.RED_700}
                    padding={{ left: 'medium', bottom: 'xsmall' }}>
                    {getString('conflictsFoundInThisBranch')}
                  </Text>
                )}
                <Text
                  flex
                  className={mergeable ? css.sectionTitle : css.sectionSubheader}
                  color={mergeable ? Color.GREEN_800 : Color.GREY_450}
                  font={{ variation: FontVariation.BODY }}
                  padding={{ left: 'medium' }}>
                  <StringSubstitute
                    str={mergeable ? getString('prHasNoConflicts') : getString('pr.useCmdLineToResolveConflicts')}
                    vars={{
                      name: pullReqMetadata.target_branch,
                      cmd: (
                        <Text
                          onClick={toggleShowCommandLineInfo}
                          padding={{ left: 'xsmall', right: 'xsmall' }}
                          className={css.cmdText}>
                          {getString('commandLine')}
                        </Text>
                      )
                    }}
                  />
                </Text>
              </Layout.Vertical>
            )}
          </Layout.Horizontal>
          {!mergeable && !unchecked && (
            <Button
              padding={{ right: 'unset' }}
              className={cx(css.blueText, css.buttonPadding)}
              variation={ButtonVariation.LINK}
              size={ButtonSize.SMALL}
              text={getString(isExpanded ? 'showLessMatches' : 'showMoreText')}
              onClick={toggleExpanded}
              rightIcon={isExpanded ? 'main-chevron-up' : 'main-chevron-down'}
              iconProps={{ size: 10, margin: { left: 'xsmall' } }}
            />
          )}
        </Layout.Horizontal>
        <Render when={showCommandLineInfo}>
          <CommandLineInfo pullReqMetadata={pullReqMetadata} />
        </Render>
      </Container>
      <Render when={isExpanded}>
        <Container className={css.conflictingContainer}>
          <Container padding={{ left: 'xxxlarge' }} className={css.greyContainer}>
            <TableV2
              className={css.conflictingFilesTable}
              sortable
              columns={columns}
              // eslint-disable-next-line @typescript-eslint/no-explicit-any
              data={conflictingFiles as any}
              getRowClassName={() => css.row}
            />
          </Container>
        </Container>
      </Render>
    </>
  )
}

export default MergeSection
