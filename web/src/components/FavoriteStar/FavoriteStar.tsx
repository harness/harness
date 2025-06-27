/*
 * Copyright 2023 Harness Inc. All rights reserved.
 * Use of this source code is governed by the PolyForm Shield 1.0.0 license
 * that can be found in the licenses directory at the root of this repository, also available at
 * https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.
 */

import { Color } from '@harnessio/design-system'
import { Icon } from '@harnessio/icons'
import { PopoverInteractionKind } from '@blueprintjs/core'
import React, { useEffect, useState } from 'react'
import { Text, Utils, useToaster } from '@harnessio/uicore'
import { useMutate } from 'restful-react'
import classNames from 'classnames'
import { useStrings } from 'framework/strings'
import type { EnumResourceType, TypesFavoriteResource } from 'services/code'
import css from './FavoriteStar.module.scss'

interface FavoriteStarProps {
  resourceId: number
  resourceType: EnumResourceType
  isFavorite?: boolean
  activeClassName?: string
  className?: string
  onChange?: (favorite: boolean) => void
}

const FavoriteStar: React.FC<FavoriteStarProps> = props => {
  const { activeClassName = '', isFavorite: isFavoriteFromProps, onChange, resourceId, resourceType } = props
  const [isFavorite, setIsFavorite] = useState<boolean>(Boolean(isFavoriteFromProps))
  const [apiInProgress, setAPIInProgress] = useState<boolean>(false)
  const { showError } = useToaster()
  const { getString } = useStrings()

  useEffect(() => {
    if (Boolean(isFavoriteFromProps) !== isFavorite) {
      setIsFavorite(Boolean(isFavoriteFromProps))
    }
  }, [isFavoriteFromProps])

  const { mutate: createFavoritePromise } = useMutate<TypesFavoriteResource>({
    verb: 'POST',
    path: `/api/v1/user/favorite`
  })
  const { mutate: deleteFavoritePromise } = useMutate<TypesFavoriteResource>({
    verb: 'DELETE',
    path: `/api/v1/user/favorite`
  })

  const toggleFavorite = async () => {
    setAPIInProgress(true)
    const promise = isFavorite ? deleteFavoritePromise : createFavoritePromise
    await promise({
      resource_id: resourceId,
      resource_type: resourceType
    })
      .then(() => {
        setIsFavorite(!isFavorite)
        onChange?.(!isFavorite)
      })
      .catch(() => {
        showError(getString(isFavorite ? 'favorite.errorUnFavorite' : 'favorite.errorFavorite'))
        setIsFavorite(isFavorite)
      })
      .finally(() => {
        setAPIInProgress(false)
      })
  }

  const handleClick = (e: React.MouseEvent<Element, MouseEvent>): void => {
    e.stopPropagation()

    if (!apiInProgress) {
      toggleFavorite()
    }
  }

  return (
    <Utils.WrapOptionalTooltip
      tooltip={<Text padding="small">{isFavorite ? getString('favorite.remove') : getString('favorite.add')}</Text>}
      tooltipProps={{ interactionKind: PopoverInteractionKind.HOVER }}>
      <Icon
        name={isFavorite ? 'star' : 'star-empty'}
        color={isFavorite ? Color.YELLOW_700 : Color.GREY_400}
        size={24}
        onClick={handleClick}
        className={classNames(css.star, props.className, { [activeClassName]: isFavorite })}
        padding="xsmall"
      />
    </Utils.WrapOptionalTooltip>
  )
}

export default FavoriteStar
