# Percona Everest CLI

This tool is a CLI client for Percona Everest and has the following features

1. Provisioning of Percona Everest on Kubernetes clusters 
2. CLI client for Percona Everest that helps you manage database clusters


## Prerequisites

1. Go
2. make

## Using Percona Everest CLI

At the moment it only provisions the clusters, however a technical preview of PMM integration and registering in Percona Everest control plane features are implemented 

```
go run cmd/everest/main.go --monitoring.enabled=false
```
