import * as yup from 'yup'
import type { UseStringsReturn } from 'framework/strings'

export const validateGitnessForm = (getString: UseStringsReturn['getString']) =>
  yup.object().shape({
    branch: yup.string().trim().required(getString('cde.branchValidationMessage')),
    code_repo_url: yup.string().trim().required(getString('cde.repoValidationMessage')),
    identifier: yup.string().trim().required(),
    ide: yup.string().trim().required(),
    resource_identifier: yup.string().trim().required(getString('cde.machineValidationMessage')),
    name: yup.string().trim().required()
  })
