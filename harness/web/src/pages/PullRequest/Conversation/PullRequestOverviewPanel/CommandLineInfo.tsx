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
import { Container, Layout, StringSubstitute, Text } from '@harnessio/uicore'
import type { TypesPullReq } from 'services/code'
import { useStrings } from 'framework/strings'
import { CopyButton } from 'components/CopyButton/CopyButton'
import { CodeIcon } from 'utils/GitUtils'
import css from './PullRequestOverviewPanel.module.scss'

interface CommandLineInfoProps {
  pullReqMetadata: TypesPullReq
}

const CommandLineInfo = (props: CommandLineInfoProps) => {
  const { pullReqMetadata } = props

  const { getString } = useStrings()
  const stepOneCopy = getString('cmdlineInfo.stepOneSub').replace('{target}', pullReqMetadata.target_branch as string)
  const stepTwoCopy = getString('cmdlineInfo.stepTwoSub').replace('{source}', pullReqMetadata.source_branch as string)
  const stepThreeCopy = getString('cmdlineInfo.stepThreeSub').replace(
    '{target}',
    pullReqMetadata.target_branch as string
  )
  const stepFiveCopy = getString('cmdlineInfo.stepFiveSub').replace('{source}', pullReqMetadata.source_branch as string)
  return (
    <Container
      className={css.cmdInfoContainer}
      margin={{ top: 'small', bottom: 'small' }}
      padding={{ top: 'large', left: 'xxxlarge', right: 'xlarge', bottom: 'large' }}>
      <Layout.Vertical>
        <Container className={css.cmdTextTitleContainer}>
          <Text padding={{ bottom: 'xsmall' }} font={{ variation: FontVariation.H5 }}>
            {getString('cmdlineInfo.title')}
          </Text>
        </Container>
        <Text
          color={Color.GREY_450}
          className={css.stepText}
          font={{ variation: FontVariation.BODY2 }}
          padding={{ top: 'xsmall' }}>
          {getString('cmdlineInfo.content')}
        </Text>
        <Layout.Vertical>
          <Layout.Horizontal flex={{ alignItems: 'center', justifyContent: 'flex-start' }} padding={{ top: 'small' }}>
            <Text className={css.checkName} padding={{ right: 'small' }} font={{ variation: FontVariation.CARD_TITLE }}>
              {getString('stepNum', { num: 1 }).toUpperCase()}
            </Text>
            <Text color={Color.GREY_450} className={css.stepText} font={{ variation: FontVariation.BODY2 }}>
              {getString('cmdlineInfo.stepOne')}
            </Text>
          </Layout.Horizontal>
          <Layout.Horizontal margin={{ left: 'small' }} padding={{ top: 'xsmall', left: 'xxxlarge' }}>
            <Layout.Horizontal
              flex={{ justifyContent: 'space-between' }}
              className={css.blueCopyContainer}
              padding={{ top: 'small', left: 'medium', right: 'medium', bottom: 'small' }}>
              <Text className={css.stepFont}>
                <StringSubstitute
                  str={getString('cmdlineInfo.stepOneSub')}
                  vars={{
                    target: pullReqMetadata.target_branch
                  }}
                />
              </Text>
              <CopyButton
                className={css.copyIconContainer}
                content={stepOneCopy}
                icon={CodeIcon.Copy}
                color={Color.PRIMARY_7}
                iconProps={{ size: 14, color: Color.PRIMARY_7 }}
                background={Color.PRIMARY_1}
              />
            </Layout.Horizontal>
          </Layout.Horizontal>
          <Layout.Horizontal flex={{ alignItems: 'center', justifyContent: 'flex-start' }} padding={{ top: 'small' }}>
            <Text className={css.checkName} padding={{ right: 'small' }} font={{ variation: FontVariation.CARD_TITLE }}>
              {getString('stepNum', { num: 2 }).toUpperCase()}
            </Text>
            <Text color={Color.GREY_450} className={css.stepText} font={{ variation: FontVariation.BODY2 }}>
              {getString('cmdlineInfo.stepTwo')}
            </Text>
          </Layout.Horizontal>
          <Layout.Horizontal margin={{ left: 'small' }} padding={{ top: 'xsmall', left: 'xxxlarge' }}>
            <Layout.Horizontal
              flex={{ justifyContent: 'space-between' }}
              className={css.blueCopyContainer}
              padding={{ top: 'small', left: 'medium', right: 'medium', bottom: 'small' }}>
              <Text className={css.stepFont}>
                <StringSubstitute
                  str={getString('cmdlineInfo.stepTwoSub')}
                  vars={{
                    source: pullReqMetadata.source_branch
                  }}
                />
              </Text>
              <CopyButton
                className={css.copyIconContainer}
                content={stepTwoCopy}
                icon={CodeIcon.Copy}
                color={Color.PRIMARY_7}
                iconProps={{ size: 14, color: Color.PRIMARY_7 }}
                background={Color.PRIMARY_1}
              />
            </Layout.Horizontal>
          </Layout.Horizontal>

          <Layout.Horizontal flex={{ alignItems: 'center', justifyContent: 'flex-start' }} padding={{ top: 'small' }}>
            <Text className={css.checkName} padding={{ right: 'small' }} font={{ variation: FontVariation.CARD_TITLE }}>
              {getString('stepNum', { num: 3 }).toUpperCase()}
            </Text>
            <Text color={Color.GREY_450} className={css.stepText} font={{ variation: FontVariation.BODY2 }}>
              {getString('cmdlineInfo.stepThree')}
            </Text>
          </Layout.Horizontal>
          <Layout.Horizontal margin={{ left: 'small' }} padding={{ top: 'xsmall', left: 'xxxlarge' }}>
            <Layout.Horizontal
              flex={{ justifyContent: 'space-between' }}
              className={css.blueCopyContainer}
              padding={{ top: 'small', left: 'medium', right: 'medium', bottom: 'small' }}>
              <Text className={css.stepFont}>
                <StringSubstitute
                  str={getString('cmdlineInfo.stepThreeSub')}
                  vars={{
                    target: pullReqMetadata.target_branch
                  }}
                />
              </Text>
              <CopyButton
                className={css.copyIconContainer}
                content={stepThreeCopy}
                icon={CodeIcon.Copy}
                color={Color.PRIMARY_7}
                iconProps={{ size: 14, color: Color.PRIMARY_7 }}
                background={Color.PRIMARY_1}
              />
            </Layout.Horizontal>
          </Layout.Horizontal>
          <Layout.Horizontal flex={{ alignItems: 'center', justifyContent: 'flex-start' }} padding={{ top: 'small' }}>
            <Text className={css.checkName} padding={{ right: 'small' }} font={{ variation: FontVariation.CARD_TITLE }}>
              {getString('stepNum', { num: 4 }).toUpperCase()}
            </Text>
            <Text color={Color.GREY_450} className={css.stepText} font={{ variation: FontVariation.BODY2 }}>
              {getString('cmdlineInfo.stepFour')}
            </Text>
          </Layout.Horizontal>
          <Layout.Horizontal margin={{ left: 'small' }} padding={{ top: 'xsmall', left: 'xxxlarge' }}>
            <Layout.Horizontal flex={{ justifyContent: 'space-between' }}>
              <Text
                padding={{ left: 'tiny' }}
                color={Color.GREY_450}
                className={css.stepText}
                font={{ variation: FontVariation.BODY2 }}>
                {getString('cmdlineInfo.stepFourSub')}
              </Text>
            </Layout.Horizontal>
          </Layout.Horizontal>
          <Layout.Horizontal flex={{ alignItems: 'center', justifyContent: 'flex-start' }} padding={{ top: 'small' }}>
            <Text className={css.checkName} padding={{ right: 'small' }} font={{ variation: FontVariation.CARD_TITLE }}>
              {getString('stepNum', { num: 5 }).toUpperCase()}
            </Text>
            <Text color={Color.GREY_450} className={css.stepText} font={{ variation: FontVariation.BODY2 }}>
              {getString('cmdlineInfo.stepFive')}
            </Text>
          </Layout.Horizontal>
          <Layout.Horizontal margin={{ left: 'small' }} padding={{ top: 'xsmall', left: 'xxxlarge' }}>
            <Layout.Horizontal
              flex={{ justifyContent: 'space-between' }}
              className={css.blueCopyContainer}
              padding={{ top: 'small', left: 'medium', right: 'medium', bottom: 'small' }}>
              <Text className={css.stepFont}>
                <StringSubstitute
                  str={getString('cmdlineInfo.stepFiveSub')}
                  vars={{
                    source: pullReqMetadata.source_branch
                  }}
                />
              </Text>
              <CopyButton
                className={css.copyIconContainer}
                content={stepFiveCopy}
                icon={CodeIcon.Copy}
                color={Color.PRIMARY_7}
                iconProps={{ size: 14, color: Color.PRIMARY_7 }}
                background={Color.PRIMARY_1}
              />
            </Layout.Horizontal>
          </Layout.Horizontal>
        </Layout.Vertical>
      </Layout.Vertical>
    </Container>
  )
}

export default CommandLineInfo
