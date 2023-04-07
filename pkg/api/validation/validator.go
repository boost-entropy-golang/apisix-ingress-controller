// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package validation

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-multierror"
	kwhmodel "github.com/slok/kubewebhook/v2/pkg/model"
	kwhvalidating "github.com/slok/kubewebhook/v2/pkg/webhook/validating"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/apisix-ingress-controller/pkg/apisix"
	v2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2"
	"github.com/apache/apisix-ingress-controller/pkg/log"
)

var (
	ApisixRouteV2GVR = metav1.GroupVersionResource{
		Group:    v2.GroupVersion.Group,
		Version:  v2.GroupVersion.Version,
		Resource: "apisixroutes",
	}

	ApisixPluginConfigV2GVR = metav1.GroupVersionResource{
		Group:    v2.GroupVersion.Group,
		Version:  v2.GroupVersion.Version,
		Resource: "apisixpluginconfigs",
	}
)

var Validator = kwhvalidating.ValidatorFunc(
	func(ctx context.Context, review *kwhmodel.AdmissionReview, object metav1.Object) (result *kwhvalidating.ValidatorResult, err error) {
		GVR := review.RequestGVR
		log.Debugw("arrive validator webhook", zap.Any("object", object))

		var valid bool
		var resultErr error
		var msg string

		switch *GVR {
		case ApisixRouteV2GVR:
			ar := object.(*v2.ApisixRoute)
			valid, resultErr = ValidateApisixRouteV2(ar)
		case ApisixPluginConfigV2GVR:
			apc := object.(*v2.ApisixPluginConfig)
			valid, resultErr = ValidateApisixPluginConfigV2(apc)
		default:
			valid = false
			resultErr = fmt.Errorf("{group: %s, version: %s, Resource: %s} not supported", GVR.Group, GVR.Version, GVR.Resource)
		}
		if resultErr != nil {
			msg = resultErr.Error()
		}
		return &kwhvalidating.ValidatorResult{
			Valid:   valid,
			Message: msg,
		}, nil
	},
)

func ValidateApisixRoutePlugins(plugins []v2.ApisixRoutePlugin) (valid bool, resultErr error) {
	valid = true
	client, err := GetSchemaClient(&apisix.ClusterOptions{})
	if err != nil {
		msg := "failed to get the schema client"
		log.Errorf("%s: %s", msg, err)
		return false, fmt.Errorf(msg)
	}

	for _, plugin := range plugins {
		if plugin.Enable {
			pluginConfig := plugin.Config
			if pluginConfig == nil {
				pluginConfig = map[string]interface{}{}
			}
			if v, err := ValidatePlugin(client, plugin.Name, pluginConfig); !v {
				valid = false
				resultErr = multierror.Append(resultErr, err)
				log.Warnf("failed to validate plugin %s: %s", plugin.Name, err)
			}
		}
	}
	return
}
