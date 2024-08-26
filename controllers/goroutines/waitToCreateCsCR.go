//
// Copyright 2023 IBM Corporation
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

package goroutines

import (
	"fmt"
	"os"
	"strings"
	"time"

	"k8s.io/klog"

	"github.com/IBM/ibm-common-service-operator/v4/controllers/bootstrap"
)

// WaitToCreateCsCR waits for the creation of the CommonService CR in the operator namespace.
func WaitToCreateCsCR(bs *bootstrap.Bootstrap) {
	for {
		klog.Infof("Start to Create CommonService CR in the namespace %s", bs.CSData.OperatorNs)
		if err := bs.CreateCsCR(); err != nil {
			if strings.Contains(fmt.Sprint(err), "failed to call webhook") {
				klog.Infof("Webhook Server not ready, waiting for it to be ready : %v", err)
				time.Sleep(time.Second * 20)
			} else {
				klog.Errorf("Failed to create CommonService CR : %v", err)
				os.Exit(1)
			}
		} else {
			break
		}

	}
}
