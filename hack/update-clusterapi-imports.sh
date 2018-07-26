#!/usr/bin/env bash
# vim: tw=500

set -eux

cd $(dirname $0)/..

for clusterapi_package in pkg/controller/machineset; do
  sed -i 's#sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset#github.com/kubermatic/machine-controller/pkg/client/clientset/versioned#g' $clusterapi_package/*
  sed -i 's#sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1#github.com/kubermatic/machine-controller/pkg/machines/v1alpha1#g' $clusterapi_package/*
  sed -i 's#sigs.k8s.io/cluster-api/pkg/client/listers_generated/cluster/v1alpha1#github.com/kubermatic/machine-controller/pkg/client/listers/machines/v1alpha1#g' $clusterapi_package/*
  sed -i 's#util.Poll#wait.Poll#g' $clusterapi_package/*
  sed -i 's#sigs.k8s.io/cluster-api/pkg/util#k8s.io/apimachinery/pkg/util/wait#g' $clusterapi_package/*
done
