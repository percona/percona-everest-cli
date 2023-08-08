#!/bin/bash

latest_release=$(curl -s https://api.github.com/repos/percona/percona-everest-cli/releases/latest |jq -r '.name')
os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m | tr '[:upper:]' '[:lower:]')

echo "Downloading the latest release of Percona Everest CLI"
curl -sL  https://github.com/percona/percona-everest-cli/releases/download/${latest_release}/everestctl-$os-$arch -o everestctl
chmod +x everestctl
echo "Deploying Backends using docker-compose"
curl -sL  https://raw.githubusercontent.com/percona/percona-everest-backend/main/quickstart.yml -o quickstart.yml
docker-compose -f quickstart.yml up -d

echo "Using default k8s cluster to provision everest without backups enabled and monitoring"
echo "You can run ./everestctl for the wizard setup"
echo ""
echo "Also you can use --kubeconfig to specify a path for kubeconfig you want to use"
./everestctl install operators --backup.enable=false --everest.endpoint=http://127.0.0.1:8080  --monitoring.enable=false --operator.mongodb=true --operator.postgresql=true --operator.xtradb-cluster=true --skip-wizard

open http://127.0.0.1:8080
