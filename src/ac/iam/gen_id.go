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

package iam

import (
	"errors"
	"fmt"
	"strconv"

	"configcenter/src/ac/meta"
	"configcenter/src/scene_server/auth_server/sdk/types"
)

func genIamResource(act ActionID, rscType TypeID, a *meta.ResourceAttribute) (*types.Resource, error) {

	switch a.Basic.Type {
	case meta.Business:
		return genBusinessResource(act, rscType, a)
	case meta.DynamicGrouping:
		return genDynamicGroupingResource(act, rscType, a)
	case meta.ProcessServiceCategory:
		return genProcessServiceCategoryResource(act, rscType, a)
	case meta.EventPushing:
		return genEventSubscribeResource(act, rscType, a)
		// case meta.Model:
		// 	return genModelResource(act, rscType, a)
		// case meta.ModelModule:
		// 	return genModelModuleResource(act, rscType, a)
		// case meta.ModelSet:
		// 	return genModelSetResource(act, rscType, a)
		// case meta.MainlineModel:
		// 	return genMainlineModelResource(act, rscType, a)
		// case meta.MainlineModelTopology:
		// 	return genMainlineModelTopologyResource(act, rscType, a)
		// case meta.MainlineInstanceTopology:
		// 	return genMainlineInstanceTopologyResource(act, rscType, a)
		// case meta.AssociationType:
		// 	return genAssociationTypeResource(act, rscType, a)
		// case meta.ModelAssociation:
		// 	return genModelAssociationResource(act, rscType, a)
		// case meta.ModelInstanceAssociation:
		// 	return genModelInstanceAssociationResource(act, rscType, a)
		// case meta.ModelInstance, meta.MainlineInstance:
		// 	return genModelInstanceResource(act, rscType, a)
		// case meta.ModelInstanceTopology:
		// 	return genModelInstanceTopologyResource(act, rscType, a)
		// case meta.ModelTopology:
		// 	return genModelTopologyResource(act, rscType, a)
		// case meta.ModelClassification:
		// 	return genModelClassificationResource(act, rscType, a)
		// case meta.ModelAttributeGroup:
		// 	return genModelAttributeGroupResource(act, rscType, a)
		// case meta.ModelAttribute:
		// 	return genModelAttributeResource(act, rscType, a)
		// case meta.ModelUnique:
		// 	return genModelUniqueResource(act, rscType, a)
		// case meta.UserCustom:
		// 	return genHostUserCustomResource(act, rscType, a)
		// case meta.HostFavorite:
		// 	return genHostFavoriteResource(act, rscType, a)
		// case meta.NetDataCollector:
		// 	return genNetDataCollectorResource(act, rscType, a)
		// case meta.HostInstance:
		// 	return genHostInstanceResource(act, rscType, a)
		// case meta.AuditLog:
		// 	return genAuditLogResource(act, rscType, a)
		// case meta.SystemBase:
		// 	return new(types.Resource), nil
		// case meta.Plat:
		// 	return genPlat(act, rscType, a)
		// case meta.Process:
		// 	return genProcessResource(act, rscType, a)
		// case meta.ProcessServiceInstance:
		// 	return genProcessServiceInstanceResource(act, rscType, a)
		// case meta.BizTopology:
		// 	return genBizTopologyResource(act, rscType, a)
		// case meta.ProcessTemplate:
		// 	return genProcessTemplateResource(act, rscType, a)
		// case meta.ProcessServiceTemplate:
		// 	return genProcessServiceTemplateResource(act, rscType, a)
		// case meta.SetTemplate:
		// 	return genSetTemplateResource(act, rscType, a)
		// case meta.OperationStatistic:
		// 	return genOperationStatisticResource(act, rscType, a)
		// case meta.HostApply:
		// 	return genHostApplyResource(act, rscType, a)
		// case meta.ResourcePoolDirectory:
		// 	return genResourcePoolDirectoryResource(act, rscType, a)
		// case meta.EventWatch:
		// 	return genResourceWatch(act, rscType, a)
		// case meta.CloudAccount:
		// 	return genCloudAccountResource(act, rscType, a)
		// case meta.CloudResourceTask:
		// 	return genCloudResourceTaskResource(act, rscType, a)
		// case meta.ConfigAdmin:
		// 	return genConfigAdminResource(act, rscType, a)
	}

	return nil, fmt.Errorf("gen id failed: unsupported resource type: %s", a.Type)
}

