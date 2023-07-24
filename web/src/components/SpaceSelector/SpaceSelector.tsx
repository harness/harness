import React, { useCallback, useEffect, useState } from 'react'
import { Button, Container, Heading, Icon, Layout, Text } from '@harness/uicore'
import cx from 'classnames'
import { Classes, Popover, Position } from '@blueprintjs/core'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useStrings } from 'framework/strings'
import { ButtonRoleProps } from 'utils/Utils'
import { useShowRequestError } from 'hooks/useShowRequestError'
import { TypesSpace, useGetSpace } from 'services/code'
import css from './SpaceSelector.module.scss'

interface SpaceSelectorProps {
  onSelect: (space: TypesSpace, isUserAction: boolean) => void
}

export const SpaceSelector: React.FC<SpaceSelectorProps> = ({ onSelect }) => {
  const { getString } = useStrings()
  const [selectedSpace, setSelectedSpace] = useState<TypesSpace | undefined>()
  const { space } = useGetRepositoryMetadata()
  const [opened, setOpened] = React.useState(false)
  const { data, error } = useGetSpace({ space_ref: space, lazy: !space })
  const selectSpace = useCallback(
    (_space: TypesSpace, isUserAction: boolean) => {
      setSelectedSpace(_space)
      onSelect(_space, isUserAction)
    },
    [onSelect]
  )

  useEffect(() => {
    if (space && !selectedSpace && data) {
      selectSpace(data, false)
    }
  }, [space, selectedSpace, data, onSelect, selectSpace])

  useShowRequestError(error)

  return (
    <Popover
      portalClassName={css.popoverPortal}
      targetClassName={css.popoverTarget}
      popoverClassName={css.popoverContent}
      position={Position.RIGHT}
      usePortal={false}
      transitionDuration={0}
      captureDismiss
      onInteraction={setOpened}
      isOpen={opened}>
      <Container
        className={cx(css.spaceSelector, { [css.selected]: opened })}
        {...ButtonRoleProps}
        onClick={() => setOpened(!opened)}>
        <Layout.Horizontal>
          <Container className={css.label}>
            <Layout.Vertical>
              <Container>
                <Text className={css.spaceLabel} icon="nav-project" iconProps={{ size: 12 }}>
                  {getString('space').toUpperCase()}
                </Text>
              </Container>
            </Layout.Vertical>
            <Text className={css.spaceName} lineClamp={1}>
              {selectedSpace ? selectedSpace.uid : getString('selectSpace')}
            </Text>
          </Container>
          <Container className={css.icon}>
            <Icon name="main-chevron-right" size={10} />
          </Container>
        </Layout.Horizontal>
      </Container>

      <Container padding="large">
        <Heading level={2}>
          {getString('spaces')}
          <Button text="close" className={Classes.POPOVER_DISMISS} />
        </Heading>
        <Container>
          <Layout.Vertical spacing="small">
            <Text
              className={Classes.POPOVER_DISMISS}
              {...ButtonRoleProps}
              onClick={() => selectSpace({ uid: 'root', path: 'root' }, true)}>
              Root Space
            </Text>
            <Text
              className={Classes.POPOVER_DISMISS}
              {...ButtonRoleProps}
              onClick={() => selectSpace({ uid: 'home', path: 'home' }, true)}>
              Home Space
            </Text>
          </Layout.Vertical>
        </Container>
      </Container>
    </Popover>
  )
}
