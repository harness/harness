import {
  Button,
  Container,
  FlexExpander,
  Layout,
  Color,
  Text,
  StringSubstitute,
  ButtonSize,
  ButtonVariation,
  Avatar
} from '@harness/uicore'
import React, { useMemo } from 'react'
import { useHistory } from 'react-router-dom'
import { useGet } from 'restful-react'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import type { TypesCommit, TypesRepository, TypesSignature } from 'services/code'
import { LIST_FETCHING_LIMIT, formatDate } from 'utils/Utils'
import css from './CommitInfo.module.scss'

const CommitInfo = (props: { repoMetadata: TypesRepository; commitRef: string }) => {
  const { repoMetadata, commitRef } = props
  const history = useHistory()
  const { getString } = useStrings()
  const { routes } = useAppContext()
  const { data: commits } = useGet<{ commits: TypesCommit[] }>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/commits`,
    queryParams: {
      limit: LIST_FETCHING_LIMIT,
      git_ref: commitRef || repoMetadata?.default_branch
    },
    lazy: !repoMetadata
  })

  const commitData = useMemo(() => {
    return commits?.commits?.filter(commit => commit.sha === commitRef)
  }, [commitRef, commits?.commits])

  return (
    <>
      {commitData && (
        <Container className={css.commitInfoContainer} padding={{ top: 'small' }}>
          <Container className={css.commitTitleContainer} color={Color.GREY_100}>
            <Layout.Horizontal className={css.alignContent} padding={{ right: 'medium' }}>
              <Text
                className={css.titleText}
                icon={'code-commit'}
                iconProps={{ size: 16 }}
                padding="medium"
                color="black">
                {commitData[0] ? commitData[0]?.title : ''}
              </Text>
              <FlexExpander />
              <Button
                size={ButtonSize.SMALL}
                variation={ButtonVariation.SECONDARY}
                text={getString('browseFiles')}
                onClick={() => {
                  history.push(
                    routes.toCODERepository({
                      repoPath: repoMetadata.path as string,
                      gitRef: commitRef
                    })
                  )
                }}
              />
            </Layout.Horizontal>
          </Container>
          <Container className={css.infoContainer}>
            <Layout.Horizontal className={css.alignContent} padding={{ left: 'small', right: 'medium' }}>
              <Avatar hoverCard={false} size="small" name={commitData[0] ? commitData[0].author?.identity?.name : ''} />
              <Text className={css.infoText} color={Color.BLACK}>
                {commitData[0] ? commitData[0].author?.identity?.name : ''}
              </Text>
              <Text
                font={{ size: 'small' }}
                padding={{ left: 'small', right: 'large', top: 'medium', bottom: 'medium' }}>
                {getString('commitsOn', {
                  date: commitData[0] ? formatDate((commitData[0].committer as TypesSignature).when as string) : ''
                })}
              </Text>
              <FlexExpander />
              <Text className={css.infoText} flex>
                <StringSubstitute
                  str={getString('commitString')}
                  vars={{
                    commit: (
                      <Text
                        className={css.infoText}
                        padding={{ left: 'small' }}
                        color={'black'}>{` ${commitRef.substring(0, 6)}`}</Text>
                    )
                  }}
                />
              </Text>
            </Layout.Horizontal>
          </Container>
        </Container>
      )}
    </>
  )
}

export default CommitInfo
