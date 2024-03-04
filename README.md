# Percona Everest CLI (Archived)

**Note: This repository has been archived and its functionality has been merged into a consolidated repository.**

For the latest updates and features, please visit the new repository:

[Everest](https://github.com/percona/everest)

#

This tool is a CLI client for Percona Everest and has the following features

1. Provisioning of Percona Everest on Kubernetes clusters 
2. CLI client for Percona Everest that helps you manage database clusters


## Prerequisites

1. Go
2. make

## Using Percona Everest CLI

At the moment it only provisions the clusters, however a technical preview of PMM integration and registering in Percona Everest control plane features are implemented 

```
go run cmd/everest/main.go install
```
