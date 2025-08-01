import React from 'react'
import { Text, Layout } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { useStrings } from 'framework/strings'
import { DefaultGitspaceImage } from 'cde-gitness/components/DefaultGitspaceImage/DefaultGitspaceImage'
import { AllowedImagePaths } from 'cde-gitness/components/AllowedImagePaths/AllowedImagePaths'
import type { TypesGitspaceSettingsResponse } from 'services/cde'
import css from './GitspaceImages.module.scss'

interface GitspaceImagesProps {
  settings: TypesGitspaceSettingsResponse | null
}

const GitspaceImages: React.FC<GitspaceImagesProps> = ({ settings }: GitspaceImagesProps) => {
  const { getString } = useStrings()

  return (
    <div className={css.container}>
      <Layout.Vertical spacing="small">
        <Text font={{ variation: FontVariation.H5 }}>{getString('cde.settings.images.manageGitspaceImages')}</Text>
        <Text font={{ variation: FontVariation.BODY }} color={Color.GREY_500}>
          {getString('cde.settings.images.manageGitspaceImagesDescription')}
        </Text>
      </Layout.Vertical>

      <DefaultGitspaceImage settings={settings} />
      <AllowedImagePaths />
    </div>
  )
}

export default GitspaceImages
