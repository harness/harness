export const InfraDetails = {
  regions: {
    'us-east-1': ['us-east-1a', 'us-east-1b', 'us-east-1c', 'us-east-1d', 'us-east-1e', 'us-east-1f'],
    'us-east-2': ['us-east-2a', 'us-east-2b', 'us-east-2c'],
    'us-west-1': ['us-west-1a', 'us-west-1b', 'us-west-1c'],
    'us-west-2': ['us-west-2a', 'us-west-2b', 'us-west-2c', 'us-west-2d'],
    'ca-central-1': ['ca-central-1a', 'ca-central-1b', 'ca-central-1d'],
    'ca-west-1': ['ca-west-1a', 'ca-west-1b', 'ca-west-1c'],
    'sa-east-1': ['sa-east-1a', 'sa-east-1b', 'sa-east-1c'],
    'eu-west-1': ['eu-west-1a', 'eu-west-1b', 'eu-west-1c'],
    'eu-west-2': ['eu-west-2a', 'eu-west-2b', 'eu-west-2c'],
    'eu-west-3': ['eu-west-3a', 'eu-west-3b', 'eu-west-3c'],
    'eu-north-1': ['eu-north-1a', 'eu-north-1b', 'eu-north-1c'],
    'eu-south-1': ['eu-south-1a', 'eu-south-1b', 'eu-south-1c'],
    'eu-south-2': ['eu-south-2a', 'eu-south-2b', 'eu-south-2c'],
    'eu-central-1': ['eu-central-1a', 'eu-central-1b', 'eu-central-1c'],
    'eu-central-2': ['eu-central-2a', 'eu-central-2b', 'eu-central-2c'],
    'ap-southeast-1': ['ap-southeast-1a', 'ap-southeast-1b', 'ap-southeast-1c'],
    'ap-southeast-2': ['ap-southeast-2a', 'ap-southeast-2b', 'ap-southeast-2c'],
    'ap-southeast-3': ['ap-southeast-3a', 'ap-southeast-3b', 'ap-southeast-3c'],
    'ap-southeast-4': ['ap-southeast-4a', 'ap-southeast-4b', 'ap-southeast-4c'],
    'ap-south-1': ['ap-south-1a', 'ap-south-1b', 'ap-south-1c'],
    'ap-south-2': ['ap-south-2a', 'ap-south-2b', 'ap-south-2c'],
    'ap-northeast-1': ['ap-northeast-1a', 'ap-northeast-1c', 'ap-northeast-1d'],
    'ap-northeast-2': ['ap-northeast-2a', 'ap-northeast-2b', 'ap-northeast-2c'],
    'ap-northeast-3': ['ap-northeast-3a', 'ap-northeast-3b', 'ap-northeast-3c'],
    'ap-east-1': ['ap-east-1a', 'ap-east-1b', 'ap-east-1c'],
    'me-south-1': ['me-south-1a', 'me-south-1b', 'me-south-1c'],
    'me-central-1': ['me-central-1a', 'me-central-1b', 'me-central-1c'],
    'il-central-1': ['il-central-1a', 'il-central-1b', 'il-central-1c'],
    'af-south-1': ['af-south-1a', 'af-south-1b', 'af-south-1c']
  },
  instance_types: [
    {
      name: 't2.nano',
      cpu: '1',
      memory: '0.5GB',
      memory_gb: 0.5,
      disk_gb: 8,
      family: 't2',
      is_default: false
    },
    {
      name: 't2.micro',
      cpu: '1',
      memory: '1GB',
      memory_gb: 1,
      disk_gb: 8,
      family: 't2',
      is_default: true
    },
    {
      name: 't2.small',
      cpu: '1',
      memory: '2GB',
      memory_gb: 2,
      disk_gb: 8,
      family: 't2',
      is_default: false
    },
    {
      name: 't2.medium',
      cpu: '2',
      memory: '4GB',
      memory_gb: 4,
      disk_gb: 8,
      family: 't2',
      is_default: false
    },
    {
      name: 't2.large',
      cpu: '2',
      memory: '8GB',
      memory_gb: 8,
      disk_gb: 8,
      family: 't2',
      is_default: false
    },
    {
      name: 't2.xlarge',
      cpu: '4',
      memory: '16GB',
      memory_gb: 16,
      disk_gb: 8,
      family: 't2',
      is_default: false
    },
    {
      name: 't2.2xlarge',
      cpu: '8',
      memory: '32GB',
      memory_gb: 32,
      disk_gb: 8,
      family: 't2',
      is_default: false
    },
    {
      name: 't3.nano',
      cpu: '2',
      memory: '0.5GB',
      memory_gb: 0.5,
      disk_gb: 8,
      family: 't3',
      is_default: false
    },
    {
      name: 't3.micro',
      cpu: '2',
      memory: '1GB',
      memory_gb: 1,
      disk_gb: 8,
      family: 't3',
      is_default: false
    },
    {
      name: 't3.small',
      cpu: '2',
      memory: '2GB',
      memory_gb: 2,
      disk_gb: 8,
      family: 't3',
      is_default: false
    },
    {
      name: 't3.medium',
      cpu: '2',
      memory: '4GB',
      memory_gb: 4,
      disk_gb: 8,
      family: 't3',
      is_default: false
    },
    {
      name: 't3.large',
      cpu: '2',
      memory: '8GB',
      memory_gb: 8,
      disk_gb: 8,
      family: 't3',
      is_default: false
    },
    {
      name: 't3.xlarge',
      cpu: '4',
      memory: '16GB',
      memory_gb: 16,
      disk_gb: 8,
      family: 't3',
      is_default: false
    },
    {
      name: 't3.2xlarge',
      cpu: '8',
      memory: '32GB',
      memory_gb: 32,
      disk_gb: 8,
      family: 't3',
      is_default: false
    },
    {
      name: 't3a.nano',
      cpu: '2',
      memory: '0.5GB',
      memory_gb: 0.5,
      disk_gb: 8,
      family: 't3a',
      is_default: false
    },
    {
      name: 't3a.micro',
      cpu: '2',
      memory: '1GB',
      memory_gb: 1,
      disk_gb: 8,
      family: 't3a',
      is_default: false
    },
    {
      name: 't3a.small',
      cpu: '2',
      memory: '2GB',
      memory_gb: 2,
      disk_gb: 8,
      family: 't3a',
      is_default: false
    },
    {
      name: 't3a.medium',
      cpu: '2',
      memory: '4GB',
      memory_gb: 4,
      disk_gb: 8,
      family: 't3a',
      is_default: false
    },
    {
      name: 't3a.large',
      cpu: '2',
      memory: '8GB',
      memory_gb: 8,
      disk_gb: 8,
      family: 't3a',
      is_default: false
    },
    {
      name: 't3a.xlarge',
      cpu: '4',
      memory: '16GB',
      memory_gb: 16,
      disk_gb: 8,
      family: 't3a',
      is_default: false
    },
    {
      name: 't3a.2xlarge',
      cpu: '8',
      memory: '32GB',
      memory_gb: 32,
      disk_gb: 8,
      family: 't3a',
      is_default: false
    },
    {
      name: 't4g.nano',
      cpu: '2',
      memory: '0.5GB',
      memory_gb: 0.5,
      disk_gb: 8,
      family: 't4g',
      is_default: false
    },
    {
      name: 't4g.micro',
      cpu: '2',
      memory: '1GB',
      memory_gb: 1,
      disk_gb: 8,
      family: 't4g',
      is_default: false
    },
    {
      name: 't4g.small',
      cpu: '2',
      memory: '2GB',
      memory_gb: 2,
      disk_gb: 8,
      family: 't4g',
      is_default: false
    },
    {
      name: 't4g.medium',
      cpu: '2',
      memory: '4GB',
      memory_gb: 4,
      disk_gb: 8,
      family: 't4g',
      is_default: false
    },
    {
      name: 't4g.large',
      cpu: '2',
      memory: '8GB',
      memory_gb: 8,
      disk_gb: 8,
      family: 't4g',
      is_default: false
    },
    {
      name: 't4g.xlarge',
      cpu: '4',
      memory: '16GB',
      memory_gb: 16,
      disk_gb: 8,
      family: 't4g',
      is_default: false
    },
    {
      name: 't4g.2xlarge',
      cpu: '8',
      memory: '32GB',
      memory_gb: 32,
      disk_gb: 8,
      family: 't4g',
      is_default: false
    },
    {
      name: 'm4.large',
      cpu: '2',
      memory: '8GB',
      memory_gb: 8,
      disk_gb: 8,
      family: 'm4',
      is_default: false
    },
    {
      name: 'm4.xlarge',
      cpu: '4',
      memory: '16GB',
      memory_gb: 16,
      disk_gb: 8,
      family: 'm4',
      is_default: false
    },
    {
      name: 'm4.2xlarge',
      cpu: '8',
      memory: '32GB',
      memory_gb: 32,
      disk_gb: 8,
      family: 'm4',
      is_default: false
    },
    {
      name: 'm4.4xlarge',
      cpu: '16',
      memory: '64GB',
      memory_gb: 64,
      disk_gb: 8,
      family: 'm4',
      is_default: false
    },
    {
      name: 'm4.10xlarge',
      cpu: '40',
      memory: '160GB',
      memory_gb: 160,
      disk_gb: 8,
      family: 'm4',
      is_default: false
    },
    {
      name: 'm4.16xlarge',
      cpu: '64',
      memory: '256GB',
      memory_gb: 256,
      disk_gb: 8,
      family: 'm4',
      is_default: false
    },
    {
      name: 'm5.large',
      cpu: '2',
      memory: '8GB',
      memory_gb: 8,
      disk_gb: 8,
      family: 'm5',
      is_default: false
    },
    {
      name: 'm5.xlarge',
      cpu: '4',
      memory: '16GB',
      memory_gb: 16,
      disk_gb: 8,
      family: 'm5',
      is_default: false
    },
    {
      name: 'm5.2xlarge',
      cpu: '8',
      memory: '32GB',
      memory_gb: 32,
      disk_gb: 8,
      family: 'm5',
      is_default: false
    },
    {
      name: 'm5.4xlarge',
      cpu: '16',
      memory: '64GB',
      memory_gb: 64,
      disk_gb: 8,
      family: 'm5',
      is_default: false
    },
    {
      name: 'm5.8xlarge',
      cpu: '32',
      memory: '128GB',
      memory_gb: 128,
      disk_gb: 8,
      family: 'm5',
      is_default: false
    },
    {
      name: 'm5.12xlarge',
      cpu: '48',
      memory: '192GB',
      memory_gb: 192,
      disk_gb: 8,
      family: 'm5',
      is_default: false
    },
    {
      name: 'm5.16xlarge',
      cpu: '64',
      memory: '256GB',
      memory_gb: 256,
      disk_gb: 8,
      family: 'm5',
      is_default: false
    },
    {
      name: 'm5.24xlarge',
      cpu: '96',
      memory: '384GB',
      memory_gb: 384,
      disk_gb: 8,
      family: 'm5',
      is_default: false
    },
    {
      name: 'm5.metal',
      cpu: '96',
      memory: '384GB',
      memory_gb: 384,
      disk_gb: 8,
      family: 'm5',
      is_default: false
    },
    {
      name: 'm5a.large',
      cpu: '2',
      memory: '8GB',
      memory_gb: 8,
      disk_gb: 8,
      family: 'm5a',
      is_default: false
    },
    {
      name: 'm5a.xlarge',
      cpu: '4',
      memory: '16GB',
      memory_gb: 16,
      disk_gb: 8,
      family: 'm5a',
      is_default: false
    },
    {
      name: 'm5a.2xlarge',
      cpu: '8',
      memory: '32GB',
      memory_gb: 32,
      disk_gb: 8,
      family: 'm5a',
      is_default: false
    },
    {
      name: 'm5a.4xlarge',
      cpu: '16',
      memory: '64GB',
      memory_gb: 64,
      disk_gb: 8,
      family: 'm5a',
      is_default: false
    },
    {
      name: 'm5a.8xlarge',
      cpu: '32',
      memory: '128GB',
      memory_gb: 128,
      disk_gb: 8,
      family: 'm5a',
      is_default: false
    },
    {
      name: 'm5a.12xlarge',
      cpu: '48',
      memory: '192GB',
      memory_gb: 192,
      disk_gb: 8,
      family: 'm5a',
      is_default: false
    },
    {
      name: 'm5a.16xlarge',
      cpu: '64',
      memory: '256GB',
      memory_gb: 256,
      disk_gb: 8,
      family: 'm5a',
      is_default: false
    },
    {
      name: 'm5a.24xlarge',
      cpu: '96',
      memory: '384GB',
      memory_gb: 384,
      disk_gb: 8,
      family: 'm5a',
      is_default: false
    },
    {
      name: 'm5ad.large',
      cpu: '2',
      memory: '8GB',
      memory_gb: 8,
      disk_gb: 75,
      family: 'm5ad',
      is_default: false
    },
    {
      name: 'm5ad.xlarge',
      cpu: '4',
      memory: '16GB',
      memory_gb: 16,
      disk_gb: 150,
      family: 'm5ad',
      is_default: false
    },
    {
      name: 'm5ad.2xlarge',
      cpu: '8',
      memory: '32GB',
      memory_gb: 32,
      disk_gb: 300,
      family: 'm5ad',
      is_default: false
    },
    {
      name: 'm5ad.4xlarge',
      cpu: '16',
      memory: '64GB',
      memory_gb: 64,
      disk_gb: 600,
      family: 'm5ad',
      is_default: false
    },
    {
      name: 'm5d.large',
      cpu: '2',
      memory: '8GB',
      memory_gb: 8,
      disk_gb: 75,
      family: 'm5d',
      is_default: false
    },
    {
      name: 'm5d.xlarge',
      cpu: '4',
      memory: '16GB',
      memory_gb: 16,
      disk_gb: 150,
      family: 'm5d',
      is_default: false
    },
    {
      name: 'm5d.2xlarge',
      cpu: '8',
      memory: '32GB',
      memory_gb: 32,
      disk_gb: 300,
      family: 'm5d',
      is_default: false
    },
    {
      name: 'm5d.4xlarge',
      cpu: '16',
      memory: '64GB',
      memory_gb: 64,
      disk_gb: 600,
      family: 'm5d',
      is_default: false
    },
    {
      name: 'm5d.8xlarge',
      cpu: '32',
      memory: '128GB',
      memory_gb: 128,
      disk_gb: 1200,
      family: 'm5d',
      is_default: false
    },
    {
      name: 'm5d.12xlarge',
      cpu: '48',
      memory: '192GB',
      memory_gb: 192,
      disk_gb: 1800,
      family: 'm5d',
      is_default: false
    },
    {
      name: 'm5d.16xlarge',
      cpu: '64',
      memory: '256GB',
      memory_gb: 256,
      disk_gb: 2400,
      family: 'm5d',
      is_default: false
    },
    {
      name: 'm5d.24xlarge',
      cpu: '96',
      memory: '384GB',
      memory_gb: 384,
      disk_gb: 3600,
      family: 'm5d',
      is_default: false
    },
    {
      name: 'm5d.metal',
      cpu: '96',
      memory: '384GB',
      memory_gb: 384,
      disk_gb: 3600,
      family: 'm5d',
      is_default: false
    },
    {
      name: 'm5n.large',
      cpu: '2',
      memory: '8GB',
      memory_gb: 8,
      disk_gb: 8,
      family: 'm5n',
      is_default: false
    },
    {
      name: 'm5n.xlarge',
      cpu: '4',
      memory: '16GB',
      memory_gb: 16,
      disk_gb: 8,
      family: 'm5n',
      is_default: false
    },
    {
      name: 'm5n.2xlarge',
      cpu: '8',
      memory: '32GB',
      memory_gb: 32,
      disk_gb: 8,
      family: 'm5n',
      is_default: false
    },
    {
      name: 'm5n.4xlarge',
      cpu: '16',
      memory: '64GB',
      memory_gb: 64,
      disk_gb: 8,
      family: 'm5n',
      is_default: false
    },
    {
      name: 'm5n.8xlarge',
      cpu: '32',
      memory: '128GB',
      memory_gb: 128,
      disk_gb: 8,
      family: 'm5n',
      is_default: false
    },
    {
      name: 'm5n.12xlarge',
      cpu: '48',
      memory: '192GB',
      memory_gb: 192,
      disk_gb: 8,
      family: 'm5n',
      is_default: false
    },
    {
      name: 'm5n.16xlarge',
      cpu: '64',
      memory: '256GB',
      memory_gb: 256,
      disk_gb: 8,
      family: 'm5n',
      is_default: false
    },
    {
      name: 'm5n.24xlarge',
      cpu: '96',
      memory: '384GB',
      memory_gb: 384,
      disk_gb: 8,
      family: 'm5n',
      is_default: false
    },
    {
      name: 'm5n.metal',
      cpu: '96',
      memory: '384GB',
      memory_gb: 384,
      disk_gb: 8,
      family: 'm5n',
      is_default: false
    },
    {
      name: 'm5zn.large',
      cpu: '2',
      memory: '8GB',
      memory_gb: 8,
      disk_gb: 8,
      family: 'm5zn',
      is_default: false
    },
    {
      name: 'm5zn.xlarge',
      cpu: '4',
      memory: '16GB',
      memory_gb: 16,
      disk_gb: 8,
      family: 'm5zn',
      is_default: false
    },
    {
      name: 'm5zn.2xlarge',
      cpu: '8',
      memory: '32GB',
      memory_gb: 32,
      disk_gb: 8,
      family: 'm5zn',
      is_default: false
    },
    {
      name: 'm5zn.3xlarge',
      cpu: '12',
      memory: '48GB',
      memory_gb: 48,
      disk_gb: 8,
      family: 'm5zn',
      is_default: false
    },
    {
      name: 'm5zn.6xlarge',
      cpu: '24',
      memory: '96GB',
      memory_gb: 96,
      disk_gb: 8,
      family: 'm5zn',
      is_default: false
    },
    {
      name: 'm5zn.12xlarge',
      cpu: '48',
      memory: '192GB',
      memory_gb: 192,
      disk_gb: 8,
      family: 'm5zn',
      is_default: false
    },
    {
      name: 'm5zn.metal',
      cpu: '48',
      memory: '192GB',
      memory_gb: 192,
      disk_gb: 8,
      family: 'm5zn',
      is_default: false
    },
    {
      name: 'm6a.large',
      cpu: '2',
      memory: '4GB',
      memory_gb: 4,
      disk_gb: 8,
      family: 'm6a',
      is_default: false
    },
    {
      name: 'm6a.xlarge',
      cpu: '4',
      memory: '8GB',
      memory_gb: 8,
      disk_gb: 8,
      family: 'm6a',
      is_default: false
    },
    {
      name: 'm6a.2xlarge',
      cpu: '8',
      memory: '16GB',
      memory_gb: 16,
      disk_gb: 8,
      family: 'm6a',
      is_default: false
    },
    {
      name: 'm6a.4xlarge',
      cpu: '16',
      memory: '32GB',
      memory_gb: 32,
      disk_gb: 8,
      family: 'm6a',
      is_default: false
    },
    {
      name: 'm6a.8xlarge',
      cpu: '32',
      memory: '64GB',
      memory_gb: 64,
      disk_gb: 8,
      family: 'm6a',
      is_default: false
    },
    {
      name: 'm6a.12xlarge',
      cpu: '48',
      memory: '96GB',
      memory_gb: 96,
      disk_gb: 8,
      family: 'm6a',
      is_default: false
    },
    {
      name: 'm6a.16xlarge',
      cpu: '64',
      memory: '128GB',
      memory_gb: 128,
      disk_gb: 8,
      family: 'm6a',
      is_default: false
    },
    {
      name: 'm6a.24xlarge',
      cpu: '96',
      memory: '192GB',
      memory_gb: 192,
      disk_gb: 8,
      family: 'm6a',
      is_default: false
    },
    {
      name: 'm6a.32xlarge',
      cpu: '128',
      memory: '256GB',
      memory_gb: 256,
      disk_gb: 8,
      family: 'm6a',
      is_default: false
    },
    {
      name: 'm6a.48xlarge',
      cpu: '192',
      memory: '384GB',
      memory_gb: 384,
      disk_gb: 8,
      family: 'm6a',
      is_default: false
    },
    {
      name: 'm6a.metal',
      cpu: '192',
      memory: '384GB',
      memory_gb: 384,
      disk_gb: 8,
      family: 'm6a',
      is_default: false
    },
    {
      name: 'c4.large',
      cpu: '2',
      memory: '3.75GB',
      memory_gb: 3.75,
      disk_gb: 8,
      family: 'c4',
      is_default: false
    },
    {
      name: 'c4.xlarge',
      cpu: '4',
      memory: '7.5GB',
      memory_gb: 7.5,
      disk_gb: 8,
      family: 'c4',
      is_default: false
    },
    {
      name: 'c4.2xlarge',
      cpu: '8',
      memory: '15GB',
      memory_gb: 15,
      disk_gb: 8,
      family: 'c4',
      is_default: false
    },
    {
      name: 'c4.4xlarge',
      cpu: '16',
      memory: '30GB',
      memory_gb: 30,
      disk_gb: 8,
      family: 'c4',
      is_default: false
    },
    {
      name: 'c4.8xlarge',
      cpu: '36',
      memory: '60GB',
      memory_gb: 60,
      disk_gb: 8,
      family: 'c4',
      is_default: false
    },
    {
      name: 'c5.large',
      cpu: '2',
      memory: '4GB',
      memory_gb: 4,
      disk_gb: 8,
      family: 'c5',
      is_default: false
    },
    {
      name: 'c5.xlarge',
      cpu: '4',
      memory: '8GB',
      memory_gb: 8,
      disk_gb: 8,
      family: 'c5',
      is_default: false
    },
    {
      name: 'c5.2xlarge',
      cpu: '8',
      memory: '16GB',
      memory_gb: 16,
      disk_gb: 8,
      family: 'c5',
      is_default: false
    },
    {
      name: 'c5.4xlarge',
      cpu: '16',
      memory: '32GB',
      memory_gb: 32,
      disk_gb: 8,
      family: 'c5',
      is_default: false
    },
    {
      name: 'c5.9xlarge',
      cpu: '36',
      memory: '72GB',
      memory_gb: 72,
      disk_gb: 8,
      family: 'c5',
      is_default: false
    },
    {
      name: 'c5.12xlarge',
      cpu: '48',
      memory: '96GB',
      memory_gb: 96,
      disk_gb: 8,
      family: 'c5',
      is_default: false
    },
    {
      name: 'c5.18xlarge',
      cpu: '72',
      memory: '144GB',
      memory_gb: 144,
      disk_gb: 8,
      family: 'c5',
      is_default: false
    },
    {
      name: 'c5.24xlarge',
      cpu: '96',
      memory: '192GB',
      memory_gb: 192,
      disk_gb: 8,
      family: 'c5',
      is_default: false
    },
    {
      name: 'c5.metal',
      cpu: '96',
      memory: '192GB',
      memory_gb: 192,
      disk_gb: 8,
      family: 'c5',
      is_default: false
    },
    {
      name: 'c5a.large',
      cpu: '2',
      memory: '4GB',
      memory_gb: 4,
      disk_gb: 8,
      family: 'c5a',
      is_default: false
    },
    {
      name: 'c5a.xlarge',
      cpu: '4',
      memory: '8GB',
      memory_gb: 8,
      disk_gb: 8,
      family: 'c5a',
      is_default: false
    },
    {
      name: 'c5a.2xlarge',
      cpu: '8',
      memory: '16GB',
      memory_gb: 16,
      disk_gb: 8,
      family: 'c5a',
      is_default: false
    },
    {
      name: 'c5a.4xlarge',
      cpu: '16',
      memory: '32GB',
      memory_gb: 32,
      disk_gb: 8,
      family: 'c5a',
      is_default: false
    },
    {
      name: 'c5a.8xlarge',
      cpu: '32',
      memory: '64GB',
      memory_gb: 64,
      disk_gb: 8,
      family: 'c5a',
      is_default: false
    },
    {
      name: 'c5a.12xlarge',
      cpu: '48',
      memory: '96GB',
      memory_gb: 96,
      disk_gb: 8,
      family: 'c5a',
      is_default: false
    },
    {
      name: 'c5a.16xlarge',
      cpu: '64',
      memory: '128GB',
      memory_gb: 128,
      disk_gb: 8,
      family: 'c5a',
      is_default: false
    },
    {
      name: 'c5a.24xlarge',
      cpu: '96',
      memory: '192GB',
      memory_gb: 192,
      disk_gb: 8,
      family: 'c5a',
      is_default: false
    },
    {
      name: 'c5d.large',
      cpu: '2',
      memory: '4GB',
      memory_gb: 4,
      disk_gb: 8,
      family: 'c5d',
      is_default: false
    },
    {
      name: 'c5d.xlarge',
      cpu: '4',
      memory: '8GB',
      memory_gb: 8,
      disk_gb: 8,
      family: 'c5d',
      is_default: false
    },
    {
      name: 'c5d.2xlarge',
      cpu: '8',
      memory: '16GB',
      memory_gb: 16,
      disk_gb: 8,
      family: 'c5d',
      is_default: false
    },
    {
      name: 'c5d.4xlarge',
      cpu: '16',
      memory: '32GB',
      memory_gb: 32,
      disk_gb: 8,
      family: 'c5d',
      is_default: false
    },
    {
      name: 'c5d.9xlarge',
      cpu: '36',
      memory: '72GB',
      memory_gb: 72,
      disk_gb: 8,
      family: 'c5d',
      is_default: false
    },
    {
      name: 'c5d.18xlarge',
      cpu: '72',
      memory: '144GB',
      memory_gb: 144,
      disk_gb: 8,
      family: 'c5d',
      is_default: false
    },
    {
      name: 'c5n.large',
      cpu: '2',
      memory: '5.25GB',
      memory_gb: 5.25,
      disk_gb: 8,
      family: 'c5n',
      is_default: false
    },
    {
      name: 'c5n.xlarge',
      cpu: '4',
      memory: '10.5GB',
      memory_gb: 10.5,
      disk_gb: 8,
      family: 'c5n',
      is_default: false
    },
    {
      name: 'c5n.2xlarge',
      cpu: '8',
      memory: '21GB',
      memory_gb: 21,
      disk_gb: 8,
      family: 'c5n',
      is_default: false
    },
    {
      name: 'c5n.4xlarge',
      cpu: '16',
      memory: '42GB',
      memory_gb: 42,
      disk_gb: 8,
      family: 'c5n',
      is_default: false
    },
    {
      name: 'c5n.9xlarge',
      cpu: '36',
      memory: '96GB',
      memory_gb: 96,
      disk_gb: 8,
      family: 'c5n',
      is_default: false
    },
    {
      name: 'c5n.18xlarge',
      cpu: '72',
      memory: '192GB',
      memory_gb: 192,
      disk_gb: 8,
      family: 'c5n',
      is_default: false
    },
    {
      name: 'c5n.metal',
      cpu: '72',
      memory: '192GB',
      memory_gb: 192,
      disk_gb: 8,
      family: 'c5n',
      is_default: false
    },
    {
      name: 'c6a.large',
      cpu: '2',
      memory: '4GB',
      memory_gb: 4,
      disk_gb: 8,
      family: 'c6a',
      is_default: false
    },
    {
      name: 'c6a.xlarge',
      cpu: '4',
      memory: '8GB',
      memory_gb: 8,
      disk_gb: 8,
      family: 'c6a',
      is_default: false
    },
    {
      name: 'c6a.2xlarge',
      cpu: '8',
      memory: '16GB',
      memory_gb: 16,
      disk_gb: 8,
      family: 'c6a',
      is_default: false
    },
    {
      name: 'c6a.4xlarge',
      cpu: '16',
      memory: '32GB',
      memory_gb: 32,
      disk_gb: 8,
      family: 'c6a',
      is_default: false
    },
    {
      name: 'c6a.8xlarge',
      cpu: '32',
      memory: '64GB',
      memory_gb: 64,
      disk_gb: 8,
      family: 'c6a',
      is_default: false
    },
    {
      name: 'c6a.12xlarge',
      cpu: '48',
      memory: '96GB',
      memory_gb: 96,
      disk_gb: 8,
      family: 'c6a',
      is_default: false
    },
    {
      name: 'c6a.16xlarge',
      cpu: '64',
      memory: '128GB',
      memory_gb: 128,
      disk_gb: 8,
      family: 'c6a',
      is_default: false
    },
    {
      name: 'c6a.24xlarge',
      cpu: '96',
      memory: '192GB',
      memory_gb: 192,
      disk_gb: 8,
      family: 'c6a',
      is_default: false
    },
    {
      name: 'c6a.32xlarge',
      cpu: '128',
      memory: '256GB',
      memory_gb: 256,
      disk_gb: 8,
      family: 'c6a',
      is_default: false
    },
    {
      name: 'c6a.48xlarge',
      cpu: '192',
      memory: '384GB',
      memory_gb: 384,
      disk_gb: 8,
      family: 'c6a',
      is_default: false
    },
    {
      name: 'c6a.metal',
      cpu: '192',
      memory: '384GB',
      memory_gb: 384,
      disk_gb: 8,
      family: 'c6a',
      is_default: false
    },
    {
      name: 'r4.large',
      cpu: '2',
      memory: '15.25GB',
      memory_gb: 15.25,
      disk_gb: 8,
      family: 'r4',
      is_default: false
    },
    {
      name: 'r4.xlarge',
      cpu: '4',
      memory: '30.5GB',
      memory_gb: 30.5,
      disk_gb: 8,
      family: 'r4',
      is_default: false
    },
    {
      name: 'r4.2xlarge',
      cpu: '8',
      memory: '61GB',
      memory_gb: 61,
      disk_gb: 8,
      family: 'r4',
      is_default: false
    },
    {
      name: 'r4.4xlarge',
      cpu: '16',
      memory: '122GB',
      memory_gb: 122,
      disk_gb: 8,
      family: 'r4',
      is_default: false
    },
    {
      name: 'r4.8xlarge',
      cpu: '32',
      memory: '244GB',
      memory_gb: 244,
      disk_gb: 8,
      family: 'r4',
      is_default: false
    },
    {
      name: 'r4.16xlarge',
      cpu: '64',
      memory: '488GB',
      memory_gb: 488,
      disk_gb: 8,
      family: 'r4',
      is_default: false
    },
    {
      name: 'r5.large',
      cpu: '2',
      memory: '16GB',
      memory_gb: 16,
      disk_gb: 8,
      family: 'r5',
      is_default: false
    },
    {
      name: 'r5.xlarge',
      cpu: '4',
      memory: '32GB',
      memory_gb: 32,
      disk_gb: 8,
      family: 'r5',
      is_default: false
    },
    {
      name: 'r5.2xlarge',
      cpu: '8',
      memory: '64GB',
      memory_gb: 64,
      disk_gb: 8,
      family: 'r5',
      is_default: false
    },
    {
      name: 'r5.4xlarge',
      cpu: '16',
      memory: '128GB',
      memory_gb: 128,
      disk_gb: 8,
      family: 'r5',
      is_default: false
    },
    {
      name: 'r5.8xlarge',
      cpu: '32',
      memory: '256GB',
      memory_gb: 256,
      disk_gb: 8,
      family: 'r5',
      is_default: false
    },
    {
      name: 'r5.12xlarge',
      cpu: '48',
      memory: '384GB',
      memory_gb: 384,
      disk_gb: 8,
      family: 'r5',
      is_default: false
    },
    {
      name: 'r5.16xlarge',
      cpu: '64',
      memory: '512GB',
      memory_gb: 512,
      disk_gb: 8,
      family: 'r5',
      is_default: false
    },
    {
      name: 'r5.24xlarge',
      cpu: '96',
      memory: '768GB',
      memory_gb: 768,
      disk_gb: 8,
      family: 'r5',
      is_default: false
    },
    {
      name: 'r5.metal',
      cpu: '96',
      memory: '768GB',
      memory_gb: 768,
      disk_gb: 8,
      family: 'r5',
      is_default: false
    },
    {
      name: 'r5a.large',
      cpu: '2',
      memory: '16GB',
      memory_gb: 16,
      disk_gb: 8,
      family: 'r5a',
      is_default: false
    },
    {
      name: 'r5a.xlarge',
      cpu: '4',
      memory: '32GB',
      memory_gb: 32,
      disk_gb: 8,
      family: 'r5a',
      is_default: false
    },
    {
      name: 'r5a.2xlarge',
      cpu: '8',
      memory: '64GB',
      memory_gb: 64,
      disk_gb: 8,
      family: 'r5a',
      is_default: false
    },
    {
      name: 'r5a.4xlarge',
      cpu: '16',
      memory: '128GB',
      memory_gb: 128,
      disk_gb: 8,
      family: 'r5a',
      is_default: false
    },
    {
      name: 'r5a.8xlarge',
      cpu: '32',
      memory: '256GB',
      memory_gb: 256,
      disk_gb: 8,
      family: 'r5a',
      is_default: false
    },
    {
      name: 'r5a.12xlarge',
      cpu: '48',
      memory: '384GB',
      memory_gb: 384,
      disk_gb: 8,
      family: 'r5a',
      is_default: false
    },
    {
      name: 'r5a.16xlarge',
      cpu: '64',
      memory: '512GB',
      memory_gb: 512,
      disk_gb: 8,
      family: 'r5a',
      is_default: false
    },
    {
      name: 'r5a.24xlarge',
      cpu: '96',
      memory: '768GB',
      memory_gb: 768,
      disk_gb: 8,
      family: 'r5a',
      is_default: false
    },
    {
      name: 'r5d.large',
      cpu: '2',
      memory: '16GB',
      memory_gb: 16,
      disk_gb: 8,
      family: 'r5d',
      is_default: false
    },
    {
      name: 'r5d.xlarge',
      cpu: '4',
      memory: '32GB',
      memory_gb: 32,
      disk_gb: 8,
      family: 'r5d',
      is_default: false
    },
    {
      name: 'r5d.2xlarge',
      cpu: '8',
      memory: '64GB',
      memory_gb: 64,
      disk_gb: 8,
      family: 'r5d',
      is_default: false
    },
    {
      name: 'r5d.4xlarge',
      cpu: '16',
      memory: '128GB',
      memory_gb: 128,
      disk_gb: 8,
      family: 'r5d',
      is_default: false
    },
    {
      name: 'r5d.8xlarge',
      cpu: '32',
      memory: '256GB',
      memory_gb: 256,
      disk_gb: 8,
      family: 'r5d',
      is_default: false
    },
    {
      name: 'r5d.12xlarge',
      cpu: '48',
      memory: '384GB',
      memory_gb: 384,
      disk_gb: 8,
      family: 'r5d',
      is_default: false
    },
    {
      name: 'r5d.16xlarge',
      cpu: '64',
      memory: '512GB',
      memory_gb: 512,
      disk_gb: 8,
      family: 'r5d',
      is_default: false
    },
    {
      name: 'r5d.24xlarge',
      cpu: '96',
      memory: '768GB',
      memory_gb: 768,
      disk_gb: 8,
      family: 'r5d',
      is_default: false
    },
    {
      name: 'r5d.metal',
      cpu: '96',
      memory: '768GB',
      memory_gb: 768,
      disk_gb: 8,
      family: 'r5d',
      is_default: false
    },
    {
      name: 'r5n.large',
      cpu: '2',
      memory: '16GB',
      memory_gb: 16,
      disk_gb: 8,
      family: 'r5n',
      is_default: false
    },
    {
      name: 'r5n.xlarge',
      cpu: '4',
      memory: '32GB',
      memory_gb: 32,
      disk_gb: 8,
      family: 'r5n',
      is_default: false
    },
    {
      name: 'r5n.2xlarge',
      cpu: '8',
      memory: '64GB',
      memory_gb: 64,
      disk_gb: 8,
      family: 'r5n',
      is_default: false
    },
    {
      name: 'r5n.4xlarge',
      cpu: '16',
      memory: '128GB',
      memory_gb: 128,
      disk_gb: 8,
      family: 'r5n',
      is_default: false
    },
    {
      name: 'r5n.8xlarge',
      cpu: '32',
      memory: '256GB',
      memory_gb: 256,
      disk_gb: 8,
      family: 'r5n',
      is_default: false
    },
    {
      name: 'r5n.12xlarge',
      cpu: '48',
      memory: '384GB',
      memory_gb: 384,
      disk_gb: 8,
      family: 'r5n',
      is_default: false
    },
    {
      name: 'r5n.16xlarge',
      cpu: '64',
      memory: '512GB',
      memory_gb: 512,
      disk_gb: 8,
      family: 'r5n',
      is_default: false
    },
    {
      name: 'r5n.24xlarge',
      cpu: '96',
      memory: '768GB',
      memory_gb: 768,
      disk_gb: 8,
      family: 'r5n',
      is_default: false
    },
    {
      name: 'r5n.metal',
      cpu: '96',
      memory: '768GB',
      memory_gb: 768,
      disk_gb: 8,
      family: 'r5n',
      is_default: false
    },
    {
      name: 'r6a.large',
      cpu: '2',
      memory: '16GB',
      memory_gb: 16,
      disk_gb: 8,
      family: 'r6a',
      is_default: false
    },
    {
      name: 'r6a.xlarge',
      cpu: '4',
      memory: '32GB',
      memory_gb: 32,
      disk_gb: 8,
      family: 'r6a',
      is_default: false
    },
    {
      name: 'r6a.2xlarge',
      cpu: '8',
      memory: '64GB',
      memory_gb: 64,
      disk_gb: 8,
      family: 'r6a',
      is_default: false
    },
    {
      name: 'r6a.4xlarge',
      cpu: '16',
      memory: '128GB',
      memory_gb: 128,
      disk_gb: 8,
      family: 'r6a',
      is_default: false
    },
    {
      name: 'r6a.8xlarge',
      cpu: '32',
      memory: '256GB',
      memory_gb: 256,
      disk_gb: 8,
      family: 'r6a',
      is_default: false
    },
    {
      name: 'r6a.12xlarge',
      cpu: '48',
      memory: '384GB',
      memory_gb: 384,
      disk_gb: 8,
      family: 'r6a',
      is_default: false
    },
    {
      name: 'r6a.16xlarge',
      cpu: '64',
      memory: '512GB',
      memory_gb: 512,
      disk_gb: 8,
      family: 'r6a',
      is_default: false
    },
    {
      name: 'r6a.24xlarge',
      cpu: '96',
      memory: '768GB',
      memory_gb: 768,
      disk_gb: 8,
      family: 'r6a',
      is_default: false
    },
    {
      name: 'r6a.32xlarge',
      cpu: '128',
      memory: '1024GB',
      memory_gb: 1024,
      disk_gb: 8,
      family: 'r6a',
      is_default: false
    },
    {
      name: 'r6a.48xlarge',
      cpu: '192',
      memory: '1536GB',
      memory_gb: 1536,
      disk_gb: 8,
      family: 'r6a',
      is_default: false
    },
    {
      name: 'r6a.metal',
      cpu: '192',
      memory: '1536GB',
      memory_gb: 1536,
      disk_gb: 8,
      family: 'r6a',
      is_default: false
    },
    {
      name: 'x1.16xlarge',
      cpu: '64',
      memory: '976GB',
      memory_gb: 976,
      disk_gb: 8,
      family: 'x1',
      is_default: false
    },
    {
      name: 'x1.32xlarge',
      cpu: '128',
      memory: '1952GB',
      memory_gb: 1952,
      disk_gb: 8,
      family: 'x1',
      is_default: false
    },
    {
      name: 'x1e.xlarge',
      cpu: '4',
      memory: '122GB',
      memory_gb: 122,
      disk_gb: 8,
      family: 'x1e',
      is_default: false
    },
    {
      name: 'x1e.2xlarge',
      cpu: '8',
      memory: '244GB',
      memory_gb: 244,
      disk_gb: 8,
      family: 'x1e',
      is_default: false
    },
    {
      name: 'x1e.4xlarge',
      cpu: '16',
      memory: '488GB',
      memory_gb: 488,
      disk_gb: 8,
      family: 'x1e',
      is_default: false
    },
    {
      name: 'x1e.8xlarge',
      cpu: '32',
      memory: '976GB',
      memory_gb: 976,
      disk_gb: 8,
      family: 'x1e',
      is_default: false
    },
    {
      name: 'x1e.16xlarge',
      cpu: '64',
      memory: '1952GB',
      memory_gb: 1952,
      disk_gb: 8,
      family: 'x1e',
      is_default: false
    },
    {
      name: 'x1e.32xlarge',
      cpu: '128',
      memory: '3904GB',
      memory_gb: 3904,
      disk_gb: 8,
      family: 'x1e',
      is_default: false
    },
    {
      name: 'x2gd.medium',
      cpu: '1',
      memory: '16GB',
      memory_gb: 16,
      disk_gb: 8,
      family: 'x2gd',
      is_default: false
    },
    {
      name: 'x2gd.large',
      cpu: '2',
      memory: '32GB',
      memory_gb: 32,
      disk_gb: 8,
      family: 'x2gd',
      is_default: false
    },
    {
      name: 'x2gd.xlarge',
      cpu: '4',
      memory: '64GB',
      memory_gb: 64,
      disk_gb: 8,
      family: 'x2gd',
      is_default: false
    },
    {
      name: 'x2gd.2xlarge',
      cpu: '8',
      memory: '128GB',
      memory_gb: 128,
      disk_gb: 8,
      family: 'x2gd',
      is_default: false
    },
    {
      name: 'x2gd.4xlarge',
      cpu: '16',
      memory: '256GB',
      memory_gb: 256,
      disk_gb: 8,
      family: 'x2gd',
      is_default: false
    },
    {
      name: 'x2gd.8xlarge',
      cpu: '32',
      memory: '512GB',
      memory_gb: 512,
      disk_gb: 8,
      family: 'x2gd',
      is_default: false
    },
    {
      name: 'x2gd.12xlarge',
      cpu: '48',
      memory: '768GB',
      memory_gb: 768,
      disk_gb: 8,
      family: 'x2gd',
      is_default: false
    },
    {
      name: 'x2gd.16xlarge',
      cpu: '64',
      memory: '1024GB',
      memory_gb: 1024,
      disk_gb: 8,
      family: 'x2gd',
      is_default: false
    },
    {
      name: 'x2gd.metal',
      cpu: '64',
      memory: '1024GB',
      memory_gb: 1024,
      disk_gb: 8,
      family: 'x2gd',
      is_default: false
    },
    {
      name: 'i3.large',
      cpu: '2',
      memory: '15.25GB',
      memory_gb: 15.25,
      disk_gb: 475,
      family: 'i3',
      is_default: false
    },
    {
      name: 'i3.xlarge',
      cpu: '4',
      memory: '30.5GB',
      memory_gb: 30.5,
      disk_gb: 950,
      family: 'i3',
      is_default: false
    },
    {
      name: 'i3.2xlarge',
      cpu: '8',
      memory: '61GB',
      memory_gb: 61,
      disk_gb: 1900,
      family: 'i3',
      is_default: false
    },
    {
      name: 'i3.4xlarge',
      cpu: '16',
      memory: '122GB',
      memory_gb: 122,
      disk_gb: 3800,
      family: 'i3',
      is_default: false
    },
    {
      name: 'i3.8xlarge',
      cpu: '32',
      memory: '244GB',
      memory_gb: 244,
      disk_gb: 7600,
      family: 'i3',
      is_default: false
    },
    {
      name: 'i3.16xlarge',
      cpu: '64',
      memory: '488GB',
      memory_gb: 488,
      disk_gb: 15200,
      family: 'i3',
      is_default: false
    },
    {
      name: 'i3.metal',
      cpu: '72',
      memory: '512GB',
      memory_gb: 512,
      disk_gb: 15200,
      family: 'i3',
      is_default: false
    },
    {
      name: 'i3en.large',
      cpu: '2',
      memory: '16GB',
      memory_gb: 16,
      disk_gb: 1250,
      family: 'i3en',
      is_default: false
    },
    {
      name: 'i3en.xlarge',
      cpu: '4',
      memory: '32GB',
      memory_gb: 32,
      disk_gb: 2500,
      family: 'i3en',
      is_default: false
    },
    {
      name: 'i3en.2xlarge',
      cpu: '8',
      memory: '64GB',
      memory_gb: 64,
      disk_gb: 5000,
      family: 'i3en',
      is_default: false
    },
    {
      name: 'i3en.3xlarge',
      cpu: '12',
      memory: '96GB',
      memory_gb: 96,
      disk_gb: 7500,
      family: 'i3en',
      is_default: false
    },
    {
      name: 'i3en.6xlarge',
      cpu: '24',
      memory: '192GB',
      memory_gb: 192,
      disk_gb: 15000,
      family: 'i3en',
      is_default: false
    },
    {
      name: 'i3en.12xlarge',
      cpu: '48',
      memory: '384GB',
      memory_gb: 384,
      disk_gb: 30000,
      family: 'i3en',
      is_default: false
    },
    {
      name: 'i3en.24xlarge',
      cpu: '96',
      memory: '768GB',
      memory_gb: 768,
      disk_gb: 60000,
      family: 'i3en',
      is_default: false
    },
    {
      name: 'i3en.metal',
      cpu: '96',
      memory: '768GB',
      memory_gb: 768,
      disk_gb: 60000,
      family: 'i3en',
      is_default: false
    },
    {
      name: 'd2.xlarge',
      cpu: '4',
      memory: '30.5GB',
      memory_gb: 30.5,
      disk_gb: 6000,
      family: 'd2',
      is_default: false
    },
    {
      name: 'd2.2xlarge',
      cpu: '8',
      memory: '61GB',
      memory_gb: 61,
      disk_gb: 12000,
      family: 'd2',
      is_default: false
    },
    {
      name: 'd2.4xlarge',
      cpu: '16',
      memory: '122GB',
      memory_gb: 122,
      disk_gb: 24000,
      family: 'd2',
      is_default: false
    },
    {
      name: 'd2.8xlarge',
      cpu: '36',
      memory: '244GB',
      memory_gb: 244,
      disk_gb: 48000,
      family: 'd2',
      is_default: false
    },
    {
      name: 'd3.xlarge',
      cpu: '4',
      memory: '32GB',
      memory_gb: 32,
      disk_gb: 6000,
      family: 'd3',
      is_default: false
    },
    {
      name: 'd3.2xlarge',
      cpu: '8',
      memory: '64GB',
      memory_gb: 64,
      disk_gb: 12000,
      family: 'd3',
      is_default: false
    },
    {
      name: 'd3.4xlarge',
      cpu: '16',
      memory: '128GB',
      memory_gb: 128,
      disk_gb: 24000,
      family: 'd3',
      is_default: false
    },
    {
      name: 'd3.8xlarge',
      cpu: '32',
      memory: '256GB',
      memory_gb: 256,
      disk_gb: 48000,
      family: 'd3',
      is_default: false
    },
    {
      name: 'g4dn.xlarge',
      cpu: '4',
      memory: '16GB',
      memory_gb: 16,
      disk_gb: 8,
      family: 'g4dn',
      is_default: false
    },
    {
      name: 'g4dn.2xlarge',
      cpu: '8',
      memory: '32GB',
      memory_gb: 32,
      disk_gb: 8,
      family: 'g4dn',
      is_default: false
    },
    {
      name: 'g4dn.4xlarge',
      cpu: '16',
      memory: '64GB',
      memory_gb: 64,
      disk_gb: 8,
      family: 'g4dn',
      is_default: false
    },
    {
      name: 'g4dn.8xlarge',
      cpu: '32',
      memory: '128GB',
      memory_gb: 128,
      disk_gb: 8,
      family: 'g4dn',
      is_default: false
    },
    {
      name: 'g4dn.16xlarge',
      cpu: '64',
      memory: '256GB',
      memory_gb: 256,
      disk_gb: 8,
      family: 'g4dn',
      is_default: false
    },
    {
      name: 'g4dn.12xlarge',
      cpu: '48',
      memory: '192GB',
      memory_gb: 192,
      disk_gb: 8,
      family: 'g4dn',
      is_default: false
    },
    {
      name: 'g4dn.metal',
      cpu: '96',
      memory: '384GB',
      memory_gb: 384,
      disk_gb: 8,
      family: 'g4dn',
      is_default: false
    },
    {
      name: 'p3.2xlarge',
      cpu: '8',
      memory: '61GB',
      memory_gb: 61,
      disk_gb: 8,
      family: 'p3',
      is_default: false
    },
    {
      name: 'p3.8xlarge',
      cpu: '32',
      memory: '244GB',
      memory_gb: 244,
      disk_gb: 8,
      family: 'p3',
      is_default: false
    },
    {
      name: 'p3.16xlarge',
      cpu: '64',
      memory: '488GB',
      memory_gb: 488,
      disk_gb: 8,
      family: 'p3',
      is_default: false
    },
    {
      name: 'p3dn.24xlarge',
      cpu: '96',
      memory: '768GB',
      memory_gb: 768,
      disk_gb: 8,
      family: 'p3',
      is_default: false
    },
    {
      name: 'p4d.24xlarge',
      cpu: '96',
      memory: '1152GB',
      memory_gb: 1152,
      disk_gb: 8,
      family: 'p4d',
      is_default: false
    },
    {
      name: 'p4de.24xlarge',
      cpu: '96',
      memory: '1152GB',
      memory_gb: 1152,
      disk_gb: 8,
      family: 'p4de',
      is_default: false
    },
    {
      name: 'inf1.xlarge',
      cpu: '4',
      memory: '8GB',
      memory_gb: 8,
      disk_gb: 8,
      family: 'inf1',
      is_default: false
    },
    {
      name: 'inf1.2xlarge',
      cpu: '8',
      memory: '16GB',
      memory_gb: 16,
      disk_gb: 8,
      family: 'inf1',
      is_default: false
    },
    {
      name: 'inf1.6xlarge',
      cpu: '24',
      memory: '48GB',
      memory_gb: 48,
      disk_gb: 8,
      family: 'inf1',
      is_default: false
    },
    {
      name: 'inf1.24xlarge',
      cpu: '96',
      memory: '192GB',
      memory_gb: 192,
      disk_gb: 8,
      family: 'inf1',
      is_default: false
    },
    {
      name: 'inf2.xlarge',
      cpu: '4',
      memory: '16GB',
      memory_gb: 16,
      disk_gb: 8,
      family: 'inf2',
      is_default: false
    },
    {
      name: 'inf2.8xlarge',
      cpu: '32',
      memory: '128GB',
      memory_gb: 128,
      disk_gb: 8,
      family: 'inf2',
      is_default: false
    },
    {
      name: 'inf2.24xlarge',
      cpu: '96',
      memory: '384GB',
      memory_gb: 384,
      disk_gb: 8,
      family: 'inf2',
      is_default: false
    },
    {
      name: 'inf2.48xlarge',
      cpu: '192',
      memory: '768GB',
      memory_gb: 768,
      disk_gb: 8,
      family: 'inf2',
      is_default: false
    },
    {
      name: 'trn1.2xlarge',
      cpu: '8',
      memory: '32GB',
      memory_gb: 32,
      disk_gb: 8,
      family: 'trn1',
      is_default: false
    },
    {
      name: 'trn1.32xlarge',
      cpu: '128',
      memory: '512GB',
      memory_gb: 512,
      disk_gb: 8,
      family: 'trn1',
      is_default: false
    },
    {
      name: 'trn1n.32xlarge',
      cpu: '128',
      memory: '512GB',
      memory_gb: 512,
      disk_gb: 8,
      family: 'trn1n',
      is_default: false
    },
    {
      name: 'dl1.24xlarge',
      cpu: '96',
      memory: '768GB',
      memory_gb: 768,
      disk_gb: 8,
      family: 'dl1',
      is_default: false
    },
    {
      name: 'hpc6a.48xlarge',
      cpu: '96',
      memory: '384GB',
      memory_gb: 384,
      disk_gb: 8,
      family: 'hpc6a',
      is_default: false
    },
    {
      name: 'hpc6id.32xlarge',
      cpu: '64',
      memory: '1024GB',
      memory_gb: 1024,
      disk_gb: 8,
      family: 'hpc6id',
      is_default: false
    },
    {
      name: 'mac1.metal',
      cpu: '12',
      memory: '32GB',
      memory_gb: 32,
      disk_gb: 8,
      family: 'mac1',
      is_default: false
    },
    {
      name: 'mac2.metal',
      cpu: '8',
      memory: '24GB',
      memory_gb: 24,
      disk_gb: 8,
      family: 'mac2',
      is_default: false
    },
    {
      name: 'mac2-m2.metal',
      cpu: '8',
      memory: '24GB',
      memory_gb: 24,
      disk_gb: 8,
      family: 'mac2-m2',
      is_default: false
    },
    {
      name: 'mac2-m2pro.metal',
      cpu: '12',
      memory: '32GB',
      memory_gb: 32,
      disk_gb: 8,
      family: 'mac2-m2pro',
      is_default: false
    }
  ],
  volume_types: [
    {
      name: 'gp2',
      description: 'General Purpose SSD Volume (Legacy)'
    },
    {
      name: 'gp3',
      description: 'General Purpose SSD Volume'
    },
    {
      name: 'io1',
      description: 'Provisioned IOPS SSD Volume (Legacy)'
    },
    {
      name: 'io2',
      description: 'Provisioned IOPS SSD Volume'
    },
    {
      name: 'io2-block-express',
      description: 'Highest Performance SSD Volume'
    },
    {
      name: 'st1',
      description: 'Throughput Optimized HDD Volume'
    },
    {
      name: 'sc1',
      description: 'Cold HDD Volume'
    },
    {
      name: 'standard',
      description: 'Standard HDD Volume'
    },
    {
      name: 'instance-store',
      description: 'Ephemeral Instance Store'
    }
  ]
}
