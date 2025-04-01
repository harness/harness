import * as yup from 'yup'
import type { UseStringsReturn } from 'framework/strings'

export const validateInfraForm = (getString: UseStringsReturn['getString']) =>
  yup.object().shape({
    name: yup.string().trim().required(getString('cde.gitspaceInfraHome.nameMessage')),
    domain: yup.string().trim().required(getString('cde.gitspaceInfraHome.domainMessage')),
    machine_type: yup.string().trim().required(getString('cde.gitspaceInfraHome.machineTypeMessage')),
    instances: yup.string().trim().required(getString('cde.gitspaceInfraHome.instanceMessage'))
  })

export const validateMachineForm = (getString: UseStringsReturn['getString']) =>
  yup.object().shape({
    name: yup.string().trim().required(getString('cde.gitspaceInfraHome.nameMessage')),
    disk_type: yup.string().trim().required(getString('cde.gitspaceInfraHome.diskTypeMessage')),
    boot_size: yup.string().trim().required(getString('cde.gitspaceInfraHome.bootSizeMessage')),
    machine_type: yup.string().trim().required(getString('cde.gitspaceInfraHome.machineTypeMessage')),
    boot_type: yup.string().trim().required(getString('cde.gitspaceInfraHome.bootTypeMessage')),
    disk_size: yup.string().trim().required(getString('cde.gitspaceInfraHome.diskSizeMessage')),
    zone: yup.string().trim().required(getString('cde.gitspaceInfraHome.zoneMessage'))
  })
