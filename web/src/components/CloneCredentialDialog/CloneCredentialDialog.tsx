import React, { useEffect, useState } from 'react'
import {
  Button,
  ButtonVariation,
  Container,
  Dialog,
  FlexExpander,
  FontVariation,
  Layout,
  Text,
  useToaster
} from '@harness/uicore'
import { useStrings } from 'framework/strings'
import { CopyButton } from 'components/CopyButton/CopyButton'
import { CodeIcon } from 'utils/GitUtils'
import { useAppContext } from 'AppContext'
import { generateAlphaNumericHash } from 'utils/Utils'
import css from './CloneCredentialDialog.module.scss'
import { useHistory } from 'react-router-dom'

interface CloneCredentialDialogProps {
  setFlag: (val: boolean) => void
  flag: boolean
}

const CloneCredentialDialog = (props: CloneCredentialDialogProps) => {
  const { setFlag, flag } = props
  const history = useHistory()
  const { getString } = useStrings()
  const { hooks, currentUser,currentUserProfileURL } = useAppContext()
  const [token, setToken] = useState('')
  const { showError } = useToaster()
  const hash = generateAlphaNumericHash(6)

  const tokenData = hooks?.useGenerateToken(hash, currentUser.uid, flag)
  useEffect(() => {
    if (tokenData) {
      if (tokenData && tokenData?.status !== 400) {
        setToken(tokenData?.data)
      } else if (tokenData?.status === 400 && flag) {
        setToken('N/A')
        showError(tokenData?.data?.message || tokenData?.message)
      }
    }
  }, [flag, tokenData])
  return (
    <Dialog
      isOpen={flag}
      enforceFocus={false}
      onClose={() => {
        setFlag(false)
      }}
      title={
        <Text font={{ variation: FontVariation.H3 }} icon={'success-tick'} iconProps={{ size: 26 }}>
          {getString('getMyCloneTitle')}
        </Text>
      }
      style={{ width: 490, maxHeight: '95vh', overflow: 'auto' }}>
      <Layout.Vertical width={380}>
        <Text padding={{ bottom: 'small' }} font={{ variation: FontVariation.FORM_LABEL, size: 'small' }}>
          {getString('userName')}
        </Text>
        <Container padding={{ bottom: 'medium' }}>
          <Layout.Horizontal className={css.layout}>
            <Text className={css.url}>{currentUser.display_name}</Text>
            <FlexExpander />
            <CopyButton
              content={currentUser.display_name}
              id={css.cloneCopyButton}
              icon={CodeIcon.Copy}
              iconProps={{ size: 14 }}
            />
          </Layout.Horizontal>
        </Container>
        <Text padding={{ bottom: 'small' }} font={{ variation: FontVariation.FORM_LABEL, size: 'small' }}>
          {getString('passwordApi')}
        </Text>

        <Container padding={{ bottom: 'medium' }}>
          <Layout.Horizontal className={css.layout}>
            <Text className={css.url}>{token}</Text>
            <FlexExpander />
            <CopyButton content={token} id={css.cloneCopyButton} icon={CodeIcon.Copy} iconProps={{ size: 14 }} />
          </Layout.Horizontal>
        </Container>
        <Text padding={{ bottom: 'medium' }} font={{ variation: FontVariation.BODY2_SEMI, size: 'small' }}>
          {getString('cloneText')}
        </Text>
        <Button onClick={()=>{
            history.push(currentUserProfileURL)
        }} variation={ButtonVariation.TERTIARY} text={getString('manageApiToken')} />
      </Layout.Vertical>
    </Dialog>
  )
}

export default CloneCredentialDialog
