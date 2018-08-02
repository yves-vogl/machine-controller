#!/usr/bin/env bash
# vim: tw=500

set -eux

cd $(dirname $0)/..

for clusterapi_sourcefile in pkg/machines/v1alpha1/machineset_types.go; do
	sed -i 's#sigs.k8s.io/cluster-api/pkg/apis/cluster/common#github.com/kubermatic/machine-controller/pkg/machines/common#' $clusterapi_sourcefile
done

sed -i '#github.com/kubernetes-incubator/apiserver-builder/pkg/controller#github.com/kubermatic/machine-controller/pkg/apiserver-builder/pkg/controller' \
  pkg/controller/sharedinformers/zz_generated.api.register.go
sed -i 's#controller.SharedInformersDefaults#SharedInformersDefaults#g' pkg/controller/sharedinformers/zz_generated.api.register.go
sed -i 's#"sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset"#clientset "github.com/kubermatic/machine-controller/pkg/client/clientset/versioned"#g' \
  pkg/controller/sharedinformers/zz_generated.api.register.go
sed -i 's#sigs.k8s.io/cluster-api/pkg/client/informers_generated/externalversions#github.com/kubermatic/machine-controller/pkg/client/informers/externalversions#g' \
  pkg/controller/sharedinformers/zz_generated.api.register.go
sed -i 's#Cluster().V1alpha1()#Machine().V1alpha1()#g' pkg/controller/sharedinformers/zz_generated.api.register.go
sed -i 's#Cluster().V1alpha1()#Machine().V1alpha1()#g' pkg/controller/machineset/zz_generated.api.register.go
sed -i 's#Cluster().V1alpha1()#Machine().V1alpha1()#g' pkg/controller/machineset/controller.go
sed -i '/si.Factory.Machine().V1alpha1().MachineDeployments().Informer().Run(shutdown)/d' pkg/controller/sharedinformers/zz_generated.api.register.go
sed -i '/si.Factory.Machine().V1alpha1().Clusters().Informer().Run(shutdown)/d' pkg/controller/sharedinformers/zz_generated.api.register.go

go fmt pkg/controller/sharedinformers/zz_generated.api.register.go
