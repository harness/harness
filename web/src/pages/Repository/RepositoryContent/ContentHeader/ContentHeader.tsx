import React, { useEffect, useMemo, useState } from 'react'
import {
  Container,
  Layout,
  Button,
  FlexExpander,
  TextInput,
  ButtonVariation,
  Text,
  DropDown,
  Icon,
  Color
} from '@harness/uicore'
import ReactJoin from 'react-join'
import { Link, useHistory } from 'react-router-dom'
import { uniq } from 'lodash-es'
import { useGet } from 'restful-react'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import type { RepoBranch } from 'services/scm'
import type { RepositoryDTO } from 'types/SCMTypes'
import css from './ContentHeader.module.scss'

interface ContentHeaderProps {
  gitRef?: string
  resourcePath?: string
  repoMetadata: RepositoryDTO
}

export function ContentHeader({ repoMetadata, gitRef, resourcePath = '' }: ContentHeaderProps): JSX.Element {
  const { getString } = useStrings()
  const { routes } = useAppContext()
  const history = useHistory()
  // const [searchTerm, setSearchTerm] = useState('README.md')
  const [branch, setBranch] = useState(gitRef || repoMetadata.defaultBranch)
  const { data } = useGet<RepoBranch[]>({
    path: `/api/v1/repos/${repoMetadata.path}/+/branches?sort=date&direction=desc&per_page=20&page=1`
  })
  // defaultBranches is computed using repository default branch, and gitRef in URL, if it exists
  const defaultBranches = useMemo(
    () => [repoMetadata.defaultBranch].concat(gitRef ? gitRef : []),
    [repoMetadata, gitRef]
  )
  const [branches, setBranches] = useState(uniq(defaultBranches.map(_branch => ({ label: _branch, value: _branch }))))

  useEffect(() => {
    if (data?.length) {
      setBranches(
        uniq(defaultBranches.concat(data.map(e => e.name) as string[])).map(_branch => ({
          label: _branch,
          value: _branch
        }))
      )
    }
  }, [data, defaultBranches])

  return (
    <Container className={css.folderHeader}>
      <Layout.Horizontal spacing="medium">
        <DropDown
          icon="git-branch"
          value={branch}
          items={branches}
          onChange={({ value: switchBranch }) => {
            setBranch(switchBranch as string)
            history.push(
              routes.toSCMRepository({
                repoPath: repoMetadata.path,
                gitRef: switchBranch as string,
                resourcePath // TODO: Handle 404 when resourcePath does not exist in newly switched branch
              })
            )
          }}
          popoverClassName={css.branchDropdown}
        />
        <Container>
          <Layout.Horizontal spacing="small">
            <Link to={routes.toSCMRepository({ repoPath: repoMetadata.path })}>
              <Icon name="main-folder" />
            </Link>
            <Text color={Color.GREY_900}>/</Text>
            <ReactJoin separator={<Text color={Color.GREY_900}>/</Text>}>
              {resourcePath.split('/').map((path, index, paths) => {
                const pathAtIndex = paths.slice(0, index + 1).join('/')

                return (
                  <Link
                    key={path + index}
                    to={routes.toSCMRepository({ repoPath: repoMetadata.path, gitRef, resourcePath: pathAtIndex })}>
                    <Text color={Color.GREY_900}>{path}</Text>
                  </Link>
                )
              })}
            </ReactJoin>
          </Layout.Horizontal>
        </Container>

        {/* <TextInput
          placeholder="Search folder or file"
          autoFocus
          onFocus={event => event.target.select()}
          value={searchTerm}
          onInput={event => {
            setSearchTerm(event.currentTarget.value)
          }}
        /> */}
        <FlexExpander />
        <Button text={getString('clone')} variation={ButtonVariation.SECONDARY} icon="main-clone" />
        <Button text={getString('newFile')} variation={ButtonVariation.PRIMARY} icon="plus" />
      </Layout.Horizontal>
    </Container>
  )
}
