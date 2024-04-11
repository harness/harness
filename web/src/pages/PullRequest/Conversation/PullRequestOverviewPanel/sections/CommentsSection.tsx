import React from 'react'
import cx from 'classnames'
import { Color } from '@harnessio/design-system'
import { Button, ButtonSize, ButtonVariation, Container, Layout, SelectOption, Text } from '@harnessio/uicore'
import { useStrings } from 'framework/strings'
import type { TypesRepository, TypesPullReq, TypesViolation } from 'services/code'
import Success from '../../../../../icons/code-success.svg?url'
import Fail from '../../../../../icons/code-fail.svg?url'
import css from '../PullRequestOverviewPanel.module.scss'
import { PRCommentFilterType } from 'utils/Utils'
interface MergeSectionProps {
  repoMetadata: TypesRepository
  pullReqMetadata: TypesPullReq
  resolvedCommentArr?: TypesViolation
  requiresCommentApproval: boolean
  setActivityFilter: (val: SelectOption) => void
}
const CommentsSection = (props: MergeSectionProps) => {
  const { resolvedCommentArr, requiresCommentApproval, setActivityFilter } = props
  const resolvedComments = requiresCommentApproval && !resolvedCommentArr?.params ? true : false
  const { getString } = useStrings()

  return (
    <Container flex={{ justifyContent: 'space-between' }}>
      <Layout.Horizontal flex={{ align: 'center-center' }}>
        {resolvedComments ? (
          <img alt="success" width={27} height={27} src={Success} />
        ) : (
          <img alt="fail" width={27} height={27} src={Fail} />
        )}

        {resolvedComments ? (
          <Text color={Color.GREEN_800} className={css.sectionTitle} padding={{ left: 'medium' }}>
            {getString('allCommentsResolved')}
          </Text>
        ) : (
          <Layout.Vertical>
            <Text color={Color.RED_500} className={css.sectionTitle} padding={{ left: 'medium', bottom: 'xsmall' }}>
              {getString('unrsolvedComment')}
            </Text>
            <Text color={Color.GREY_450} className={css.sectionSubheader} padding={{ left: 'medium' }}>
              {getString('resolveComments', { n: resolvedCommentArr?.params })}
            </Text>
          </Layout.Vertical>
        )}
      </Layout.Horizontal>
      {!resolvedComments ? (
        <Button
          className={cx(css.blueText, css.buttonPadding)}
          variation={ButtonVariation.LINK}
          size={ButtonSize.SMALL}
          text={getString('view')}
          padding={{ bottom: 'medium' }}
          iconProps={{ size: 10, margin: { left: 'xsmall' } }}
          onClick={() => {
            setActivityFilter({
              label: getString('unrsolvedComment'),
              value: PRCommentFilterType.UNRESOLVED_COMMENTS
            })
            document.querySelectorAll('.bp3-input[value="Active"]')[0].scrollIntoView({ behavior: 'smooth' })
          }}
        />
      ) : null}
    </Container>
  )
}

export default CommentsSection
