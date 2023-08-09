import React, { useMemo, useState } from 'react'
import { Button, ButtonVariation, Dialog, FormikForm, FormInput, SelectOption, useToaster } from '@harness/uicore'
import { useModalHook } from '@harness/use-modal'
import { Formik } from 'formik'

import * as Yup from 'yup'

import { useStrings } from 'framework/strings'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import {
  MembershipAddRequestBody,
  TypesMembershipUser,
  TypesPrincipalInfo,
  useListPrincipals,
  useMembershipAdd,
  useMembershipUpdate
} from 'services/code'
import { getErrorMessage, LIST_FETCHING_LIMIT } from 'utils/Utils'

import { roleStringKeyMap } from '../SpaceAccessControl'

const roles = ['reader', 'executor', 'contributor', 'space_owner'] as const

const useAddNewMember = ({ onClose }: { onClose: () => void }) => {
  const [isEditFlow, setIsEditFlow] = useState(false)
  const [membershipDetails, setMembershipDetails] = useState<TypesMembershipUser>()
  const [searchTerm, setSearchTerm] = useState('')

  const space = useGetSpaceParam()
  const { getString } = useStrings()
  const { showError, showSuccess } = useToaster()

  const { mutate: addMembership } = useMembershipAdd({ space_ref: space })
  const { mutate: updateMembership } = useMembershipUpdate({
    space_ref: space,
    user_uid: membershipDetails?.principal?.uid || ''
  })

  const { data: users, loading: fetchingUsers } = useListPrincipals({
    queryParams: {
      query: searchTerm,
      page: 1,
      limit: LIST_FETCHING_LIMIT,
      type: ['user']
    },
    debounce: 500
  })

  const roleOptions: SelectOption[] = useMemo(
    () =>
      roles.map(role => ({
        value: role,
        label: getString(roleStringKeyMap[role])
      })),
    []
  )

  const userOptions: SelectOption[] = useMemo(
    () =>
      users?.map(user => ({
        value: user.uid as string,
        label: (user.display_name || user.email) as string
      })) || [],
    [users]
  )

  const handleClose = () => {
    setSearchTerm('')
    hideModal()
  }

  const [openModal, hideModal] = useModalHook(() => {
    return (
      <Dialog
        isOpen
        enforceFocus={false}
        onClose={handleClose}
        title={isEditFlow ? getString('changeRole') : getString('spaceMemberships.addMember')}>
        <Formik<MembershipAddRequestBody>
          initialValues={{
            user_uid: membershipDetails?.principal?.uid || '',
            role: membershipDetails?.role || 'reader'
          }}
          validationSchema={Yup.object().shape({
            user_uid: Yup.string().required(getString('validation.uidRequired'))
          })}
          onSubmit={async values => {
            try {
              if (isEditFlow) {
                await updateMembership({ role: values.role })
                showSuccess(getString('spaceMemberships.memberUpdated'))
              } else {
                await addMembership(values)
                showSuccess(getString('spaceMemberships.memberAdded'))
              }

              handleClose()
              onClose()
            } catch (error) {
              showError(getErrorMessage(error))
            }
          }}>
          <FormikForm>
            <FormInput.Select
              usePortal
              name="user_uid"
              label={getString('user')}
              items={userOptions}
              value={
                isEditFlow
                  ? {
                      label: membershipDetails?.principal?.display_name as string,
                      value: membershipDetails?.principal?.uid as string
                    }
                  : undefined
              }
              disabled={isEditFlow}
              onQueryChange={setSearchTerm}
              selectProps={{ loadingItems: fetchingUsers }}
            />
            <FormInput.Select name="role" label={getString('role')} items={roleOptions} usePortal />
            <Button
              type="submit"
              margin={{ top: 'xxxlarge' }}
              text={isEditFlow ? getString('save') : getString('addMember')}
              variation={ButtonVariation.PRIMARY}
            />
          </FormikForm>
        </Formik>
      </Dialog>
    )
  }, [isEditFlow, membershipDetails, userOptions])

  return {
    openModal: (isEditing?: boolean, memberInfo?: TypesPrincipalInfo) => {
      openModal()
      setIsEditFlow(Boolean(isEditing))
      setMembershipDetails(memberInfo)
    },
    hideModal
  }
}

export default useAddNewMember
