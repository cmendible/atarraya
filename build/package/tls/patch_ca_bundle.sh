#!/bin/bash
# Orginal file here: https://github.com/morvencao/kube-mutating-webhook-tutorial/tree/master/deployment

set -o errexit
set -o nounset
set -o pipefail

CA_BUNDLE=$(kubectl config view --raw --minify --flatten -o jsonpath='{.clusters[].cluster.certificate-authority-data}')
sed -i'' -e "s|__CA_BUNDLE_BASE64__|$CA_BUNDLE|g" ../k8s/mutating-webhook-configuration.yaml