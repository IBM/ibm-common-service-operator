//
// Copyright 2022 IBM Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package constant

var (
	CertManagerAPIGroupVersionV1Alpha1 = "certmanager.k8s.io/v1alpha1"
	CertManagerAPIGroupVersionV1       = "cert-manager.io/v1"
	CertManagerKinds                   = []string{"Issuer", "Certificate"}
	CertManagerIssuers                 = []string{CSSSIssuer, CSCAIssuer}
	CertManagerCerts                   = []string{CSCACert}
)

// CSCAIssuer is the CR of cs-ca-issuer
const CSCAIssuer = `
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  labels:
    app.kubernetes.io/instance: cs-ca-issuer
    app.kubernetes.io/managed-by: cert-manager-controller
    app.kubernetes.io/name: Issuer
  name: cs-ca-issuer
  namespace: placeholder
spec:
  ca:
    secretName: cs-ca-certificate-secret
`

// CSSSIsuuer is the CR of cs-ss-issuer
const CSSSIssuer = `
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  labels:
    app.kubernetes.io/instance: cs-ss-issuer
    app.kubernetes.io/managed-by: cert-manager-controller
    app.kubernetes.io/name: Issuer
  name: cs-ss-issuer
  namespace: placeholder
spec:
  selfSigned: {}
`

// CSCACert is the CR of cs-ca-certificate
const CSCACert = `
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  labels:
    app.kubernetes.io/instance: cs-ca-certificate
    app.kubernetes.io/managed-by: cert-manager-controller
    app.kubernetes.io/name: Certificate
    ibm-cert-manager-operator/refresh-ca-chain: 'true'
  name: cs-ca-certificate
  namespace: placeholder
spec:
  secretName: cs-ca-certificate-secret
  secretTemplate:
    labels:
      ibm-cert-manager-operator/refresh-ca-chain: 'true'
  issuerRef:
    name: cs-ss-issuer
    kind: Issuer
  commonName: cs-ca-certificate
  isCA: true
  duration: 17520h0m0s
  renewBefore: 720h0m0s
`
