/*
Copyright 2016 The Kubernetes Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"crypto/tls"
	"flag"
	"log"
	"strings"
	"time"

	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/pkg/tlsutil"
	"golang.org/x/net/context"
)

var (
	etcdCACert    = flag.String("cacert", "", "Etcd cacert")
	etcdCert      = flag.String("cert", "", "Etcd cert")
	etcdKey       = flag.String("key", "", "Etcd key")
	etcdAddress   = flag.String("etcd-address", "", "Etcd address")
	scanTimeout   = flag.Duration("timeout", 60*time.Second, "Etcd scan timeout default is 60s")
	ttlKeysPrefix = flag.String("ttl-keys-prefix", "", "Prefix for TTL keys")
	leaseDuration = flag.Duration("lease-duration", time.Hour, "Lease duration (seconds granularity)")
)

func main() {
	flag.Parse()

	if *etcdAddress == "" {
		log.Fatalf("--etcd-address flag is required")
	}

	var cfg *tls.Config

	cfg = nil
	if *etcdCert != "" {
		cfg = &tls.Config{}
		cs := make([]string, 0)
		if *etcdCACert != "" {
			cs = append(cs, *etcdCACert)
		}
		var err error
		cfg.RootCAs, err = tlsutil.NewCertPool(cs)
		if err != nil {
			log.Fatalf("Error while creating etcd tlsconfig: %v", err)
		}
		cfg.GetClientCertificate = func(unused *tls.CertificateRequestInfo) (cert *tls.Certificate, err error) {
			cert, err = tlsutil.NewCert(*etcdCert, *etcdKey, nil)
			return cert, err
		}
	}

	log.Printf("Connecting to etcd %v", *etcdAddress)
	client, err := clientv3.New(clientv3.Config{Endpoints: []string{*etcdAddress}, TLS: cfg})
	if err != nil {
		log.Fatalf("Error while creating etcd client: %v", err)
	}

	// Make sure that ttlKeysPrefix is ended with "/" so that we only get children "directories".
	if !strings.HasSuffix(*ttlKeysPrefix, "/") {
		*ttlKeysPrefix += "/"
	}
	ctx, cancelFunc := context.WithTimeout(context.Background(), *scanTimeout)
	defer cancelFunc()

	log.Printf("Getting all keys from etcd with prefix %s with timeout %v", *ttlKeysPrefix, *scanTimeout)
	objectsResp, err := client.KV.Get(ctx, *ttlKeysPrefix, clientv3.WithPrefix())
	if err != nil {
		log.Fatalf("Error while getting objects to attach to the lease")
	}

	lease, err := client.Lease.Grant(ctx, int64(*leaseDuration/time.Second))
	if err != nil {
		log.Fatalf("Error while creating lease: %v", err)
	}
	log.Printf("Lease with TTL: %v created", lease.TTL)

	log.Printf("Attaching lease to %d entries", len(objectsResp.Kvs))
	for _, kv := range objectsResp.Kvs {
		_, err := client.KV.Put(ctx, string(kv.Key), string(kv.Value), clientv3.WithLease(lease.ID))
		if err != nil {
			log.Printf("Error while attaching lease to: %s", string(kv.Key))
		}
	}
}
