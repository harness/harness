import React from 'react'
import { Button, Color, Container, FlexExpander, Icon, Layout, Text } from '@harness/uicore'
import { useStrings } from 'framework/strings'
import css from './PullRequestStatusInfo.module.scss'

export const PullRequestStatusInfo: React.FC = () => {
  const { getString } = useStrings()

  return (
    <Container className={css.main} padding="xlarge">
      <Layout.Vertical spacing="xlarge">
        <Container>
          <Layout.Horizontal spacing="medium" flex={{ alignItems: 'center' }}>
            <Icon name="tick-circle" size={28} color={Color.GREEN_700} />
            <Container>
              <Text className={css.heading}>{getString('pr.branchHasNoConflicts')}</Text>
              <Text className={css.sub}>{getString('pr.prCanBeMerged')}</Text>
            </Container>
            <FlexExpander />
          </Layout.Horizontal>
        </Container>
        <Container>
          <Button className={css.btn} text={getString('pr.mergePR')} />
        </Container>
      </Layout.Vertical>
    </Container>
  )
}
