import React from 'react'
import { Button, ButtonVariation, Layout, Text } from '@harnessio/uicore'
import { useStrings } from 'framework/strings'
import css from './NoMachineCard.module.scss'

function NoMachineCard({ setIsOpen }: { setIsOpen: (key: boolean) => void }) {
  const { getString } = useStrings()
  return (
    <Layout.Vertical spacing={'normal'}>
      <Text className={css.noMachineText}>{getString('cde.gitspaceInfraHome.noMachineAvailable')}</Text>
      <Text className={css.addMachineText}>{getString('cde.gitspaceInfraHome.addMachineType')}</Text>
      <Button
        text={getString('cde.gitspaceInfraHome.newMachine')}
        icon="plus"
        variation={ButtonVariation.PRIMARY}
        className={css.addMachineBtn}
        onClick={() => setIsOpen(true)}
      />
    </Layout.Vertical>
  )
}

export default NoMachineCard
