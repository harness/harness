import React, { Dispatch, SetStateAction } from 'react'
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
import type { TypesRepository } from 'services/code'

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
  selectedLanguage?: string
  setLanguage: Dispatch<SetStateAction<string | undefined>>
}

const KeywordSearchFilters: React.FC<KeywordSearchFiltersProps> = ({
  isRepoLevelSearch,
  selectedLanguage,
  selectedRepositories,
  setLanguage,
  setRepositories
}) => {
  const { getString } = useStrings()
  const space = useGetSpaceParam()

  const { data } = useGet<TypesRepository[]>({
    path: `/api/v1/spaces/${space}/+/repos`,
    debounce: 500,
    lazy: isRepoLevelSearch
  })

  const repositoryOptions =
    data?.map(repository => ({
      label: String(repository.uid),
      value: String(repository.path)
    })) || []

  return (
    <div className={css.filtersCtn}>
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
        <DropDown
          className={css.multiSelect}
          value={selectedLanguage}
          placeholder={getString('selectLanguagePlaceholder')}
          onChange={option => setLanguage(String(option.value))}
          items={languageOptions}
        />
      </Container>
      <Button
        variation={ButtonVariation.LINK}
        text={getString('clear')}
        onClick={() => {
          setRepositories([])
          setLanguage(undefined)
        }}
      />
    </div>
  )
}

export default KeywordSearchFilters
