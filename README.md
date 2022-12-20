# KubeProxyCtl
Kubectl Proxy Modifying Multiple Clusters
## 查询集群
kubectl --kubeconfig hw get cluster
## 查询某个集群的 nodes
kubectl --kubeconfig hw get node --selector "cluster=hw"
## 查询多个集群的 pods
kubectl --kubeconfig hw get node --selector "cluster=hw_tx"