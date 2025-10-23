import React from 'react'
import { Container, Text, CardSelect, Layout } from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { Color, FontVariation } from '@harnessio/design-system'
import { useStrings } from 'framework/strings'
import css from './PublicPrivateRegistrySelect.module.scss'

export type AccessType = 'public' | 'private'

export interface RegistryAccessCard {
  type: AccessType
  title: string
  icon: string
  size: number
}

export interface PublicPrivateRegistrySelectProps {
  selected: AccessType
  onChange: (accessType: AccessType) => void
  className?: string
}

function getRegistryAccessCards(getString: ReturnType<typeof useStrings>['getString']): RegistryAccessCard[] {
  return [
    {
      type: 'public',
      title: getString('cde.settings.images.public'),
      icon: 'globe',
      size: 16
    },
    {
      type: 'private',
      title: getString('cde.settings.images.private'),
      icon: 'lock',
      size: 16
    }
  ]
}

export const PublicPrivateRegistrySelect: React.FC<PublicPrivateRegistrySelectProps> = ({ selected, onChange }) => {
  const { getString } = useStrings()

  const cards = getRegistryAccessCards(getString)
  const selectedCard = cards.find(card => card.type === selected)

  const handleCardChange = (item: RegistryAccessCard) => {
    onChange(item.type)
  }

  return (
    <CardSelect
      data={cards}
      cornerSelected
      className={css.registrySelectWrapper}
      renderItem={(item: RegistryAccessCard) => {
        const iconProps = {
          name: item.icon as any,
          size: item.size,
          color: selected === item.type ? Color.PRIMARY_7 : Color.GREY_600
        }

        return (
          <Layout.Horizontal flex spacing={'small'}>
            <Icon {...iconProps} />
            <Container>
              <Text
                font={{ variation: FontVariation.FORM_TITLE }}
                color={selected === item.type ? Color.PRIMARY_7 : Color.GREY_800}>
                {item.title}
              </Text>
            </Container>
          </Layout.Horizontal>
        )
      }}
      selected={selectedCard}
      onChange={handleCardChange}
    />
  )
}
