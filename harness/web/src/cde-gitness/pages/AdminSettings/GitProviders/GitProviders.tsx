import React, { useEffect, useState, useMemo } from 'react'
import { Card, Text, Layout, Checkbox, Container } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { useFormikContext, getIn } from 'formik'
import { useStrings } from 'framework/strings'
import type { StringsMap } from 'framework/strings/stringTypes'
import { scmOptionsCDE, SCMType } from 'cde-gitness/pages/GitspaceCreate/CDECreateGitspace'
import type { AdminSettingsFormValues } from '../utils/adminSettingsUtils'
import css from './GitProviders.module.scss'

type Provider = { name: string; labelKey: string; icon: string }
type ProviderSection = { titleKey: keyof StringsMap; providers: Provider[] }

const providerSectionsTemplate: ProviderSection[] = [
  { titleKey: 'exportSpace.harness', providers: [] },
  { titleKey: 'cde.settings.providers.github', providers: [] },
  { titleKey: 'cde.settings.providers.gitlab', providers: [] },
  { titleKey: 'cde.settings.providers.bitbucket', providers: [] },
  { titleKey: 'cde.settings.other', providers: [] }
]

const sectionMapping: Record<string, string> = {
  harness_code: 'exportSpace.harness',
  github: 'cde.settings.providers.github',
  github_enterprise: 'cde.settings.providers.github',
  gitlab: 'cde.settings.providers.gitlab',
  gitlab_on_prem: 'cde.settings.providers.gitlab',
  bitbucket: 'cde.settings.providers.bitbucket',
  bitbucket_server: 'cde.settings.providers.bitbucket',
  unknown: 'cde.settings.other'
}

const GitProviders: React.FC = () => {
  const { getString } = useStrings()
  const { values, setFieldValue } = useFormikContext<AdminSettingsFormValues>()
  const [selectAllChecked, setSelectAllChecked] = useState(true)

  const providerSections: ProviderSection[] = useMemo(() => {
    const sections: ProviderSection[] = structuredClone(providerSectionsTemplate)

    scmOptionsCDE.forEach((provider: SCMType) => {
      const sectionKey = sectionMapping[provider.value]
      const section = sections.find((s: ProviderSection) => s.titleKey === sectionKey)

      if (section) {
        section.providers.push({ name: provider.value, labelKey: provider.name, icon: provider.icon })
      }
    })

    return sections
  }, [])

  useEffect(() => {
    if (values.gitProviders) {
      const allProviders = scmOptionsCDE.map(p => p.value)
      const allSelected = allProviders.every(provider => values.gitProviders[provider])
      setSelectAllChecked(allSelected)
    }
  }, [values.gitProviders])

  const handleSelectAll = (checked: boolean) => {
    setSelectAllChecked(checked)
    const newGitProviderValues = { ...values.gitProviders }
    scmOptionsCDE.forEach(provider => {
      newGitProviderValues[provider.value] = checked
    })
    setFieldValue('gitProviders', newGitProviderValues)
  }

  return (
    <div className={css.gitProvidersContainer}>
      <Layout.Vertical spacing="small">
        <Text font={{ variation: FontVariation.H5 }}>{getString('cde.settings.availableGitProviders')}</Text>
        <Text font={{ variation: FontVariation.BODY }} color={Color.GREY_500}>
          {getString('cde.settings.gitProvidersDescription')}
        </Text>
      </Layout.Vertical>

      <Card className={css.mainContentCard}>
        <Layout.Vertical spacing="medium">
          <Checkbox
            checked={selectAllChecked}
            className={css.checkbox}
            labelElement={
              <Layout.Horizontal spacing="small">
                <Text font={{ variation: FontVariation.BODY2 }} color={Color.GREY_700}>
                  {getString('cde.settings.selectAllGitProviders')}
                </Text>
              </Layout.Horizontal>
            }
            onChange={event => handleSelectAll(event.currentTarget.checked)}
          />

          <Layout.Vertical spacing="medium">
            {providerSections.map((section: ProviderSection) => {
              return (
                <Layout.Vertical spacing="medium" key={section.titleKey}>
                  <Text font={{ variation: FontVariation.BODY }} color={Color.GREY_500}>
                    {getString(section.titleKey)}
                  </Text>
                  <Layout.Horizontal spacing="large">
                    {section.providers.map((provider: Provider) => (
                      <Container className={css.providerCheckbox} key={provider.name}>
                        <Checkbox
                          className={css.checkbox}
                          checked={getIn(values.gitProviders, provider.name)}
                          labelElement={
                            <Layout.Horizontal spacing="small" flex={{ alignItems: 'center' }}>
                              <img src={provider.icon} alt={provider.labelKey} height={24} width={24} />
                              <Text font={{ variation: FontVariation.BODY2 }} color={Color.GREY_700}>
                                {provider.labelKey}
                              </Text>
                            </Layout.Horizontal>
                          }
                          onChange={event =>
                            setFieldValue(`gitProviders.${provider.name}`, event.currentTarget.checked)
                          }
                        />
                      </Container>
                    ))}
                  </Layout.Horizontal>
                </Layout.Vertical>
              )
            })}
          </Layout.Vertical>
        </Layout.Vertical>
      </Card>
    </div>
  )
}

export default GitProviders
