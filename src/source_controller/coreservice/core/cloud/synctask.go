/*
 * Tencent is pleased to support the open source community by making 蓝鲸 available.,
 * Copyright (C) 2017-2018 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the ",License",); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an ",AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cloud

import (
	"configcenter/src/common/util"
	"time"

	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/common/errors"
	"configcenter/src/common/http/rest"
	"configcenter/src/common/mapstr"
	"configcenter/src/common/metadata"
)

func (c *cloudOperation) CreateSyncTask(kit *rest.Kit, task *metadata.CloudSyncTask) (*metadata.CloudSyncTask, errors.CCErrorCoder) {
	if err := c.validCreateSyncTask(kit, task); nil != err {
		blog.Errorf("CreateAccount failed, valid error: %+v, rid: %s", err, kit.Rid)
		return nil, err
	}

	cloudVendor, errVendor := c.getSyncTaskCloudVendor(kit, task.AccountID)
	if errVendor != nil {
		blog.ErrorJSON("CreateSyncTask getSyncTaskCloudVendor failed, taskName: %s, err: %v, rid: %s", task.TaskName, errVendor, kit.Rid)
		return nil, errVendor
	}
	task.CloudVendor = cloudVendor

	id, err := c.dbProxy.NextSequence(kit.Ctx, common.BKTableNameCloudSyncTask)
	if nil != err {
		blog.Errorf("CreateSyncTask failed, generate id failed, err: %+v, rid: %s", err, kit.Rid)
		return nil, kit.CCError.CCErrorf(common.CCErrCommGenerateRecordIDFailed)
	}
	task.TaskID = int64(id)
	ts := time.Now()
	task.OwnerID = kit.SupplierAccount
	task.LastEditor = task.Creator
	task.CreateTime = ts
	task.LastTime = ts

	err = c.dbProxy.Table(common.BKTableNameCloudSyncTask).Insert(kit.Ctx, task)
	if err != nil {
		blog.ErrorJSON("CreateSyncTask failed, db insert failed, taskName: %s, err: %v, rid: %s", task.TaskName, err, kit.Rid)
		return nil, kit.CCError.CCError(common.CCErrCommDBInsertFailed)
	}

	return task, nil
}

func (c *cloudOperation) SearchSyncTask(kit *rest.Kit, option *metadata.SearchCloudOption) (*metadata.MultipleCloudSyncTask, errors.CCErrorCoder) {
	results := make([]metadata.CloudSyncTask, 0)
	err := c.dbProxy.Table(common.BKTableNameCloudSyncTask).Find(option.Condition).Fields(option.Fields...).
		Start(uint64(option.Page.Start)).Limit(uint64(option.Page.Limit)).Sort(option.Page.Sort).All(kit.Ctx, &results)
	if err != nil {
		blog.ErrorJSON("SearchSyncTask failed, db find failed, option: %v, err: %v, rid: %s", option, err, kit.Rid)
		return nil, kit.CCError.CCError(common.CCErrCommDBSelectFailed)
	}

	// 任务总个数
	count, err := c.countTask(kit, option.Condition)
	if err != nil {
		blog.ErrorJSON("SearchSyncTask countTask error %v, option: %v, rid: %s", err, option.Condition, kit.Rid)
		return nil, kit.CCError.CCError(common.CCErrCommDBSelectFailed)
	}

	return &metadata.MultipleCloudSyncTask{Count: int64(count), Info: results}, nil
}

func (c *cloudOperation) UpdateSyncTask(kit *rest.Kit, taskID int64, option mapstr.MapStr) errors.CCErrorCoder {
	if err := c.validUpdateSyncTask(kit, taskID, option); nil != err {
		blog.Errorf("UpdateSyncTask failed, valid error: %+v, rid: %s", err, kit.Rid)
		return err
	}

	filter := map[string]int64{common.BKCloudSyncTaskID: taskID}
	option.Set(common.LastTimeField, time.Now())
	// 确保不会更新云厂商类型、云账户id、开发商id
	option.Remove(common.BKCloudVendor)
	option.Remove(common.BKCloudIDField)
	option.Remove(common.BKOwnerIDField)
	if e := c.dbProxy.Table(common.BKTableNameCloudSyncTask).Update(kit.Ctx, filter, option); e != nil {
		blog.Errorf("UpdateSyncTask failed, mongodb failed, table: %s, filter: %+v, err: %+v, rid: %s", common.BKTableNameCloudSyncTask, filter, e, kit.Rid)
		return kit.CCError.CCError(common.CCErrCommDBUpdateFailed)
	}
	return nil
}

func (c *cloudOperation) DeleteSyncTask(kit *rest.Kit, taskID int64) errors.CCErrorCoder {
	cond := map[string]int64{common.BKCloudSyncTaskID: taskID}
	if e := c.dbProxy.Table(common.BKTableNameCloudSyncTask).Delete(kit.Ctx, cond); e != nil {
		blog.Errorf("DeleteSyncTask failed, mongodb operate failed, table: %s, filter: %+v, err: %+v, rid: %s", common.BKTableNameCloudAccount, cond, e, kit.Rid)
		return kit.CCError.CCError(common.CCErrCommDBDeleteFailed)
	}
	return nil
}

func (c *cloudOperation) SearchSyncHistory(kit *rest.Kit, option *metadata.SearchSyncHistoryOption) (*metadata.MultipleSyncHistory, errors.CCErrorCoder) {
	results := make([]metadata.SyncHistory, 0)
	// 设置查询条件
	cond := option.Condition
	cond.Set(common.BKCloudSyncTaskID, option.TaskID)
	if option.StarTime != "" {
		cond.Set(common.CreateTimeField, mapstr.MapStr{common.BKDBGTE: option.StarTime})
	}
	if option.EndTime != "" {
		cond.Set(common.CreateTimeField, mapstr.MapStr{common.BKDBLTE: option.EndTime})
	}

	err := c.dbProxy.Table(common.BKTableNameCloudSyncHistory).Find(cond).Fields(option.Fields...).
		Start(uint64(option.Page.Start)).Limit(uint64(option.Page.Limit)).Sort(option.Page.Sort).All(kit.Ctx, &results)
	if err != nil {
		blog.ErrorJSON("SearchSyncHistory failed, db find failed, option: %v, err: %v, rid: %s", option, err, kit.Rid)
		return nil, kit.CCError.CCError(common.CCErrCommDBSelectFailed)
	}

	// 同步历史记录总个数
	count, err := c.countHistory(kit, cond)
	if err != nil {
		blog.ErrorJSON("SearchSyncHistory countHistory error %v, cond: %v, rid: %s", err, cond, kit.Rid)
		return nil, kit.CCError.CCError(common.CCErrCommDBSelectFailed)
	}

	return &metadata.MultipleSyncHistory{Count: int64(count), Info: results}, nil
}

func (c *cloudOperation) countTask(kit *rest.Kit, cond mapstr.MapStr) (uint64, error) {
	cond = util.SetQueryOwner(cond, kit.SupplierAccount)
	count, err := c.dbProxy.Table(common.BKTableNameCloudSyncTask).Find(cond).Count(kit.Ctx)
	return count, err

}

func (c *cloudOperation) countHistory(kit *rest.Kit, cond mapstr.MapStr) (uint64, error) {
	cond = util.SetQueryOwner(cond, kit.SupplierAccount)
	count, err := c.dbProxy.Table(common.BKTableNameCloudSyncHistory).Find(cond).Count(kit.Ctx)
	return count, err

}

func (c *cloudOperation) getSyncTaskCloudVendor(kit *rest.Kit, accountID int64) (string, errors.CCErrorCoder) {
	result := new(metadata.CloudAccount)
	cond := map[string]interface{}{common.BKCloudAccountID: accountID}
	err := c.dbProxy.Table(common.BKTableNameCloudAccount).Find(cond).One(kit.Ctx, result)
	if err != nil {
		blog.ErrorJSON("getSyncTaskCloudVendor failed, db operate failed, cond: %v, err: %v, rid: %s", cond, err, kit.Rid)
		return "", kit.CCError.CCError(common.CCErrCommDBSelectFailed)
	}

	return result.CloudVendor, nil
}