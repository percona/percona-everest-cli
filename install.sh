#!/bin/bash

if ! command -v curl &> /dev/null
then
	echo "curl command not found. Please install it."
	exit
fi
if ! command -v docker &> /dev/null
then
	echo "docker command not found. Please install it."
	exit
fi
if ! docker compose version &> /dev/null
then
	echo "docker compose (v2) not found. Please install it."
	exit
fi
if ! command -v jq &> /dev/null
then
	echo "jq is not found. Please install it."
	exit
fi

latest_release=$(curl -s https://api.github.com/repos/percona/percona-everest-cli/releases/latest |jq -r '.name')
os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m | tr '[:upper:]' '[:lower:]')

if [[ ($os == "linux" || $os == "darwin") && $arch == "x86_64" ]]
then
	arch="amd64"
fi


echo "Downloading the latest release of Percona Everest CLI"
echo "https://github.com/percona/percona-everest-cli/releases/download/${latest_release}/everestctl-$os-$arch"
curl -sL  https://github.com/percona/percona-everest-cli/releases/download/${latest_release}/everestctl-$os-$arch -o everestctl
chmod +x everestctl
echo "Deploying Backends using docker compose"
curl -sL  https://raw.githubusercontent.com/percona/percona-everest-backend/main/quickstart.yml -o quickstart.yml
docker compose -f quickstart.yml up -d


# If KUBECONFIG is set let the user know we are using it
if [[ -n "${KUBECONFIG}" ]]; then
	echo "Using KUBECONFIG: ${KUBECONFIG}"
else
	echo "KUBECONFIG is not set. Using default k8s cluster"
fi

echo "Provisioning Everest with monitoring disabled"
echo "If you want to enable monitoring please refer to the everest installation documentation."
echo ""
./everestctl install operators --backup.enable=false --everest.endpoint=http://127.0.0.1:8080  --monitoring.enable=false --operator.mongodb=true --operator.postgresql=true --operator.xtradb-cluster=true --skip-wizard

if [[ $os == "linux" ]]
then
	echo "Your provisioned Everest instance is available at http://127.0.0.1:8080"
	exit
fi
open http://127.0.0.1:8080