// generate business related resource id.
func genBusinessResource(act ActionID, resourceType TypeID, attribute *meta.ResourceAttribute) (*types.Resource, error) {
	r := &types.Resource{
		System:    SystemIDCMDB,
		Type:      types.ResourceType(resourceType),
		Attribute: nil,
	}

	// create business do not related to instance authorize
	if act == CreateBusiness {
		return r, nil
	}

	// if attribute.InstanceID <= 0 {
	// 	return nil, errors.New("business instance id is 0")
	// }

	r.ID = strconv.FormatInt(attribute.InstanceID, 10)

	return r, nil
}

func genDynamicGroupingResource(act ActionID, typ TypeID, att *meta.ResourceAttribute) (*types.Resource, error) {

	r := &types.Resource{
		System:    SystemIDCMDB,
		Attribute: nil,
	}

	if att.BusinessID <= 0 {
		return nil, errors.New("biz id can not be 0")
	}

	// do not related to instance authorize
	if act == CreateBusinessCustomQuery || act == FindBusinessCustomQuery {
		r.Type = types.ResourceType(Business)
		r.ID = strconv.FormatInt(att.BusinessID, 10)
		return r, nil
	}

	r.Type = types.ResourceType(typ)

	// authorize based on business
	r.Attribute = map[string]interface{}{
		types.IamPathKey: []string{fmt.Sprintf("/%s,%d/", Business, att.BusinessID)},
	}

	r.ID = att.InstanceIDEx

	return r, nil
}

func genProcessServiceCategoryResource(_ ActionID, _ TypeID, att *meta.ResourceAttribute) (*types.Resource, error) {

	r := &types.Resource{
		System:    SystemIDCMDB,
		Attribute: nil,
	}

	if att.BusinessID <= 0 {
		return nil, errors.New("biz id can not be 0")
	}

	// do not related to instance authorize
	r.Type = types.ResourceType(Business)
	r.ID = strconv.FormatInt(att.BusinessID, 10)

	return r, nil
}

func genEventSubscribeResource(act ActionID, typ TypeID, att *meta.ResourceAttribute) (*types.Resource, error) {
	r := &types.Resource{
		System:    SystemIDCMDB,
		Type:      types.ResourceType(typ),
		Attribute: nil,
	}

	if act == CreateEventPushing {
		return r, nil
	}

	r.ID = strconv.FormatInt(att.InstanceID, 10)

	return r, nil
}

