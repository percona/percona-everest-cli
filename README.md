# Percona Everest CLI

This tool is a CLI client for Percona Everest and has the following features

1. Provisioning of Percona Everest on Kubernetes clusters 
2. CLI client for Percona Everest that helps you to manage database clusters


## Prerequisites

1. Go
2. make

## Using Percona Everest CLI

At the moment it only provisions the clusters, however a techinical preview of PMM integration and registering in Percona Everest control plane features are implemented 

```
go run cmd/percona-everest-cli/main.go --monitoring.enabled=false
```
