import React, { useState } from 'react'
import { Container, Text } from '@harnessio/uicore'
import { useStrings } from 'framework/strings'
import { useConfirmAct } from 'hooks/useConfirmAction'
import css from './RepositoryTypeButton.module.scss'

export enum RepositoryType {
  GITNESS = 'gitness',
  THIRDPARTY = 'thirdParty'
}

const RepositoryTypeButton = ({
  hasChange,
  onChange
}: {
  hasChange?: boolean
  onChange: (type: RepositoryType) => void
}) => {
  const { getString } = useStrings()
  const confirmSwitch = useConfirmAct()
  const [activeButton, setActiveButton] = useState(RepositoryType.GITNESS)

  const handleSwitch = (type: RepositoryType) => {
    const onConfirm = () => {
      setActiveButton(type)
      onChange(type)
    }
    if (hasChange) {
      confirmSwitch({
        title: getString('cde.create.unsaved.title'),
        message: getString('cde.create.unsaved.message'),
        action: async () => {
          onConfirm()
        }
      })
    } else {
      onConfirm()
    }
  }

  return (
    <Container className={css.splitButton}>
      <Text
        className={activeButton === RepositoryType.GITNESS ? css.active : ''}
        onClick={() => {
          handleSwitch(RepositoryType.GITNESS)
        }}>
        {getString('cde.create.gitnessRepositories')}
      </Text>
      <Text
        className={activeButton === RepositoryType.THIRDPARTY ? css.active : ''}
        onClick={() => {
          handleSwitch(RepositoryType.THIRDPARTY)
        }}>
        {getString('cde.create.thirdPartyGitRepositories')}
      </Text>
    </Container>
  )
}

export default RepositoryTypeButton
