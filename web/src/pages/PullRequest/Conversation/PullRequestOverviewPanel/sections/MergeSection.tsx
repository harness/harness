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
              <img alt="success" width={27} height={27} src={Success} />
            ) : (
              <img alt="fail" width={27} height={27} src={Fail} />
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
