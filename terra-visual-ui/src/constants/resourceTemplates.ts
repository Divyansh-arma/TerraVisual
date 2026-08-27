export type CloudProvider = 'aws' | 'azure' | 'gcp';
export type ResourceCategory = 'Networking' | 'Compute' | 'Database' | 'Storage' | 'Load Balancers';

export interface ResourceTemplate {
  id: string;
  name: string;
  provider: CloudProvider;
  category: ResourceCategory;
  resourceType: string;
  defaultName: string;
  description: string;
  defaultAttributes: Record<string, any>;
}

export const RESOURCE_TEMPLATES: ResourceTemplate[] = [
  // --- Networking ---
  {
    id: 'aws_vpc',
    name: 'AWS VPC',
    provider: 'aws',
    category: 'Networking',
    resourceType: 'aws_vpc',
    defaultName: 'main_vpc',
    description: 'Virtual Private Cloud isolated virtual network in AWS.',
    defaultAttributes: {
      cidr_block: '10.0.0.0/16',
      enable_dns_hostnames: true,
      enable_dns_support: true,
    },
  },
  {
    id: 'aws_subnet',
    name: 'AWS Subnet',
    provider: 'aws',
    category: 'Networking',
    resourceType: 'aws_subnet',
    defaultName: 'public_subnet',
    description: 'Subnet inside an AWS Virtual Private Cloud.',
    defaultAttributes: {
      cidr_block: '10.0.1.0/24',
      vpc_id: 'aws_vpc.main.id',
    },
  },
  {
    id: 'azurerm_virtual_network',
    name: 'Azure Virtual Network',
    provider: 'azure',
    category: 'Networking',
    resourceType: 'azurerm_virtual_network',
    defaultName: 'main_vnet',
    description: 'Isolated network environment for Azure resources.',
    defaultAttributes: {
      address_space: ['10.0.0.0/16'],
      location: 'eastus',
    },
  },
  {
    id: 'google_compute_network',
    name: 'GCP VPC Network',
    provider: 'gcp',
    category: 'Networking',
    resourceType: 'google_compute_network',
    defaultName: 'custom_network',
    description: 'Global virtual network in Google Cloud Platform.',
    defaultAttributes: {
      auto_create_subnetworks: false,
    },
  },

  // --- Compute ---
  {
    id: 'aws_instance',
    name: 'AWS EC2 Instance',
    provider: 'aws',
    category: 'Compute',
    resourceType: 'aws_instance',
    defaultName: 'web_server',
    description: 'Scalable cloud virtual computing instance.',
    defaultAttributes: {
      ami: 'ami-0c55b159cbfafe1f0',
      instance_type: 't3.micro',
    },
  },
  {
    id: 'azurerm_linux_virtual_machine',
    name: 'Azure Linux VM',
    provider: 'azure',
    category: 'Compute',
    resourceType: 'azurerm_linux_virtual_machine',
    defaultName: 'linux_vm',
    description: 'Enterprise Linux Virtual Machine on Microsoft Azure.',
    defaultAttributes: {
      size: 'Standard_B1s',
      admin_username: 'azureuser',
    },
  },
  {
    id: 'google_compute_instance',
    name: 'GCP Compute Engine',
    provider: 'gcp',
    category: 'Compute',
    resourceType: 'google_compute_instance',
    defaultName: 'compute_node',
    description: 'High-performance virtual machine in Google Cloud.',
    defaultAttributes: {
      machine_type: 'e2-micro',
      zone: 'us-central1-a',
    },
  },

  // --- Database ---
  {
    id: 'aws_db_instance',
    name: 'AWS RDS Database',
    provider: 'aws',
    category: 'Database',
    resourceType: 'aws_db_instance',
    defaultName: 'primary_db',
    description: 'Managed relational SQL database service on AWS.',
    defaultAttributes: {
      engine: 'postgres',
      instance_class: 'db.t3.micro',
      allocated_storage: 20,
    },
  },
  {
    id: 'azurerm_postgresql_server',
    name: 'Azure Database for PostgreSQL',
    provider: 'azure',
    category: 'Database',
    resourceType: 'azurerm_postgresql_server',
    defaultName: 'postgres_server',
    description: 'Fully managed PostgreSQL database server on Azure.',
    defaultAttributes: {
      sku_name: 'B_Gen5_1',
      storage_mb: 5120,
    },
  },
  {
    id: 'google_sql_database_instance',
    name: 'GCP Cloud SQL',
    provider: 'gcp',
    category: 'Database',
    resourceType: 'google_sql_database_instance',
    defaultName: 'sql_instance',
    description: 'Fully-managed relational database service on GCP.',
    defaultAttributes: {
      database_version: 'POSTGRES_15',
      tier: 'db-f1-micro',
    },
  },

  // --- Storage ---
  {
    id: 'aws_s3_bucket',
    name: 'AWS S3 Bucket',
    provider: 'aws',
    category: 'Storage',
    resourceType: 'aws_s3_bucket',
    defaultName: 'app_storage',
    description: 'Scalable cloud object storage bucket.',
    defaultAttributes: {
      bucket: 'my-app-storage-bucket',
    },
  },
  {
    id: 'azurerm_storage_account',
    name: 'Azure Storage Account',
    provider: 'azure',
    category: 'Storage',
    resourceType: 'azurerm_storage_account',
    defaultName: 'appstorageacct',
    description: 'High-availability unified storage for Azure.',
    defaultAttributes: {
      account_tier: 'Standard',
      account_replication_type: 'LRS',
    },
  },
  {
    id: 'google_storage_bucket',
    name: 'GCP Cloud Storage Bucket',
    provider: 'gcp',
    category: 'Storage',
    resourceType: 'google_storage_bucket',
    defaultName: 'app_gcp_bucket',
    description: 'Worldwide object storage service by Google Cloud.',
    defaultAttributes: {
      name: 'my-gcp-storage-bucket',
      location: 'US',
    },
  },

  // --- Load Balancers ---
  {
    id: 'aws_lb',
    name: 'AWS Application Load Balancer',
    provider: 'aws',
    category: 'Load Balancers',
    resourceType: 'aws_lb',
    defaultName: 'app_alb',
    description: 'Elastic Load Balancer routing HTTP/HTTPS traffic.',
    defaultAttributes: {
      internal: false,
      load_balancer_type: 'application',
    },
  },
  {
    id: 'azurerm_lb',
    name: 'Azure Load Balancer',
    provider: 'azure',
    category: 'Load Balancers',
    resourceType: 'azurerm_lb',
    defaultName: 'app_lb',
    description: 'Ultra-low latency Layer 4 load balancing on Azure.',
    defaultAttributes: {
      sku: 'Standard',
    },
  },
  {
    id: 'google_compute_forwarding_rule',
    name: 'GCP Forwarding Rule',
    provider: 'gcp',
    category: 'Load Balancers',
    resourceType: 'google_compute_forwarding_rule',
    defaultName: 'frontend_forwarder',
    description: 'Load balancing forwarding rule for GCP traffic routing.',
    defaultAttributes: {
      ip_protocol: 'TCP',
      port_range: '80',
    },
  },
];
