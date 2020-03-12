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

package service

import (
	"fmt"
	"reflect"
	"strconv"

	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/common/errors"
	"configcenter/src/common/http/rest"
	"configcenter/src/common/mapstr"
	"configcenter/src/common/metadata"
	"configcenter/src/common/paraparse"
)

func (s *Service) CreateResourceDirectory(ctx *rest.Contexts) {
	data := mapstr.MapStr{}
	if err := ctx.DecodeInto(&data); nil != err {
		blog.Errorf("CreateResourceDirectory failed to parse the params, error info is %s, rid: %s", err.Error(), ctx.Kit.Rid)
		ctx.RespErrorCodeOnly(common.CCErrCommJSONUnmarshalFailed, "")
		return
	}

	// 给资源池目录加上资源池(业务id)和空闲机池（集群id）, service_category_id, service_template_id
	bizID, setID, err := s.getResourcePoolIDAndSetID(ctx)
	if err != nil {
		blog.ErrorJSON("CreateResourceDirectory fail with getResourcePoolIDAndSetID failed, err: %v, rid: %s", err, ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}
	data[common.BKAppIDField] = bizID
	data[common.BKSetIDField] = setID
	data[common.BKServiceCategoryIDField] = 0
	data[common.BKServiceTemplateIDField] = 0

	// 资源池目录的default设置为4
	data[common.BKDefaultField] = 4
	input := &metadata.CreateModelInstance{Data: data}
	rsp, err := s.Engine.CoreAPI.CoreService().Instance().CreateInstance(ctx.Kit.Ctx, ctx.Kit.Header, common.BKInnerObjIDModule, input)
	if err != nil {
		blog.ErrorJSON("CreateResourceDirectory, failed to CreateInstance, err: %s, rid: %s", err.Error(), ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}
	if !rsp.Result {
		blog.ErrorJSON("CreateResourceDirectory, failed to CreateInstance, errMsg: %s, rid: %s", rsp.ErrMsg, ctx.Kit.Rid)
		ctx.RespAutoError(errors.New(rsp.Code, rsp.ErrMsg))
		return
	}

	ctx.RespEntity(rsp.Data)
}

func (s *Service) getResourcePoolIDAndSetID(ctx *rest.Contexts) (int64, int64, error) {
	query := &metadata.QueryCondition{
		Condition: mapstr.MapStr{common.BKDefaultField: 1},
	}
	bizRsp, err := s.Engine.CoreAPI.CoreService().Instance().ReadInstance(ctx.Kit.Ctx, ctx.Kit.Header, common.BKInnerObjIDApp, query)
	if err != nil {
		blog.ErrorJSON("getResourcePoolIDAndSetID, failed to find business by query condition: %s, err: %s, rid: %s", query, err.Error(), ctx.Kit.Rid)
		return 0, 0, err
	}
	if !bizRsp.Result {
		return 0, 0, ctx.Kit.CCError.New(bizRsp.Code, bizRsp.ErrMsg)
	}
	if bizRsp.Data.Count <= 0 {
		return 0, 0, fmt.Errorf("get resource pool info success, but count < 0")
	}

	intBizID, err := bizRsp.Data.Info[0].Int64(common.BKAppIDField)
	if err != nil {
		blog.Errorf("getResourcePoolIDAndSetID, bizID convert to float64 failed, err:%v, rid: %v", err, ctx.Kit.Rid)
		return 0, 0, err
	}

	query.Condition = mapstr.MapStr{common.BKAppIDField: intBizID}
	setRsp, err := s.Engine.CoreAPI.CoreService().Instance().ReadInstance(ctx.Kit.Ctx, ctx.Kit.Header, common.BKInnerObjIDSet, query)
	if err != nil {
		blog.ErrorJSON("getResourcePoolIDAndSetID, failed to find business by query condition: %s, err: %s, rid: %s", query, err.Error(), ctx.Kit.Rid)
		return 0, 0, err
	}
	if !setRsp.Result {
		return 0, 0, ctx.Kit.CCError.New(setRsp.Code, setRsp.ErrMsg)
	}
	if setRsp.Data.Count <= 0 {
		return 0, 0, fmt.Errorf("get set info success, but count < 0")
	}

	intSetID, err := setRsp.Data.Info[0].Int64(common.BKSetIDField)
	if err != nil {
		blog.Errorf("getResourcePoolIDAndSetID, setID convert to float64 failed, err:%v, rid: %v", err, ctx.Kit.Rid)
		return 0, 0, err
	}

	return intBizID, intSetID, nil
}

func (s *Service) UpdateResourceDirectory(ctx *rest.Contexts) {
	input := mapstr.MapStr{}
	if err := ctx.DecodeInto(&input); nil != err {
		blog.Errorf("UpdateResourceDirectory failed to parse the params, error info is %s, rid: %s", err.Error(), ctx.Kit.Rid)
		ctx.RespErrorCodeOnly(common.CCErrCommJSONUnmarshalFailed, "")
		return
	}

	if !input.Exists(common.BKModuleNameField) {
		ctx.RespErrorCodeF(common.CCErrorTopoOnlyResourceDirNameCanBeUpdated, "UpdateResourceDirectory, field bk_module_name not exist, rid: %s", ctx.Kit.Rid)
		return
	}
	if len(input) > 1 {
		delete(input, common.BKModuleNameField)
		ctx.RespErrorCodeF(common.CCErrorTopoOnlyResourceDirNameCanBeUpdated, "UpdateResourceDirectory invalid params %s, rid: %s", input, ctx.Kit.Rid)
		return
	}
	moduleID := ctx.Request.PathParameter(common.BKModuleIDField)
	intModuleID, err := strconv.ParseInt(moduleID, 10, 64)
	if err != nil {
		blog.Errorf("DeleteResourceDirectory, moduleID convert to int64 failed, err:%v, rid: %v", err, ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}
	option := &metadata.UpdateOption{
		Data:      input,
		Condition: mapstr.MapStr{common.BKModuleIDField: intModuleID},
	}
	rsp, err := s.Engine.CoreAPI.CoreService().Instance().UpdateInstance(ctx.Kit.Ctx, ctx.Kit.Header, common.BKInnerObjIDModule, option)
	if err != nil {
		blog.Errorf("UpdateResourceDirectory failed, coreservice UpdateInstance http call fail, option: %v, err: %v, rid:%s", option.Data, err, ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}
	if !rsp.Result {
		blog.ErrorJSON("UpdateResourceDirectory, failed to UpdateInstance, errMsg: %s, rid: %s", rsp.ErrMsg, ctx.Kit.Rid)
		ctx.RespAutoError(errors.New(rsp.Code, rsp.ErrMsg))
		return
	}

	ctx.RespEntity(rsp.Data)
}

func (s *Service) SearchResourceDirectory(ctx *rest.Contexts) {
	input := new(metadata.SearchResourceDirParams)
	if err := ctx.DecodeInto(input); nil != err {
		blog.Errorf("SearchResourceDirectory failed to parse the params, error info is %s, rid: %s", err.Error(), ctx.Kit.Rid)
		ctx.RespErrorCodeOnly(common.CCErrCommJSONUnmarshalFailed, "")
		return
	}
	if input.Exact == false {
		for k, v := range input.Condition {
			if reflect.TypeOf(v).Kind() == reflect.String {
				field := v.(string)
				input.Condition[k] = mapstr.MapStr{
					common.BKDBLIKE: params.SpecialCharChange(field),
					"$options":      "i",
				}
			}
		}
	}

	bizID, setID, err := s.getResourcePoolIDAndSetID(ctx)
	if err != nil {
		blog.ErrorJSON("SearchResourceDirectory fail with getResourcePoolIDAndSetID failed, err: %v, rid: %s", err, ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}
	if len(input.Condition) == 0 {
		input.Condition = mapstr.MapStr{}
	}
	input.Condition[common.BKAppIDField] = bizID
	input.Condition[common.BKSetIDField] = setID
	query := &metadata.QueryCondition{Condition: input.Condition}
	rsp, err := s.Engine.CoreAPI.CoreService().Instance().ReadInstance(ctx.Kit.Ctx, ctx.Kit.Header, common.BKInnerObjIDModule, query)
	if err != nil {
		blog.Errorf("SearchResourceDirectory failed, coreservice http ReadInstance fail, input: %v, err: %v, %s", input, err, ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}
	if !rsp.Result {
		blog.ErrorJSON("SearchResourceDirectory, failed to SearchResourceDirectory, errMsg: %s, rid: %s", rsp.ErrMsg, ctx.Kit.Rid)
		ctx.RespAutoError(errors.New(rsp.Code, rsp.ErrMsg))
		return
	}

	moduleIDArr := make([]int64, 0)
	mapModuleIdInfo := make(map[int64]mapstr.MapStr)
	for _, item := range rsp.Data.Info {
		moduleID, err := item.Int64(common.BKModuleIDField)
		if err != nil {
			blog.ErrorJSON("SearchResourceDirectory fail with moduleID convert from interface to int64 failed, err: %v, rid: %s", err, ctx.Kit.Rid)
			ctx.RespAutoError(err)
			return
		}
		moduleIDArr = append(moduleIDArr, moduleID)
		mapModuleIdInfo[moduleID] = item
	}

	// count hosts
	listHostOption := &metadata.HostModuleRelationRequest{
		ApplicationID: bizID,
		SetIDArr:      []int64{setID},
		ModuleIDArr:   moduleIDArr,
		Page: metadata.BasePage{
			Limit: common.BKNoLimit,
		},
	}
	hostModuleRelations, e := s.Engine.CoreAPI.CoreService().Host().GetHostModuleRelation(ctx.Kit.Ctx, ctx.Kit.Header, listHostOption)
	if e != nil {
		blog.Errorf("GetInternalModuleWithStatistics failed, list host modules failed, option: %+v, err: %s, rid: %s", listHostOption, e.Error(), ctx.Kit.Rid)
		ctx.RespAutoError(e)
		return
	}
	moduleHostIDs := make(map[int64][]int64, 0)
	for _, relation := range hostModuleRelations.Data.Info {
		if _, ok := moduleHostIDs[relation.ModuleID]; ok == false {
			moduleHostIDs[relation.ModuleID] = make([]int64, 0)
		}
		moduleHostIDs[relation.ModuleID] = append(moduleHostIDs[relation.ModuleID], relation.HostID)
	}
	retInfo := make([]mapstr.MapStr, 0)
	for moduleID, moduleInfo := range mapModuleIdInfo {
		for id, hostIDs := range moduleHostIDs {
			moduleInfo["host_count"] = 0
			if moduleID == id {
				moduleInfo["host_count"] = len(hostIDs)
				break
			}
		}
		retInfo = append(retInfo, moduleInfo)
	}

	ret := make(map[string]interface{}, 0)
	ret["count"] = len(retInfo)
	ret["info"] = retInfo
	ctx.RespEntity(ret)
}

func (s *Service) DeleteResourceDirectory(ctx *rest.Contexts) {
	moduleID := ctx.Request.PathParameter(common.BKModuleIDField)
	intModuleID, err := strconv.ParseInt(moduleID, 10, 64)
	if err != nil {
		blog.Errorf("DeleteResourceDirectory, moduleID convert to int64 failed, err:%v, rid: %v", err, ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}

	bizID, setID, err := s.getResourcePoolIDAndSetID(ctx)

	hasHost, err := s.hasHost(ctx, bizID, []int64{setID}, []int64{intModuleID})
	if err != nil {
		blog.Errorf("DeleteResourceDirectory, check if resource directory has host failed, err: %v, rid: %s", err, ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}
	if hasHost {
		ctx.RespErrorCodeF(common.CCErrTopoHasHostCheckFailed, "DeleteResourceDirectory, resource directory has hosts, rid: %s", ctx.Kit.Rid)
		return
	}

	cond := &metadata.DeleteOption{Condition: mapstr.MapStr{common.BKModuleIDField: intModuleID}}
	rsp, err := s.Engine.CoreAPI.CoreService().Instance().DeleteInstance(ctx.Kit.Ctx, ctx.Kit.Header, common.BKInnerObjIDModule, cond)
	if err != nil {
		blog.Errorf("DeleteResourceDirectory, coreservice http DeleteInstance fail, bk_module_id: %d, err: %v, rid: %s")
		ctx.RespAutoError(err)
		return
	}
	if !rsp.Result {
		blog.ErrorJSON("DeleteResourceDirectory, failed to DeleteInstance, errMsg: %s, rid: %s", rsp.ErrMsg, ctx.Kit.Rid)
		ctx.RespAutoError(errors.New(rsp.Code, rsp.ErrMsg))
		return
	}

	ctx.RespEntity(rsp.Data)
}

func (s *Service) hasHost(ctx *rest.Contexts, bizID int64, setIDs, moduleIDS []int64) (bool, error) {
	option := &metadata.HostModuleRelationRequest{
		ApplicationID: bizID,
		ModuleIDArr:   moduleIDS,
	}
	if len(setIDs) > 0 {
		option.SetIDArr = setIDs
	}
	if len(moduleIDS) > 0 {
		option.ModuleIDArr = moduleIDS
	}
	rsp, err := s.Engine.CoreAPI.CoreService().Host().GetHostModuleRelation(ctx.Kit.Ctx, ctx.Kit.Header, option)
	if nil != err {
		blog.Errorf("[operation-module] failed to request the object controller, err: %s, rid: %s", err.Error(), ctx.Kit.Rid)
		return false, ctx.Kit.CCError.Error(common.CCErrCommHTTPDoRequestFailed)
	}

	if !rsp.Result {
		blog.Errorf("[operation-module]  failed to search the host module configures, err: %s, rid: %s", rsp.ErrMsg, ctx.Kit.Rid)
		return false, ctx.Kit.CCError.New(rsp.Code, rsp.ErrMsg)
	}

	return 0 != len(rsp.Data.Info), nil
}
