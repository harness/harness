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

import React, { useEffect, useMemo } from 'react'
import { Divider, PopoverInteractionKind, Position } from '@blueprintjs/core'
import { Checkbox, Container, FlexExpander, Layout, Popover, Text } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import ReactTimeago from 'react-timeago'

import { useStrings } from 'framework/strings'
import type { TypesCommit } from 'services/code'

import css from '../Changes.module.scss'

type CommitRangeDropdownProps = {
  allCommits: TypesCommit[]
  selectedCommits: string[]
  setSelectedCommits: React.Dispatch<React.SetStateAction<string[]>>
}

const sortSelectedCommits = (selectedCommits: string[], sortedCommits: string[]) => {
  return selectedCommits.sort((commitA, commitB) => {
    const commitAIdx = sortedCommits.indexOf(commitA)
    const commitBIdx = sortedCommits.indexOf(commitB)

    return commitBIdx - commitAIdx
  })
}

function getBiggerSubarray(array: Array<string>, index: number) {
  if (index >= 0 && index < array.length) {
    const subarray1 = array.slice(0, index)
    const subarray2 = array.slice(index + 1)

    return subarray1.length > subarray2.length ? subarray1 : subarray2
  } else {
    return []
  }
}

const getCommitRange = (selectedCommits: string[], allCommitsSHA: string[]) => {
  const sortedCommits = sortSelectedCommits(selectedCommits, allCommitsSHA)
  const selectedCommitRange = allCommitsSHA
    .slice(allCommitsSHA.indexOf(sortedCommits[sortedCommits.length - 1]), allCommitsSHA.indexOf(sortedCommits[0]) + 1)
    .reverse()

  return selectedCommitRange
}

const CommitRangeDropdown: React.FC<CommitRangeDropdownProps> = ({
  allCommits,
  selectedCommits,
  setSelectedCommits
}) => {
  const { getString } = useStrings()
  const allCommitsSHA = useMemo(() => allCommits.map(commit => commit.sha as string), [allCommits])

  useEffect(() => {
    if (selectedCommits.length && allCommitsSHA.length) {
      setSelectedCommits(prevVal => getCommitRange(prevVal, allCommitsSHA))
    }
  }, [allCommitsSHA, setSelectedCommits, selectedCommits.length])

  const handleCheckboxClick = (
    event: React.MouseEvent<HTMLInputElement | HTMLDivElement, MouseEvent>,
    selectedCommitSHA: string
  ) => {
    setSelectedCommits(current => {
      // handle single commit clicked (either no shift held, or no commit selected yet)
      if (!event.shiftKey || current.length == 0) {
        return [selectedCommitSHA]
      }

      // handle already selected commit clicked
      if (current.includes(selectedCommitSHA)) {
        const sortedCommits = sortSelectedCommits(current, allCommitsSHA)
        const subArray = getBiggerSubarray(sortedCommits, sortedCommits.indexOf(selectedCommitSHA))

        return subArray
      }

      // clicked commit is outside of current range - extend it!
      const extendedArray = getCommitRange([...current, selectedCommitSHA], allCommitsSHA)

      // NOTE: this CAN contain all commits - we let it through for consistent user experience.
      // This way, the user sees selected exactly what they clicked on (+ we don't have to handle single commit pr differently)
      return extendedArray
    })
  }

  const areAllCommitsSelected = !selectedCommits.length

  return (
    <Popover
      minimal
      interactionKind={PopoverInteractionKind.CLICK}
      position={Position.BOTTOM_LEFT}
      onClose={() => setSelectedCommits(selectedCommits)}
      content={
        <Container padding="medium" width={350}>
          <Checkbox
            labelElement={<Text font={{ variation: FontVariation.SMALL_SEMI }}>{getString('allCommits')}</Text>}
            checked={areAllCommitsSelected}
            onClick={() => setSelectedCommits([])}
            margin={{ bottom: 'small' }}
          />
          <Divider />
          <Container margin={{ top: 'small', bottom: 'small' }} style={{ maxHeight: '40vh', overflow: 'auto' }}>
            {allCommits?.map((prCommit, index) => {
              const isSelected = selectedCommits.includes(prCommit.sha || '')

              return (
                <Layout.Horizontal
                  key={prCommit.sha}
                  style={{ alignItems: 'center', cursor: 'pointer' }}
                  padding={{ top: 'xsmall', bottom: 'xsmall' }}
                  onClick={e => handleCheckboxClick(e, prCommit.sha as string)}>
                  <Checkbox checked={isSelected} onClick={e => handleCheckboxClick(e, prCommit.sha as string)} />
                  <Text font={{ variation: FontVariation.SMALL_SEMI }} lineClamp={1} padding={{ right: 'small' }}>
                    {`${allCommits.length - index} ${prCommit.title}`}
                  </Text>
                  <FlexExpander />
                  <Text font={{ variation: FontVariation.SMALL_SEMI }}>
                    <ReactTimeago style={{ whiteSpace: 'nowrap' }} date={prCommit.committer?.when || ''} />
                  </Text>
                </Layout.Horizontal>
              )
            })}
          </Container>
          <Divider />
          <Text
            padding={{ top: 'small' }}
            color={Color.GREY_250}
            font={{ variation: FontVariation.SMALL_SEMI, align: 'center' }}>
            {getString('selectRange')}
          </Text>
        </Container>
      }>
      <Text
        className={css.commitsDropdown}
        rightIcon="chevron-down"
        color={Color.GREY_700}
        font={{ variation: FontVariation.BODY2 }}
        margin={{ right: 'medium' }}>
        {areAllCommitsSelected
          ? getString('allCommits')
          : `${selectedCommits.length} ${selectedCommits.length > 1 ? getString('commits') : getString('commit')}`}
      </Text>
    </Popover>
  )
}

export default CommitRangeDropdown
