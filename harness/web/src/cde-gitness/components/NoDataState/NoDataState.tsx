import React from 'react'
import { Container, Text, Button } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import NoRegionIcon from 'cde-gitness/assests/noRegion.svg?url'
import { useStrings } from 'framework/strings'
import NoMachineIcon from '../../../icons/NoMachine.svg?url'
import css from './NoDataState.module.scss'

export interface NoDataStateProps {
  onButtonClick?: () => void
  type: 'region' | 'machine'
  setIsOpen?: (key: boolean) => void
}

const NoDataState: React.FC<NoDataStateProps> = ({ onButtonClick, type, setIsOpen }) => {
  const { getString } = useStrings()
  return (
    <Container className={css.noDataContainer} padding="large" flex={{ align: 'center-center' }}>
      <Container flex>
        <img src={type === 'region' ? NoRegionIcon : NoMachineIcon} className={css.icon} />
        <Container className={css.contentWithIcon}>
          <Text font={{ variation: FontVariation.H4 }}>
            {type === 'region'
              ? getString('cde.gitspaceInfraHome.noRegionConfigured')
              : getString('cde.gitspaceInfraHome.noMachineAvailable')}
          </Text>
          <Text font={{ variation: FontVariation.BODY }} padding={{ top: 'small' }} color={Color.GREY_500}>
            {type === 'region'
              ? getString('cde.gitspaceInfraHome.noRegionConfiguredText')
              : getString('cde.gitspaceInfraHome.addMachineType')}
          </Text>
          <Container padding={{ top: 'large' }}>
            <Button
              icon="plus"
              text={
                type === 'region'
                  ? getString('cde.gitspaceInfraHome.newRegion')
                  : getString('cde.gitspaceInfraHome.newMachine')
              }
              intent="primary"
              onClick={type === 'region' ? onButtonClick : setIsOpen ? () => setIsOpen(true) : undefined}
            />
          </Container>
        </Container>
      </Container>
    </Container>
  )
}

export default NoDataState
