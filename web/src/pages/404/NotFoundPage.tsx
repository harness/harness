import React from 'react'
import { Container } from '@harness/uicore'
import { useStrings } from '../../framework/strings'

//
// TODO Build this 404 page in UICore
//
export const NotFoundPage: React.FC = () => {
  const { getString } = useStrings()
  return <Container padding="xlarge">{getString('pageNotFound')}</Container>
}