//
// // generate model's resource id, works for app model and model management
// // resource type in auth center.
// func genModelResource(act ActionID, resourceType TypeID, attribute *meta.ResourceAttribute) (*types.Resource, error) {
// 	if attribute.InstanceID <= 0 {
// 		return make([]RscTypeAndID, 0), nil
// 	}
// 	id := RscTypeAndID{
// 		ResourceType: resourceType,
// 	}
// 	id.ResourceID = strconv.FormatInt(attribute.InstanceID, 10)
//
// 	return []RscTypeAndID{id}, nil
// }
//
// // generate module resource id.
// func genModelModuleResource(act ActionID, resourceType TypeID,
// 	attribute *meta.ResourceAttribute) (*types.Resource, error) {
// 	return make([]RscTypeAndID, 0), nil
// }
//
// func genModelSetResource(act ActionID, resourceType TypeID,
// 	attribute *meta.ResourceAttribute) (*types.Resource, error) {
// 	return make([]RscTypeAndID, 0), nil
// }
//
// func genMainlineModelResource(act ActionID, resourceType TypeID,
// 	attribute *meta.ResourceAttribute) ([]RscTypeAndID,
// 	error) {
// 	return make([]RscTypeAndID, 0), nil
// }
//
// func genMainlineModelTopologyResource(act ActionID, resourceType TypeID,
// 	attribute *meta.ResourceAttribute) (*types.Resource, error) {
// 	return make([]RscTypeAndID, 0), nil
// }
//
// func genMainlineInstanceTopologyResource(act ActionID, resourceType TypeID,
// 	attribute *meta.ResourceAttribute) (*types.Resource, error) {
//
// 	return make([]RscTypeAndID, 0), nil
// }
//
// func genModelAssociationResource(act ActionID, resourceType TypeID,
// 	attribute *meta.ResourceAttribute) ([]RscTypeAndID,
// 	error) {
//
// 	return make([]RscTypeAndID, 0), nil
// }
//
// func genAssociationTypeResource(act ActionID, resourceType TypeID,
// 	attribute *meta.ResourceAttribute) ([]RscTypeAndID,
// 	error) {
// 	if attribute.InstanceID <= 0 {
// 		return make([]RscTypeAndID, 0), nil
// 	}
// 	id := RscTypeAndID{
// 		ResourceType: resourceType,
// 		ResourceID:   strconv.FormatInt(attribute.InstanceID, 10),
// 	}
//
// 	return []RscTypeAndID{id}, nil
// }
//
// func genModelInstanceAssociationResource(act ActionID, resourceType TypeID,
// 	attribute *meta.ResourceAttribute) (*types.Resource, error) {
//
// 	return nil, nil
// }
//
// func genModelInstanceResource(act ActionID, resourceType TypeID,
// 	attribute *meta.ResourceAttribute) ([]RscTypeAndID,
// 	error) {
// 	if attribute.InstanceID <= 0 {
// 		if len(attribute.Layers) == 0 {
// 			return make([]RscTypeAndID, 0), nil
// 		}
// 		// for create
// 		return []RscTypeAndID{{
// 			ResourceType: SysInstanceModel,
// 			ResourceID:   attribute.Layers[0].InstanceIDEx,
// 		}}, nil
// 	}
//
// 	if len(attribute.Layers) < 1 {
// 		return nil, NotEnoughLayer
// 	}
//
// 	return []RscTypeAndID{
// 		{
// 			ResourceType: SysInstanceModel,
// 			ResourceID:   attribute.Layers[0].InstanceIDEx,
// 		},
// 		{
// 			ResourceType: resourceType,
// 			ResourceID:   strconv.FormatInt(attribute.InstanceID, 10),
// 		},
// 	}, nil
// }
//
// func genModelInstanceTopologyResource(act ActionID, resourceType TypeID,
// 	attribute *meta.ResourceAttribute) (*types.Resource, error) {
//
// 	return make([]RscTypeAndID, 0), nil
// }
//
// func genModelTopologyResource(act ActionID, resourceType TypeID,
// 	attribute *meta.ResourceAttribute) ([]RscTypeAndID,
// 	error) {
//
// 	return make([]RscTypeAndID, 0), nil
// }
//
// func genModelClassificationResource(act ActionID, resourceType TypeID,
// 	attribute *meta.ResourceAttribute) ([]RscTypeAndID,
// 	error) {
// 	if attribute.InstanceID <= 0 {
// 		return make([]RscTypeAndID, 0), nil
// 	}
// 	id := RscTypeAndID{
// 		ResourceType: resourceType,
// 		ResourceID:   strconv.FormatInt(attribute.InstanceID, 10),
// 	}
// 	return []RscTypeAndID{id}, nil
// }
//
// func genModelAttributeGroupResource(act ActionID, resourceType TypeID,
// 	attribute *meta.ResourceAttribute) ([]RscTypeAndID,
// 	error) {
// 	if len(attribute.Layers) < 1 {
// 		return nil, NotEnoughLayer
// 	}
// 	id := RscTypeAndID{
// 		ResourceType: SysModel,
// 	}
// 	id.ResourceID = strconv.FormatInt(attribute.Layers[len(attribute.Layers)-1].InstanceID, 10)
// 	return []RscTypeAndID{id}, nil
// }
//
// func genModelAttributeResource(act ActionID, resourceType TypeID,
// 	attribute *meta.ResourceAttribute) ([]RscTypeAndID,
// 	error) {
// 	if len(attribute.Layers) < 1 {
// 		return nil, NotEnoughLayer
// 	}
// 	id := RscTypeAndID{
// 		ResourceType: SysModel,
// 	}
// 	if attribute.BusinessID > 0 {
// 		id.ResourceType = BizCustomField
// 	}
// 	id.ResourceID = strconv.FormatInt(attribute.Layers[len(attribute.Layers)-1].InstanceID, 10)
// 	return []RscTypeAndID{id}, nil
// }
//
// func genModelUniqueResource(act ActionID, resourceType TypeID,
// 	attribute *meta.ResourceAttribute) (*types.Resource, error) {
// 	if len(attribute.Layers) < 1 {
// 		return nil, NotEnoughLayer
// 	}
// 	id := RscTypeAndID{
// 		ResourceType: SysModel,
// 	}
// 	id.ResourceID = strconv.FormatInt(attribute.Layers[len(attribute.Layers)-1].InstanceID, 10)
// 	return []RscTypeAndID{id}, nil
// }
//
// func genHostUserCustomResource(act ActionID, resourceType TypeID,
// 	attribute *meta.ResourceAttribute) ([]RscTypeAndID,
// 	error) {
//
// 	return make([]RscTypeAndID, 0), nil
// }
//
// func genHostFavoriteResource(act ActionID, resourceType TypeID,
// 	attribute *meta.ResourceAttribute) (*types.Resource, error) {
//
// 	return make([]RscTypeAndID, 0), nil
// }
//
// func genNetDataCollectorResource(act ActionID, resourceType TypeID,
// 	attribute *meta.ResourceAttribute) ([]RscTypeAndID,
// 	error) {
//
// 	return make([]RscTypeAndID, 0), nil
// }
//
// func genHostInstanceResource(act ActionID, resourceType TypeID,
// 	attribute *meta.ResourceAttribute) (*types.Resource, error) {
// 	// translate all parent layers
// 	resourceIDs := make([]RscTypeAndID, 0)
//
// 	if attribute.InstanceID == 0 {
// 		return resourceIDs, nil
// 	}
//
// 	for _, layer := range attribute.Layers {
// 		iamResourceType, err := ConvertResourceType(layer.Type, attribute.BusinessID)
// 		if err != nil {
// 			return nil, fmt.Errorf("convert resource type to iam resource type failed, layer: %+v, err: %+v", layer, err)
// 		}
// 		resourceID := RscTypeAndID{
// 			ResourceType: *iamResourceType,
// 			ResourceID:   strconv.FormatInt(layer.InstanceID, 10),
// 		}
// 		resourceIDs = append(resourceIDs, resourceID)
// 	}
//
// 	// append host resource id to end
// 	hostResourceID := RscTypeAndID{
// 		ResourceType: resourceType,
// 		ResourceID:   strconv.FormatInt(attribute.InstanceID, 10),
// 	}
// 	resourceIDs = append(resourceIDs, hostResourceID)
//
// 	return resourceIDs, nil
// }
//

