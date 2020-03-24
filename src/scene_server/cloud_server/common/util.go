/*
 * Tencent is pleased to support the open source community by making 蓝鲸 available.
 * Copyright (C) 2017-2018 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package common

import (
	"net/http"
	"strings"

	"configcenter/src/common"
	"configcenter/src/common/http/rest"
	"configcenter/src/common/util"
)

// 将不同云厂商的实例状态转为统一的实例状态
func CovertInstState(instState string) string {
	switch strings.ToLower(instState) {
	case "starting", "pending", "rebooting":
		return "starting"
	case "running":
		return "running"
	case "stopping", "shutting-down", "terminating":
		return "stopping"
	case "stopped", "shutdown", "terminated":
		return "stopped"
	default:
		return "unknow"
	}
	return instState
}

// 获取api调用的header
func GetHeader() http.Header {
	header := make(http.Header)
	header.Add(common.BKHTTPOwnerID, "0")
	header.Add(common.BKSupplierIDField, "0")
	header.Add(common.BKHTTPHeaderUser, "admin")
	header.Add(common.BKHTTPLanguage, "cn")
	header.Add("Content-Type", "application/json")
	return header
}

// 获取api调用的header
func GetKit(header http.Header) *rest.Kit {
	ctx := util.NewContextFromHTTPHeader(header)
	rid := util.GetHTTPCCRequestID(header)
	user := util.GetUser(header)
	supplierAccount := util.GetOwnerID(header)
	defaultCCError := util.GetDefaultCCError(header)

	return &rest.Kit{
		Rid:             rid,
		Header:          header,
		Ctx:             ctx,
		CCError:         defaultCCError,
		User:            user,
		SupplierAccount: supplierAccount,
	}
}
