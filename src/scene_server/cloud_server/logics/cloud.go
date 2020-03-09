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

package logics

import (
	"context"
	"fmt"
	"sync"

	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/common/mapstr"
	"configcenter/src/common/metadata"
	"configcenter/src/scene_server/cloud_server/cloudvendor"
	ccom "configcenter/src/scene_server/cloud_server/common"
)

func (lgc *Logics) AccountVerify(conf ccom.AccountConf) (bool, error) {
	client, err := cloudvendor.GetVendorClient(conf)
	if err != nil {
		blog.Errorf("AccountVerify GetVendorClient err:%s", err.Error())
		return false, err
	}

	_, err = client.GetRegions(nil)
	if err != nil {
		blog.Errorf("AccountVerify GetRegions err:%s", err.Error())

		return false, err
	}

	return true, nil
}

// 获取地域下的vpc详情和主机数
func (lgc *Logics) GetVpcHostCnt(region string, conf ccom.AccountConf) (*metadata.VpcHostCntResult, error) {
	client, err := cloudvendor.GetVendorClient(conf)
	if err != nil {
		blog.Errorf("GetVpcHostCnt GetVendorClient err:%s", err.Error())
		return nil, err
	}

	vpcsInfo, err := client.GetVpcs(region, nil)
	if err != nil {
		blog.Errorf("GetVpcHostCnt GetVpcs err:%s", err.Error())
		return nil, err
	}

	vpcHostCnt := make(map[string]int64)
	hostCntChan := make(chan []interface{}, 10)
	var wg, wg2 sync.WaitGroup
	// 并发请求获取每个vpc的实例个数
	for _, vpc := range vpcsInfo.VpcSet {
		wg.Add(1)
		go func(vpc *metadata.Vpc) {
			defer wg.Done()
			instancesInfo, err := client.GetInstances(region, &ccom.RequestOpt{
				Filters: []*ccom.Filter{&ccom.Filter{ccom.StringPtr("vpc-id"), ccom.StringPtrs([]string{vpc.VpcId})}},
				Limit:   ccom.Int64Ptr(ccom.MaxLimit),
			})
			if err != nil {
				blog.Errorf("GetVpcHostCnt GetInstances err:%s", err.Error())
				return
			}
			hostCntChan <- []interface{}{vpc.VpcId, instancesInfo.Count}
		}(vpc)
	}
	wg2.Add(1)
	go func() {
		defer wg2.Done()
		for hostCnt := range hostCntChan {
			vpcHostCnt[hostCnt[0].(string)] = hostCnt[1].(int64)
		}
	}()
	wg.Wait()
	close(hostCntChan)
	wg2.Wait()

	result := new(metadata.VpcHostCntResult)
	result.Count = vpcsInfo.Count
	for i, _ := range vpcsInfo.VpcSet {
		vpc := vpcsInfo.VpcSet[i]
		result.Info = append(result.Info, metadata.VpcSyncInfo{
			VpcID:        vpc.VpcId,
			VpcName:      vpc.VpcName,
			Region:       region,
			VpcHostCount: vpcHostCnt[vpc.VpcId],
		})
	}

	return result, nil
}

// 获取地域下的vpc和主机详情
func (lgc *Logics) GetCloudHostResource(syncVpcs []metadata.VpcSyncInfo, conf ccom.AccountConf) (*metadata.CloudHostResource, error) {
	client, err := cloudvendor.GetVendorClient(conf)
	if err != nil {
		blog.Errorf("GetVpcHostCnt GetVendorClient err:%s", err.Error())
		return nil, err
	}

	blog.V(4).Infof("GetCloudHostResource syncVpcs %#v", syncVpcs)
	vpcHostDetail := make(map[string][]*metadata.Instance)
	hostDetailChan := make(chan []*metadata.Instance, 10)
	var wg, wg2 sync.WaitGroup
	// 并发请求获取每个vpc的实例详情
	for _, vpc := range syncVpcs {
		wg.Add(1)
		go func(vpc metadata.VpcSyncInfo) {
			defer wg.Done()
			instancesInfo, err := client.GetInstances(vpc.Region, &ccom.RequestOpt{
				Filters: []*ccom.Filter{&ccom.Filter{ccom.StringPtr("vpc-id"), ccom.StringPtrs([]string{vpc.VpcID})}},
				Limit:   ccom.Int64Ptr(ccom.MaxLimit),
			})
			if err != nil {
				blog.Errorf("GetCloudHostResource GetInstances err:%s", err.Error())
				return
			}
			blog.V(4).Infof("GetCloudHostResource vpc-id:%s, instances count %#v", vpc.VpcID, instancesInfo.Count)
			hostDetailChan <- instancesInfo.InstanceSet
		}(vpc)
	}
	wg2.Add(1)
	go func() {
		defer wg2.Done()
		for hostDetail := range hostDetailChan {
			if len(hostDetail) > 0 {
				vpcHostDetail[hostDetail[0].VpcId] = hostDetail
			}

		}
	}()
	wg.Wait()
	close(hostDetailChan)
	wg2.Wait()

	result := new(metadata.CloudHostResource)

	for i, _ := range syncVpcs {
		vpc := syncVpcs[i]
		result.HostResource = append(result.HostResource, &metadata.VpcInstances{
			Vpc:       &vpc,
			Instances: vpcHostDetail[vpc.VpcID],
		})
	}

	return result, nil
}

// 获取云厂商账户配置
func (lgc *Logics) GetAccountConf(accountID int64) (*ccom.AccountConf, error) {
	result := []ccom.AccountConf{}
	option := mapstr.MapStr{common.BKCloudAccountID: accountID}
	err := lgc.db.Table(common.BKTableNameCloudAccount).Find(option).All(context.Background(), &result)
	if err != nil {
		blog.Errorf("GetAccountConf failed, accountID: %v, err: %s", accountID, err.Error())
		return nil, err
	}
	if len(result) == 0 {
		blog.Errorf("GetAccountConf failed, accountID: %v is not exist", accountID)
		return nil, fmt.Errorf("GetAccountConf failed, accountID: %v is not exist", accountID)
	}
	return &result[0], nil
}