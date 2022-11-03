import React from 'react'
import { Container, Layout, Text, Color } from '@harness/uicore'

export function RepoMetadata() {
  return (
    <Container width="70%">
      <Layout.Horizontal spacing="large">
        <Text icon="dot" iconProps={{ size: 20, color: Color.BLUE_500 }}>
          Java
        </Text>
        <Text color={Color.GREY_200}>{' | '}</Text>
        <Text icon="git-new-branch">165</Text>
        <Text icon="git-branch-existing">123</Text>
        <Text icon="git-merge">432</Text>
      </Layout.Horizontal>
    </Container>
  )
}
