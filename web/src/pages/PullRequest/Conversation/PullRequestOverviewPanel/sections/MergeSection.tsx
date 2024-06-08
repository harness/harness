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
import { Color, FontVariation } from '@harnessio/design-system'
import { Container, Layout, Text } from '@harnessio/uicore'
import { Images } from 'images'
import type { TypesPullReq } from 'services/code'
import { useStrings } from 'framework/strings'
import Success from '../../../../../icons/code-success.svg?url'
import Fail from '../../../../../icons/code-fail.svg?url'
import css from '../PullRequestOverviewPanel.module.scss'

interface MergeSectionProps {
  mergeable: boolean
  unchecked: boolean
  pullReqMetadata: TypesPullReq
}

const MergeSection = (props: MergeSectionProps) => {
  const { mergeable, unchecked, pullReqMetadata } = props
  const { getString } = useStrings()

  return (
    <Container>
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
              <Text className={css.sectionTitle} color={Color.RED_700} padding={{ left: 'medium', bottom: 'xsmall' }}>
                {getString('allConflictsNeedToBeResolved')}
              </Text>
            )}
            <Text
              className={mergeable ? css.sectionTitle : css.sectionSubheader}
              color={mergeable ? Color.GREEN_800 : Color.GREY_450}
              font={{ variation: FontVariation.BODY }}
              padding={{ left: 'medium' }}>
              {getString(mergeable ? 'prHasNoConflicts' : 'pr.cantBeMerged', { name: pullReqMetadata?.target_branch })}
            </Text>
          </Layout.Vertical>
        )}
      </Layout.Horizontal>
    </Container>
  )
}

export default MergeSection
