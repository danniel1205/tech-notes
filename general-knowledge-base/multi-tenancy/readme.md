# Multi-tenancy

## What is multi-tenancy

Modern companies are looking for more efficient and cost-effective ways to manage their data and resources. Multitenancy
enables multiple customers to share a single software instance.

Multitenancy is a software architecture that serves multiple users by delivering a single instance of the software.
Different users can access the data, configuration and other specific functionalities of a given instance.

* Single tenancy provides an isolated environment deployed separately for each tenant.

* Multitenancy uses a shared platform to serve multiple tenants; these tenants receive defined user roles and access
  rights to their dedicated environments to manage their interface.

## Difference between single-tenant and multi-tenant

![](resources/single-tenant-vs-multi-tenant.png)

## Key Features of Multi-tenancy

Each multi-tenant solution should have the following features:

* Shared infra: Serve multiple customers with a single instance, while maintaining tenant isolation.

* Tenant isolation: Each tenant is restricted by their assigned role in order to access or manage their dedicated
  environment.

* Customization: Each tenant can has its own configurations.

* Scalability: System can accommodate growth without been overwhelmed or experiencing degradation in performance.
  Multi-tenant architecture should be able to handle increasing numbers of tenants and automatically allocate resources
  as needed.

* Centralized management/maintenance: Helps to manage multiple tenants from a single point of control via a single interface.

## Multi-tenancy Architecture

A design approach that enable multiple tenants to access to one instance of a system. Multi-tenant architecture creates
distinct, isolated environments within a single physical infrastructure, such as a virtual machine, server, or cloud
platform. This is accomplished by partitioning the data storage and processing; providing each tenant with their own
dedicated space in the system. A tenant interacts with the application and can access their own data.

### Types of Multi-tenancy

* Separate data source for each tenant

* Single data source, multiple schemas

* Single data source, shared schema

![img.png](resources/types-of-multitenancy.png)

### Advantages and Disadvantages of Multi-tenancy

#### Advantages

* Scalability: Single instance of a system can be easily scale up/out.
* Cost saving & Increased efficiency: Multi-tenant can share resources like compute, storage.
* Easy Maintenance/Management: Easily fan-out the updates.
* Customization: Each tenant can have its own configurations.
* Tenant privacy: Tenant isolation.

#### Disadvantages

* Potential security issue if incorrectly implemented
* Implementing additional logic for tenant separation, data filtering and tenant identification
* System outage wil have impact on all tenants

### Multi-tenancy Architecture in K8S

### Multi-tenancy Architecture in ETCD

ETCD is the key-value store, it implements the multi-tenancy by using [namespaces](https://pkg.go.dev/go.etcd.io/etcd/clientv3/namespace).
A namespace allows to add a prefix for all keys, so it logically isolate data from different tenants. Once a prefix is
added, [ETCD RBAC](https://etcd.io/docs/v3.5/op-guide/authentication/rbac/) can be applied.

#### Implementation of multi-tenancy in etcd

There are two main approaches to implementing multi-tenancy in etcd using namespaces:

* Per-tenant namespace: In this approach, each tenant is assigned a unique namespace. This is the simplest approach to
  implement multi-tenancy, but it can be difficult to manage if there are a large number of tenants.

* Shared namespace: In this approach, all tenants share a single namespace. This approach is more scalable than the
  per-tenant namespace approach, but it requires more careful management to ensure that tenants do not conflict with each other.

#### How does KCP make changes to ETCD to fulfil its multi-tenancy

More details can be found [here](https://docs.kcp.io/kcp/main/developers/etcd-structure/)

![](resources/built-in-apis.png)

![](resources/shared-cr-instances.png)

![](resources/bound-cr-instances.png)

## References

* <https://www.gooddata.com/blog/what-multitenancy/>
* <https://romanglushach.medium.com/kubernetes-multi-tenancy-challenges-benefits-and-best-practices-for-enterprises-26c6aa76f0d7>