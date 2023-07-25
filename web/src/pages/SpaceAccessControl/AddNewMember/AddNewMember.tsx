import React, { useMemo, useState } from 'react'
import { Button, ButtonVariation, Dialog, FormikForm, FormInput, SelectOption, useToaster } from '@harness/uicore'
import { useModalHook } from '@harness/use-modal'
import { Formik } from 'formik'
import { capitalize } from 'lodash-es'

import * as Yup from 'yup'

import { useStrings } from 'framework/strings'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import {
  MembershipAddRequestBody,
  TypesMembership,
  TypesPrincipalInfo,
  useMembershipAdd,
  useMembershipUpdate
} from 'services/code'
import { getErrorMessage } from 'utils/Utils'

const roles = ['contributor', 'executor', 'reader', 'space_owner'] as const

const useAddNewMember = ({ onClose }: { onClose: () => void }) => {
  const [isEditFlow, setIsEditFlow] = useState(false)
  const [membershipDetails, setMembershipDetails] = useState<TypesMembership>()

  const space = useGetSpaceParam()
  const { getString } = useStrings()
  const { showError, showSuccess } = useToaster()

  const { mutate: addMembership } = useMembershipAdd({ space_ref: space })
  const { mutate: updateMembership } = useMembershipUpdate({
    space_ref: space,
    user_uid: membershipDetails?.principal?.uid || ''
  })

  const roleOptions: SelectOption[] = useMemo(
    () =>
      roles.map(role => ({
        value: role,
        label: capitalize(role)
      })),
    []
  )

  const [openModal, hideModal] = useModalHook(() => {
    return (
      <Dialog isOpen enforceFocus={false} onClose={hideModal} title={isEditFlow ? 'Change role' : 'Add new member'}>
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

              hideModal()
              onClose()
            } catch (error) {
              showError(getErrorMessage(error))
            }
          }}>
          <FormikForm>
            <FormInput.Text
              name="user_uid"
              label={getString('userId')}
              placeholder={getString('newUserModal.uidPlaceholder')}
              disabled={isEditFlow}
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
  }, [isEditFlow, membershipDetails])

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
