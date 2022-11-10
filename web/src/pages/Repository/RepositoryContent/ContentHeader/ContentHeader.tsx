import React, { ChangeEvent, useEffect, useMemo, useState } from 'react'
import {
  Container,
  Layout,
  Button,
  FlexExpander,
  ButtonVariation,
  Text,
  DropDown,
  Icon,
  Color,
  SelectOption
} from '@harness/uicore'
import ReactJoin from 'react-join'
import { Link, useHistory } from 'react-router-dom'
import { uniq } from 'lodash-es'
import { useGet } from 'restful-react'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import type { RepoBranch, TypesRepository } from 'services/scm'
import { BRANCH_PER_PAGE } from 'utils/Utils'
import { CloneButtonTooltip } from 'components/CloneButtonTooltip/CloneButtonTooltip'
import { GitIcon } from 'utils/GitUtils'
import css from './ContentHeader.module.scss'

interface ContentHeaderProps {
  gitRef?: string
  resourcePath?: string
  repoMetadata: TypesRepository
}

export function ContentHeader({ repoMetadata, gitRef, resourcePath = '' }: ContentHeaderProps) {
  const { getString } = useStrings()
  const { routes } = useAppContext()
  const history = useHistory()
  const [query, setQuery] = useState('')
  const [activeBranch, setActiveBranch] = useState(gitRef || repoMetadata.defaultBranch)
  const { data, loading } = useGet<RepoBranch[]>({
    path: `/api/v1/repos/${repoMetadata.path}/+/branches`,
    queryParams: { sort: 'date', direction: 'desc', per_page: BRANCH_PER_PAGE, page: 1, query }
  })
  // defaultBranches is computed using repository default branch, and gitRef in URL, if it exists
  const defaultBranches = useMemo(
    () => [repoMetadata.defaultBranch].concat(gitRef ? gitRef : []),
    [repoMetadata, gitRef]
  )
  const [branches, setBranches] = useState<SelectOption[]>(
    uniq(defaultBranches.map(_branch => ({ label: _branch, value: _branch }))) as SelectOption[]
  )

  useEffect(() => {
    if (data?.length) {
      setBranches(
        uniq(defaultBranches.concat(data.map(e => e.name) as string[])).map(_branch => ({
          label: _branch,
          value: _branch
        })) as SelectOption[]
      )
    }
  }, [data, defaultBranches])

  return (
    <Container className={css.main}>
      <Layout.Horizontal spacing="medium">
        <DropDown
          icon={GitIcon.BRANCH}
          value={activeBranch}
          items={branches}
          {...{
            inputProps: {
              leftElement: (
                <Icon name={loading ? 'steps-spinner' : 'thinner-search'} size={12} color={Color.GREY_500} />
              ),
              placeholder: getString('search'),
              onInput: (event: ChangeEvent<HTMLInputElement>) => {
                if (event.target.value !== query) {
                  setQuery(event.target.value)
                }
              },
              onBlur: (event: ChangeEvent<HTMLInputElement>) => {
                setTimeout(() => {
                  setQuery(event.target.value || '')
                }, 250)
              }
            }
          }}
          onChange={({ value: switchBranch }) => {
            setActiveBranch(switchBranch as string)
            history.push(
              routes.toSCMRepository({
                repoPath: repoMetadata.path as string,
                gitRef: switchBranch as string,
                resourcePath // TODO: Handle 404 when resourcePath does not exist in newly switched branch
              })
            )
          }}
          popoverClassName={css.branchDropdown}
        />
        <Container>
          <Layout.Horizontal spacing="small">
            <Link to={routes.toSCMRepository({ repoPath: repoMetadata.path as string, gitRef })}>
              <Icon name="main-folder" />
            </Link>
            <Text color={Color.GREY_900}>/</Text>
            <ReactJoin separator={<Text color={Color.GREY_900}>/</Text>}>
              {resourcePath.split('/').map((_path, index, paths) => {
                const pathAtIndex = paths.slice(0, index + 1).join('/')

                return (
                  <Link
                    key={_path + index}
                    to={routes.toSCMRepository({
                      repoPath: repoMetadata.path as string,
                      gitRef,
                      resourcePath: pathAtIndex
                    })}>
                    <Text color={Color.GREY_900}>{_path}</Text>
                  </Link>
                )
              })}
            </ReactJoin>
          </Layout.Horizontal>
        </Container>
        <FlexExpander />
        <Button
          text={getString('clone')}
          variation={ButtonVariation.SECONDARY}
          icon="main-clone"
          iconProps={{ size: 10 }}
          tooltip={<CloneButtonTooltip httpsURL={repoMetadata.gitUrl as string} />}
          tooltipProps={{
            interactionKind: 'click',
            minimal: true,
            position: 'bottom-right'
          }}
        />
        <Button
          text={getString('newFile')}
          icon="plus"
          iconProps={{ size: 10 }}
          variation={ButtonVariation.PRIMARY}
          onClick={() => {
            history.push(
              routes.toSCMRepositoryFileEdit({
                repoPath: repoMetadata.path as string,
                resourcePath,
                gitRef: gitRef || (repoMetadata.defaultBranch as string)
              })
            )
          }}
        />
      </Layout.Horizontal>
    </Container>
  )
}
