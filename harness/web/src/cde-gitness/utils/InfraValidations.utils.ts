import * as yup from 'yup'
import cidrRegex from 'cidr-regex'
import type { UseStringsReturn } from 'framework/strings'

export const AWS_AMI_ID_PATTERN = /^ami-([0-9a-f]{8}|[0-9a-f]{17})$/
const GCP_IMAGE_NAME_PATTERN = /^projects\/[a-zA-Z0-9-]+\/global\/images\/(family\/[a-zA-Z0-9-]+|[a-zA-Z0-9-]+)$/

export const validateInfraForm = (getString: UseStringsReturn['getString']) =>
  yup.object().shape({
    name: yup
      .string()
      .trim()
      .required(getString('cde.gitspaceInfraHome.nameMessage'))
      .min(5, getString('cde.gitspaceInfraHome.minMessage', { field: 'Infrastructure Name', count: '5' }))
      .max(20, getString('cde.gitspaceInfraHome.maxMessage', { field: 'Infrastructure Name', count: '20' })),
    domain: yup.string().trim().required(getString('cde.gitspaceInfraHome.domainMessage')),
    machine_type: yup.string().trim().required(getString('cde.gitspaceInfraHome.machineTypeMessage')),
    gateway: yup.object().shape({
      vm_image_name: yup
        .string()
        .trim()
        // .required(getString('cde.configureInfra.gatewayImageNameRequired'))
        .matches(GCP_IMAGE_NAME_PATTERN, getString('cde.gitspaceInfraHome.invalidImageNameFormat'))
    }),
    runner: yup.object().shape({
      region: yup.string().trim().required('Region is required'),
      zone: yup.string().trim().required('Zone is required'),
      vm_image_name: yup
        .string()
        .trim()
        // .required(getString('cde.gitspaceInfraHome.machineImageNameRequired'))
        .matches(GCP_IMAGE_NAME_PATTERN, getString('cde.gitspaceInfraHome.invalidImageNameFormat'))
    })
  })

export const validateAwsInfraForm = (getString: UseStringsReturn['getString']) =>
  yup.object().shape({
    name: yup
      .string()
      .trim()
      .required(getString('cde.gitspaceInfraHome.nameMessage'))
      .min(5, getString('cde.gitspaceInfraHome.minMessage', { field: 'Infrastructure Name', count: '5' }))
      .max(20, getString('cde.gitspaceInfraHome.maxMessage', { field: 'Infrastructure Name', count: '20' })),
    domain: yup.string().trim().required(getString('cde.gitspaceInfraHome.domainMessage')),
    instance_type: yup.string().trim().required(getString('cde.gitspaceInfraHome.instanceTypeMessage')),
    vpc_cidr_block: yup
      .string()
      .trim()
      .required('VPC CIDR Block is required')
      .matches(cidrRegex({ exact: true }), 'Invalid CIDR format'),
    runner: yup.object().shape({
      region: yup.string().trim().required('Region is required'),
      availability_zones: yup.string().trim().required('Availability Zone is required'),
      ami_id: yup
        .string()
        .trim()
        .required(getString('cde.Aws.runnerAmiIdRequired'))
        .matches(AWS_AMI_ID_PATTERN, getString('cde.Aws.invalidAmiIdFormat'))
    })
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
    zone: yup.string().trim().required(getString('cde.gitspaceInfraHome.zoneMessage')),
    os: yup.string().trim().required(getString('cde.gitspaceInfraHome.osRequired')),
    arch: yup.string().trim().required(getString('cde.gitspaceInfraHome.architectureRequired')),
    image_name: yup
      .string()
      .trim()
      // .required(getString('cde.gitspaceInfraHome.machineImageNameRequired'))
      .matches(GCP_IMAGE_NAME_PATTERN, getString('cde.gitspaceInfraHome.invalidImageNameFormat'))
  })

export const validateAwsMachineForm = (getString: UseStringsReturn['getString']) =>
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
    zone: yup.string().trim().required(getString('cde.gitspaceInfraHome.zoneMessage')),
    os: yup.string().trim().required(getString('cde.gitspaceInfraHome.osRequired')),
    arch: yup.string().trim().required(getString('cde.gitspaceInfraHome.architectureRequired')),
    vm_image_name: yup
      .string()
      .trim()
      .required(getString('cde.Aws.machineAmiIdRequired'))
      .matches(AWS_AMI_ID_PATTERN, getString('cde.Aws.invalidAmiIdFormat'))
  })