// 	return []RscTypeAndID{
// 		{
// 			ResourceType: resourceType,
// 			ResourceID:   strconv.FormatInt(attribute.InstanceID, 10),
// 		},
// 	}, nil
// }
//
//
// func genAuditLogResource(act ActionID, resourceType TypeID,
// 	attribute *meta.ResourceAttribute) (*types.Resource, error) {
// 	if len(attribute.InstanceIDEx) == 0 {
// 		return make([]RscTypeAndID, 0), nil
// 	}
// 	instanceID := attribute.InstanceIDEx
// 	return []RscTypeAndID{
// 		{
// 			ResourceType: resourceType,
// 			ResourceID:   instanceID,
// 		},
// 	}, nil
// }
//
// func genPlat(act ActionID, resourceType TypeID, attribute *meta.ResourceAttribute) (*types.Resource,
// 	error) {
// 	if len(attribute.Layers) < 1 {
// 		return nil, NotEnoughLayer
// 	}
//
// 	return []RscTypeAndID{
// 		{
// 			ResourceType: resourceType,
// 			ResourceID:   strconv.FormatInt(attribute.InstanceID, 10),
// 		},
// 	}, nil
// }
//
// func genProcessResource(act ActionID, resourceType TypeID,
// 	attribute *meta.ResourceAttribute) (*types.Resource, error) {
// 	return make([]RscTypeAndID, 0), nil
// }
//
// func genProcessServiceInstanceResource(act ActionID, resourceType TypeID,
// 	attribute *meta.ResourceAttribute) (*types.Resource, error) {
// 	return make([]RscTypeAndID, 0), nil
// }
//
// func genBizTopologyResource(act ActionID, resourceType TypeID,
// 	attribute *meta.ResourceAttribute) (*types.Resource, error) {
// 	return make([]RscTypeAndID, 0), nil
// }
//
// func genProcessTemplateResource(act ActionID, resourceType TypeID,
// 	attribute *meta.ResourceAttribute) ([]RscTypeAndID,
// 	error) {
// 	if attribute.InstanceID == 0 {
// 		return make([]RscTypeAndID, 0), nil
// 	}
// 	return []RscTypeAndID{
// 		{
// 			ResourceType: resourceType,
// 			ResourceID:   strconv.FormatInt(attribute.InstanceID, 10),
// 		},
// 	}, nil
// }
//
//
// func genProcessServiceTemplateResource(act ActionID, resourceType TypeID,
// 	attribute *meta.ResourceAttribute) (*types.Resource, error) {
// 	if attribute.InstanceID == 0 {
// 		return make([]RscTypeAndID, 0), nil
// 	}
// 	return []RscTypeAndID{
// 		{
// 			ResourceType: resourceType,
// 			ResourceID:   strconv.FormatInt(attribute.InstanceID, 10),
// 		},
// 	}, nil
// }
//
// func genSetTemplateResource(act ActionID, resourceType TypeID,
// 	attribute *meta.ResourceAttribute) (*types.Resource, error) {
// 	if attribute.InstanceID == 0 {
// 		return make([]RscTypeAndID, 0), nil
// 	}
// 	return []RscTypeAndID{
// 		{
// 			ResourceType: resourceType,
// 			ResourceID:   strconv.FormatInt(attribute.InstanceID, 10),
// 		},
// 	}, nil
// }
//
// func genOperationStatisticResource(act ActionID, resourceType TypeID,
// 	attribute *meta.ResourceAttribute) ([]RscTypeAndID,
// 	error) {
// 	if attribute.InstanceID == 0 {
// 		return make([]RscTypeAndID, 0), nil
// 	}
// 	return []RscTypeAndID{
// 		{
// 			ResourceType: resourceType,
// 			ResourceID:   strconv.FormatInt(attribute.InstanceID, 10),
// 		},
// 	}, nil
// }
//
// func genHostApplyResource(act ActionID, resourceType TypeID,
// 	attribute *meta.ResourceAttribute) (*types.Resource, error) {
// 	if attribute.InstanceID == 0 {
// 		return make([]RscTypeAndID, 0), nil
// 	}
// 	return []RscTypeAndID{
// 		{
// 			ResourceType: resourceType,
// 			ResourceID:   strconv.FormatInt(attribute.InstanceID, 10),
// 		},
// 	}, nil
// }
//
// func genResourcePoolDirectoryResource(act ActionID, resourceType TypeID,
// 	attribute *meta.ResourceAttribute) (*types.Resource, error) {
// 	if attribute.InstanceID == 0 {
// 		return make([]RscTypeAndID, 0), nil
// 	}
// 	return []RscTypeAndID{
// 		{
// 			ResourceType: resourceType,
// 			ResourceID:   strconv.FormatInt(attribute.InstanceID, 10),
// 		},
// 	}, nil
// }
//
// func genResourceWatch(act ActionID, resourceType TypeID,
// 	attribute *meta.ResourceAttribute) (*types.Resource, error) {
//
// 	return make([]RscTypeAndID, 0), nil
// }
//
// func genCloudAccountResource(act ActionID, resourceType TypeID,
// 	attribute *meta.ResourceAttribute) (*types.Resource, error) {
// 	if attribute.InstanceID == 0 {
// 		return make([]RscTypeAndID, 0), nil
// 	}
// 	return []RscTypeAndID{
// 		{
// 			ResourceType: resourceType,
// 			ResourceID:   strconv.FormatInt(attribute.InstanceID, 10),
// 		},
// 	}, nil
// }
//
// func genCloudResourceTaskResource(act ActionID, resourceType TypeID,
// 	attribute *meta.ResourceAttribute) ([]RscTypeAndID,
// 	error) {
// 	if attribute.InstanceID == 0 {
// 		return make([]RscTypeAndID, 0), nil
// 	}
// 	return []RscTypeAndID{
// 		{
// 			ResourceType: resourceType,
// 			ResourceID:   strconv.FormatInt(attribute.InstanceID, 10),
// 		},
// 	}, nil
// }
//
// func genConfigAdminResource(act ActionID, resourceType TypeID,
// 	attribute *meta.ResourceAttribute) (*types.Resource, error) {
// 	return make([]RscTypeAndID, 0), nil
// }
