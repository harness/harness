import * as yup from 'yup'
import type { UseStringsReturn } from 'framework/strings'

export const validateInfraForm = (getString: UseStringsReturn['getString']) =>
  yup.object().shape({
    name: yup
      .string()
      .trim()
      .required(getString('cde.gitspaceInfraHome.nameMessage'))
      .min(5, getString('cde.gitspaceInfraHome.minMessage', { field: 'Infrastructure Name', count: '5' }))
      .max(20, getString('cde.gitspaceInfraHome.maxMessage', { field: 'Infrastructure Name', count: '20' })),
    domain: yup.string().trim().required(getString('cde.gitspaceInfraHome.domainMessage')),
    machine_type: yup.string().trim().required(getString('cde.gitspaceInfraHome.machineTypeMessage'))
    // instances: yup.string().trim().required(getString('cde.gitspaceInfraHome.instanceMessage'))
  })

export const validateMachineForm = (getString: UseStringsReturn['getString']) =>
  yup.object().shape({
    name: yup
      .string()
      .trim()
      .required(getString('cde.gitspaceInfraHome.nameMessage'))
      .min(4, getString('cde.gitspaceInfraHome.minMessage', { field: 'Name', count: '4' }))
      .max(20, getString('cde.gitspaceInfraHome.maxMessage', { field: 'Name', count: '20' })),
    disk_type: yup.string().trim().required(getString('cde.gitspaceInfraHome.diskTypeMessage')),
    boot_size: yup
      .number()
      .required(getString('cde.gitspaceInfraHome.bootSizeMessage'))
      .min(1, getString('cde.gitspaceInfraHome.minNumber', { field: 'Boot Size', count: '0' })),
    machine_type: yup.string().trim().required(getString('cde.gitspaceInfraHome.machineTypeMessage')),
    boot_type: yup.string().trim().required(getString('cde.gitspaceInfraHome.bootTypeMessage')),
    disk_size: yup
      .number()
      .required(getString('cde.gitspaceInfraHome.diskSizeMessage'))
      .min(1, getString('cde.gitspaceInfraHome.minNumber', { field: 'Persistent Disk Size', count: '0' })),
    zone: yup.string().trim().required(getString('cde.gitspaceInfraHome.zoneMessage'))
  })
