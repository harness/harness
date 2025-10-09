import React, { Dispatch, SetStateAction, useMemo, useState } from 'react'
import {
  Container,
  Text,
  type SelectOption,
  MultiSelectDropDown,
  Button,
  ButtonVariation,
  DropDown
} from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { useGet } from 'restful-react'
import { useStrings } from 'framework/strings'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import type { RepoRepositoryOutput } from 'services/code'
import { ScopeLevelEnum } from 'utils/Utils'
import css from './Search.module.scss'

const languageOptions = [
  { label: 'JavaScript', value: 'javascript' },
  { label: 'Python', value: 'python' },
  { label: 'Java', value: 'java' },
  { label: 'C#', value: 'csharp' },
  { label: 'PHP', value: 'php' },
  { label: 'TypeScript', value: 'typescript' },
  { label: 'C++', value: 'cpp' },
  { label: 'C', value: 'c' },
  { label: 'Ruby', value: 'ruby' },
  { label: 'Go', value: 'go' },
  { label: 'Swift', value: 'swift' },
  { label: 'Kotlin', value: 'kotlin' },
  { label: 'Rust', value: 'rust' },
  { label: 'Scala', value: 'scala' }
]

interface KeywordSearchFiltersProps {
  isRepoLevelSearch?: boolean
  selectedRepositories: SelectOption[]
  setRepositories: Dispatch<SetStateAction<SelectOption[]>>
  selectedLanguages: SelectOption[]
  setLanguages: Dispatch<SetStateAction<SelectOption[]>>
  recursiveSearchEnabled: boolean
  setRecursiveSearchEnabled: React.Dispatch<React.SetStateAction<boolean>>
  curScopeLabel: SelectOption | undefined
  setCurScopeLabel: React.Dispatch<React.SetStateAction<SelectOption | undefined>>
}

const KeywordSearchFilters: React.FC<KeywordSearchFiltersProps> = ({
  isRepoLevelSearch,
  selectedLanguages,
  selectedRepositories,
  setLanguages,
  setRepositories,
  recursiveSearchEnabled,
  setRecursiveSearchEnabled,
  curScopeLabel,
  setCurScopeLabel
}) => {
  const { getString } = useStrings()
  const space = useGetSpaceParam()
  const accId = space?.split('/')[0]
  const orgId = space?.split('/')[1]

  const projectId = space?.split('/')[2]
  const enabledRecursive = useMemo(() => recursiveSearchEnabled, [recursiveSearchEnabled])
  const { data } = useGet<RepoRepositoryOutput[]>({
    path: `/api/v1/spaces/${space}/+/repos`,
    debounce: 500,

    lazy: isRepoLevelSearch,
    queryParams: {
      recursive: enabledRecursive
    }
  })

  const repositoryOptions =
    data?.map(repository => ({
      label: String(repository.identifier),
      value: String(repository.path)
    })) || []

  const scopeOption = [
    accId && !orgId
      ? {
          label: getString('searchScope.allScopes'),
          value: ScopeLevelEnum.ALL
        }
      : null,
    accId && !orgId ? { label: getString('searchScope.accOnly'), value: ScopeLevelEnum.CURRENT } : null,
    orgId ? { label: getString('searchScope.orgAndProj'), value: ScopeLevelEnum.ALL } : null,
    orgId ? { label: getString('searchScope.orgOnly'), value: ScopeLevelEnum.CURRENT } : null
  ].filter(Boolean) as SelectOption[]

  const [scopeLabel, setScopeLabel] = useState<SelectOption>(curScopeLabel ? curScopeLabel : scopeOption[0])
  return (
    <div className={css.filtersCtn}>
      {projectId || isRepoLevelSearch ? null : (
        <>
          <Container>
            <Text font={{ variation: FontVariation.SMALL_SEMI }} color={Color.GREY_600} margin={{ bottom: 'xsmall' }}>
              {getString('searchScope.title')}
            </Text>
            <DropDown
              placeholder={scopeLabel.label}
              className={css.dropdown}
              value={scopeLabel}
              items={scopeOption}
              onChange={e => {
                if (e.value === ScopeLevelEnum.ALL) {
                  setRecursiveSearchEnabled(true)
                } else if (e.value === ScopeLevelEnum.CURRENT) {
                  setRecursiveSearchEnabled(false)
                }
                setScopeLabel(e)
                setCurScopeLabel(e)
              }}
              popoverClassName={css.branchDropdown}
            />
          </Container>
        </>
      )}

      {isRepoLevelSearch ? null : (
        <Container>
          <Text font={{ variation: FontVariation.SMALL_SEMI }} color={Color.GREY_600} margin={{ bottom: 'xsmall' }}>
            {getString('pageTitle.repository')}
          </Text>
          <MultiSelectDropDown
            className={css.multiSelect}
            value={selectedRepositories}
            placeholder={selectedRepositories.length ? '' : getString('selectRepositoryPlaceholder')}
            onChange={setRepositories}
            items={repositoryOptions}
          />
        </Container>
      )}
      <Container>
        <Text font={{ variation: FontVariation.SMALL_SEMI }} color={Color.GREY_600} margin={{ bottom: 'xsmall' }}>
          {getString('language')}
        </Text>
        <MultiSelectDropDown
          className={css.multiSelect}
          value={selectedLanguages}
          placeholder={selectedLanguages.length ? '' : getString('selectLanguagePlaceholder')}
          onChange={setLanguages}
          items={languageOptions}
        />
      </Container>
      <Container>
        <Button
          variation={ButtonVariation.LINK}
          text={getString('clear')}
          onClick={() => {
            setRepositories([])
            setLanguages([])
          }}
        />
      </Container>
    </div>
  )
}

export default KeywordSearchFilters
