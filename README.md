# etcdattachlease
Attach lease to etcd v3 key prefix, this code is copied from https://github.com/kubernetes/kubernetes/blob/release-1.10/cluster/images/etcd/attachlease/attachlease.go, creating it here since just require the binary

Generated linux binary so that bash scripts can directly download and run

```bash
curl --retry 5 --retry-delay 10 --retry-max-time 30 --silent -L https://raw.githubusercontent.com/awesomenix/etcdattachlease/master/attachlease -o /tmp/attachlease && chmod +x /tmp/attachlease
```
