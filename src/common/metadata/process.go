/*
 * Tencent is pleased to support the open source community by making 蓝鲸 available.,
 * Copyright (C) 2017,-2018 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the ",License",); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an ",AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */
package metadata

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"configcenter/src/common"
	"configcenter/src/common/mapstr"
	"configcenter/src/common/selector"
	"configcenter/src/common/util"
)

type DeleteCategoryInput struct {
	Metadata Metadata `json:"metadata"`
	ID       int64    `json:"id"`
}

type CreateProcessTemplateBatchInput struct {
	Metadata          Metadata        `json:"metadata"`
	ServiceTemplateID int64           `json:"service_template_id"`
	Processes         []ProcessDetail `json:"processes"`
}

type DeleteProcessTemplateBatchInput struct {
	Metadata         Metadata `json:"metadata"`
	ProcessTemplates []int64  `json:"process_templates"`
}

type ProcessDetail struct {
	Spec *ProcessProperty `json:"spec"`
}

type ListServiceTemplateInput struct {
	Metadata Metadata `json:"metadata"`
	// this field can be empty, it a optional condition.
	ServiceCategoryID int64    `json:"service_category_id,omitempty"`
	Page              BasePage `json:"page"`
}

type ListServiceTemplateWithDetailResult struct {
	ServiceTemplate      ServiceTemplate `json:"service_template"`
	ProcessTemplateCount int64           `json:"process_template_count"`
	ServiceInstanceCount int64           `json:"service_instance_count"`
	ModuleCount          int64           `json:"module_count"`
}

type DeleteServiceTemplatesInput struct {
	Metadata          Metadata `json:"metadata"`
	ServiceTemplateID int64    `json:"service_template_id"`
}

type CreateServiceInstanceForServiceTemplateInput struct {
	Metadata  Metadata                      `json:"metadata"`
	Name      string                        `json:"name"`
	ModuleID  int64                         `json:"bk_module_id"`
	Instances []CreateServiceInstanceDetail `json:"instances"`
}

type CreateRawProcessInstanceInput struct {
	Metadata          Metadata                `json:"metadata"`
	ServiceInstanceID int64                   `json:"service_instance_Id"`
	Processes         []ProcessInstanceDetail `json:"processes"`
}

type UpdateRawProcessInstanceInput struct {
	Metadata  Metadata  `json:"metadata"`
	Processes []Process `json:"processes"`
}

type DeleteProcessInstanceInServiceInstanceInput struct {
	Metadata           Metadata `json:"metadata"`
	ProcessInstanceIDs []int64  `json:"process_instance_ids"`
}

type GetServiceInstanceInModuleInput struct {
	Metadata  Metadata           `json:"metadata"`
	ModuleID  int64              `json:"bk_module_id"`
	Page      BasePage           `json:"page"`
	SearchKey *string            `json:"search_key,omitempty"`
	Selectors selector.Selectors `json:"selectors"`
}

type ListServiceInstanceDetailRequest struct {
	Metadata           Metadata           `json:"metadata"`
	SetID              int64              `json:"bk_set_id"`
	ModuleID           int64              `json:"bk_module_id"`
	HostID             int64              `json:"bk_host_id"`
	ServiceInstanceIDs []int64            `json:"service_instance_ids"`
	Page               BasePage           `json:"page,omitempty"`
	Selectors          selector.Selectors `json:"selectors,omitempty"`
}

type DiffModuleWithTemplateOption struct {
	Metadata Metadata `json:"metadata"`
	ModuleID int64    `json:"bk_module_id"`
}

type DeleteServiceInstanceOption struct {
	Metadata           Metadata `json:"metadata"`
	ServiceInstanceIDs []int64  `json:"service_instance_ids" field:"service_instance_ids" bson:"service_instance_ids"`
}

type FindServiceAndProcessInstanceOption struct {
	Metadata          Metadata `json:"metadata" field:"metadata" bson:"metadata"`
	ModuleID          int64    `json:"bk_module_id" field:"bk_module_id" bson:"bk_module_id"`
	ServiceTemplateID int64    `json:"service_template_id" field:"service_template_id" bson:"service_template_id"`
}

// to describe the differences between service instance and it's service template's
// process template's attribute.
type ServiceProcessInstanceDifference struct {
	ServiceInstanceID   int64             `json:"service_instance_id" field:"service_instance_id" bson:"service_instance_id"`
	ServiceInstanceName string            `json:"service_instance_name" field:"service_instance_name" bson:"service_instance_name"`
	HostID              int64             `json:"bk_host_id" field:"bk_host_id" bson:"bk_host_id"`
	Differences         *DifferenceDetail `json:"differences" field:"differences" bson:"differences"`
}

type DifferenceDetail struct {
	Unchanged []ProcessDifferenceDetail `json:"unchanged"`
	Changed   []ProcessDifferenceDetail `json:"changed"`
	Added     []ProcessDifferenceDetail `json:"added"`
	Removed   []ProcessDifferenceDetail `json:"removed"`
}

type ProcessDifferenceDetail struct {
	ProcessTemplateID int64                     `json:"process_template_id"`
	ProcessInstance   Process                   `json:"process_instance"`
	ChangedAttributes []ProcessChangedAttribute `json:"changed_attributes"`
}

type ProcessChangedAttribute struct {
	ID                    int64       `json:"id"`
	PropertyID            string      `json:"property_id"`
	PropertyName          string      `json:"property_name"`
	PropertyValue         interface{} `json:"property_value"`
	TemplatePropertyValue interface{} `json:"template_property_value"`
}

// ModuleDiffWithTemplateDetail 模块与服务模板间的差异
type ModuleDiffWithTemplateDetail struct {
	Unchanged         []ServiceInstanceDifference `json:"unchanged"`
	Changed           []ServiceInstanceDifference `json:"changed"`
	Added             []ServiceInstanceDifference `json:"added"`
	Removed           []ServiceInstanceDifference `json:"removed"`
	ChangedAttributes []ModuleChangedAttribute    `json:"changed_attributes"`
	HasDifference     bool                        `json:"has_difference"`
}

type ModuleChangedAttribute struct {
	ID                    int64       `json:"id"`
	PropertyID            string      `json:"property_id"`
	PropertyName          string      `json:"property_name"`
	PropertyValue         interface{} `json:"property_value"`
	TemplatePropertyValue interface{} `json:"template_property_value"`
}

// ServiceInstanceDifference 服务实例内的进程信息与进程模板ID不一致的服务实例列表
type ServiceInstanceDifference struct {
	ProcessTemplateID    int64                      `json:"process_template_id"`
	ProcessTemplateName  string                     `json:"process_template_name"`
	ServiceInstanceCount int                        `json:"service_instance_count"`
	ServiceInstances     []ServiceDifferenceDetails `json:"service_instances"`
}

// ServiceDifferenceDetails 服务实例与模板差异信息
type ServiceDifferenceDetails struct {
	ServiceInstance   ServiceInstance           `json:"service_instance"`
	Process           *Process                  `json:"process"`
	ChangedAttributes []ProcessChangedAttribute `json:"changed_attributes,omitempty"`
}

type CreateServiceInstanceDetail struct {
	HostID int64 `json:"bk_host_id"`
	// Processes parameter usable only when create instance with raw
	Processes []ProcessInstanceDetail `json:"processes"`
}

type ProcessInstanceDetail struct {
	// ProcessTemplateID int64   `json:"process_template_id"`
	ProcessInfo Process `json:"process_info"`
}

type ListProcessTemplateWithServiceTemplateInput struct {
	Metadata            Metadata `json:"metadata"`
	ProcessTemplatesIDs []int64  `json:"process_template_ids"`
	ServiceTemplateID   int64    `json:"service_template_id"`
	Page                BasePage `json:"page" field:"page" bson:"page"`
}

type SyncServiceInstanceByTemplateOption struct {
	Metadata Metadata `json:"metadata"`
	ModuleID int64    `json:"bk_module_id"`
}

type ListServiceInstancesWithHostInput struct {
	Metadata  Metadata           `json:"metadata"`
	HostID    int64              `json:"bk_host_id"`
	SearchKey *string            `json:"search_key,omitempty"`
	Selectors selector.Selectors `json:"selectors,omitempty"`
	Page      BasePage           `json:"page,omitempty"`
}

type ListProcessInstancesOption struct {
	Metadata          Metadata `json:"metadata"`
	ServiceInstanceID int64    `json:"service_instance_id"`
}

type RemoveTemplateBindingOnModuleOption struct {
	Metadata Metadata `json:"metadata"`
	ModuleID int64    `json:"bk_module_id"`
}

type UpdateProcessTemplateInput struct {
	Metadata          Metadata         `json:"metadata"`
	ProcessTemplateID int64            `json:"process_template_id"`
	ProcessProperty   *ProcessProperty `json:"process_property"`
}

type SocketBindType string

const (
	BindLocalHost SocketBindType = "1"
	BindAll       SocketBindType = "2"
	BindInnerIP   SocketBindType = "3"
	BindOtterIP   SocketBindType = "4"
)

func (p *SocketBindType) IP() string {
	// TODO: support BindInnerIP and BindOtterIP
	if p == nil {
		return ""
	}
	switch *p {
	case BindLocalHost:
		return "127.0.0.1"
	case BindAll:
		return "0.0.0.0"
	default:
		return ""
	}
}

func (p *SocketBindType) String() string {
	// TODO: how to support internationalization?
	if p == nil {
		return ""
	}
	switch *p {
	case BindLocalHost:
		return "127.0.0.1"
	case BindAll:
		return "0.0.0.0"
	case BindInnerIP:
		return "第一内网IP"
	case BindOtterIP:
		return "第一外网IP"
	default:
		return ""
	}
}

func (p SocketBindType) Validate() error {
	// validValues := []SocketBindType{BindLocalHost, BindAll, BindInnerIP, BindOtterIP}
	validValues := []SocketBindType{BindLocalHost, BindAll}
	if util.InArray(p, validValues) == false {
		return fmt.Errorf("invalid socket bind type, value: %s, available values: %+v", p, validValues)
	}
	return nil
}

type ProtocolType string

const (
	ProtocolTypeTCP ProtocolType = "1"
	ProtocolTypeUDP ProtocolType = "2"
)

func (p ProtocolType) String() string {
	switch p {
	case ProtocolTypeTCP:
		return "TCP"
	case ProtocolTypeUDP:
		return "UDP"
	default:
		return ""
	}
}

func (p ProtocolType) Validate() error {
	validValues := []ProtocolType{ProtocolTypeTCP, ProtocolTypeUDP}
	if util.InArray(p, validValues) == false {
		return fmt.Errorf("invalid protocol type, value: %s, available values: %+v", p, validValues)
	}
	return nil
}

type Process struct {
	Metadata        Metadata      `field:"metadata" json:"metadata" bson:"metadata" structs:"metadata"`
	ProcNum         *int64        `field:"proc_num" json:"proc_num,omitempty" bson:"proc_num" structs:"proc_num"`
	StopCmd         *string       `field:"stop_cmd" json:"stop_cmd,omitempty" bson:"stop_cmd" structs:"stop_cmd"`
	RestartCmd      *string       `field:"restart_cmd" json:"restart_cmd,omitempty" bson:"restart_cmd" structs:"restart_cmd"`
	ForceStopCmd    *string       `field:"face_stop_cmd" json:"face_stop_cmd,omitempty" bson:"face_stop_cmd" structs:"face_stop_cmd"`
	ProcessID       int64         `field:"bk_process_id" json:"bk_process_id,omitempty" bson:"bk_process_id" structs:"bk_process_id"`
	FuncName        *string       `field:"bk_func_name" json:"bk_func_name,omitempty" bson:"bk_func_name" structs:"bk_func_name"`
	WorkPath        *string       `field:"work_path" json:"work_path,omitempty" bson:"work_path" structs:"work_path"`
	BindIP          *string       `field:"bind_ip" json:"bind_ip,omitempty" bson:"bind_ip" structs:"bind_ip"`
	Priority        *int64        `field:"priority" json:"priority,omitempty" bson:"priority" structs:"priority"`
	ReloadCmd       *string       `field:"reload_cmd" json:"reload_cmd,omitempty" bson:"reload_cmd" structs:"reload_cmd"`
	ProcessName     *string       `field:"bk_process_name" json:"bk_process_name,omitempty" bson:"bk_process_name" structs:"bk_process_name"`
	Port            *string       `field:"port" json:"port,omitempty" bson:"port" structs:"port"`
	PidFile         *string       `field:"pid_file" json:"pid_file,omitempty" bson:"pid_file" structs:"pid_file"`
	AutoStart       *bool         `field:"auto_start" json:"auto_start,omitempty" bson:"auto_start" structs:"auto_start"`
	AutoTimeGap     *int64        `field:"auto_time_gap" json:"auto_time_gap,omitempty" bson:"auto_time_gap" structs:"auto_time_gap"`
	LastTime        time.Time     `field:"last_time" json:"last_time,omitempty" bson:"last_time" structs:"last_time"`
	CreateTime      time.Time     `field:"create_time" json:"create_time,omitempty" bson:"create_time" structs:"create_time"`
	BusinessID      int64         `field:"bk_biz_id" json:"bk_biz_id,omitempty" bson:"bk_biz_id" structs:"bk_biz_id"`
	StartCmd        *string       `field:"start_cmd" json:"start_cmd,omitempty" bson:"start_cmd" structs:"start_cmd"`
	FuncID          *string       `field:"bk_func_id" json:"bk_func_id,omitempty" bson:"bk_func_id" structs:"bk_func_id"`
	User            *string       `field:"user" json:"user,omitempty" bson:"user" structs:"user"`
	TimeoutSeconds  *int64        `field:"timeout" json:"timeout,omitempty" bson:"timeout" structs:"timeout"`
	Protocol        *ProtocolType `field:"protocol" json:"protocol,omitempty" bson:"protocol" structs:"protocol"`
	Description     *string       `field:"description" json:"description,omitempty" bson:"description" structs:"description"`
	SupplierAccount string        `field:"bk_supplier_account" json:"bk_supplier_account,omitempty" bson:"bk_supplier_account" structs:"bk_supplier_account"`
	StartParamRegex *string       `field:"bk_start_param_regex" json:"bk_start_param_regex,omitempty" bson:"bk_start_param_regex,omitempty" structs:"bk_start_param_regex"`
}

type ServiceCategory struct {
	Metadata Metadata `field:"metadata" json:"metadata" bson:"metadata"`

	ID   int64  `field:"id" json:"id,omitempty" bson:"id"`
	Name string `field:"name" json:"name,omitempty" bson:"name"`

	RootID          int64  `field:"bk_root_id" json:"bk_root_id,omitempty" bson:"bk_root_id"`
	ParentID        int64  `field:"bk_parent_id" json:"bk_parent_id,omitempty" bson:"bk_parent_id"`
	SupplierAccount string `field:"bk_supplier_account" json:"bk_supplier_account,omitempty" bson:"bk_supplier_account"`

	// IsBuiltIn indicates internal system service category, which shouldn't be modified.
	IsBuiltIn bool `field:"is_built_in" json:"is_built_in" bson:"is_built_in"`
}

func (sc *ServiceCategory) Validate() (field string, err error) {
	if len(sc.Name) == 0 {
		return "name", errors.New("name can't be empty")
	}

	if len(sc.Name) > common.NameFieldMaxLength {
		return "name", fmt.Errorf("name too long, input: %d > max: %d", len(sc.Name), common.NameFieldMaxLength)
	}
	return "", nil
}

type ServiceCategoryWithStatistics struct {
	ServiceCategory ServiceCategory `field:"category" json:"category" bson:"category"`
	UsageAmount     int64           `field:"usage_amount" json:"usage_amount" bson:"usage_amount"`
}

type ServiceTemplateDetail struct {
	Template             ServiceTemplate `field:"template" json:"template" bson:"template"`
	ServiceInstanceCount int64           `field:"service_instance_count" json:"service_instance_count" bson:"service_instance_count"`
	ProcessInstanceCount int64           `field:"process_instance_count" json:"process_instance_count" bson:"process_instance_count"`
}

type ServiceTemplate struct {
	Metadata Metadata `field:"metadata" json:"metadata" bson:"metadata"`

	ID int64 `field:"id" json:"id,omitempty" bson:"id"`
	// name of this service, can not be empty
	Name string `field:"name" json:"name,omitempty" bson:"name"`

	// the class of this service, each field means a class label.
	// now, the class must have two labels.
	ServiceCategoryID int64 `field:"service_category_id" json:"service_category_id,omitempty" bson:"service_category_id"`

	Creator         string    `field:"creator" json:"creator,omitempty" bson:"creator"`
	Modifier        string    `field:"modifier" json:"modifier,omitempty" bson:"modifier"`
	CreateTime      time.Time `field:"create_time" json:"create_time,omitempty" bson:"create_time"`
	LastTime        time.Time `field:"last_time" json:"last_time,omitempty" bson:"last_time"`
	SupplierAccount string    `field:"bk_supplier_account" json:"bk_supplier_account,omitempty" bson:"bk_supplier_account"`
}

func (st *ServiceTemplate) Validate() (field string, err error) {
	if len(st.Name) == 0 {
		return "name", errors.New("name can't be empty")
	}

	if len(st.Name) > common.NameFieldMaxLength {
		return "name", fmt.Errorf("name too long, input: %d > max: %d", len(st.Name), common.NameFieldMaxLength)
	}
	return "", nil
}

// this works for the process instance which is used for a template.
type ProcessTemplate struct {
	ID          int64    `field:"id" json:"id,omitempty" bson:"id"`
	ProcessName string   `field:"bk_process_name" json:"bk_process_name" bson:"bk_process_name"`
	Metadata    Metadata `field:"metadata" json:"metadata" bson:"metadata"`
	// the service template's, which this process template belongs to.
	ServiceTemplateID int64 `field:"service_template_id" json:"service_template_id" bson:"service_template_id"`

	// stores a process instance's data includes all the process's
	// properties's value.
	Property *ProcessProperty `field:"property" json:"property,omitempty" bson:"property"`

	Creator         string    `field:"creator" json:"creator,omitempty" bson:"creator"`
	Modifier        string    `field:"modifier" json:"modifier,omitempty" bson:"modifier"`
	CreateTime      time.Time `field:"create_time" json:"create_time,omitempty" bson:"create_time"`
	LastTime        time.Time `field:"last_time" json:"last_time,omitempty" bson:"last_time"`
	SupplierAccount string    `field:"bk_supplier_account" json:"bk_supplier_account,omitempty" bson:"bk_supplier_account"`
}

func (pt *ProcessTemplate) Validate() (field string, err error) {
	if pt.Property == nil {
		return "property", errors.New("property field shouldn't be nil")
	}
	field, err = pt.Property.Validate()
	if err != nil {
		return field, err
	}
	return "", nil
}

func IsAsDefaultValue(asDefaultValue *bool) bool {
	if asDefaultValue != nil {
		return *asDefaultValue
	}
	return false
}

func (pt *ProcessTemplate) NewProcess(bizID int64, supplierAccount string) *Process {
	now := time.Now()
	processInstance := &Process{
		LastTime:        now,
		CreateTime:      now,
		BusinessID:      bizID,
		Metadata:        NewMetaDataFromBusinessID(strconv.FormatInt(bizID, 10)),
		SupplierAccount: supplierAccount,
	}

	property := pt.Property
	processInstance.ProcessName = property.ProcessName.Value
	processInstance.ProcNum = property.ProcNum.Value
	processInstance.StopCmd = property.StopCmd.Value
	processInstance.RestartCmd = property.RestartCmd.Value
	processInstance.ForceStopCmd = property.ForceStopCmd.Value
	processInstance.FuncName = property.FuncName.Value
	processInstance.WorkPath = property.WorkPath.Value
	processInstance.BindIP = new(string)
	*processInstance.BindIP = property.BindIP.Value.IP()
	processInstance.Priority = property.Priority.Value
	processInstance.ReloadCmd = property.ReloadCmd.Value
	processInstance.Port = property.Port.Value
	processInstance.PidFile = property.PidFile.Value
	processInstance.AutoStart = property.AutoStart.Value
	processInstance.AutoTimeGap = property.AutoTimeGapSeconds.Value
	processInstance.StartCmd = property.StartCmd.Value
	processInstance.FuncID = property.FuncID.Value
	processInstance.User = property.User.Value
	processInstance.TimeoutSeconds = property.TimeoutSeconds.Value
	processInstance.Protocol = property.Protocol.Value
	processInstance.Description = property.Description.Value
	processInstance.StartParamRegex = property.StartParamRegex.Value
	return processInstance
}

// ExtractChangeInfo get changes that will be applied to process instance
func (pt *ProcessTemplate) ExtractChangeInfo(i *Process) (mapstr.MapStr, bool) {
	t := pt.Property
	var changed bool
	if t == nil || i == nil {
		return nil, false
	}

	process := make(mapstr.MapStr)
	if IsAsDefaultValue(t.ProcNum.AsDefaultValue) {
		if t.ProcNum.Value == nil && i.ProcNum != nil {
			process["proc_num"] = nil
			changed = true
		} else if t.ProcNum.Value != nil && i.ProcNum == nil {
			process["proc_num"] = *t.ProcNum.Value
			changed = true
		} else if t.ProcNum.Value != nil && i.ProcNum != nil && *t.ProcNum.Value != *i.ProcNum {
			process["proc_num"] = *t.ProcNum.Value
			changed = true
		}
	}

	if IsAsDefaultValue(t.StopCmd.AsDefaultValue) {
		if t.StopCmd.Value == nil && i.StopCmd != nil {
			process["stop_cmd"] = nil
			changed = true
		} else if t.StopCmd.Value != nil && i.StopCmd == nil {
			process["stop_cmd"] = *t.StopCmd.Value
			changed = true
		} else if t.StopCmd.Value != nil && i.StopCmd != nil && *t.StopCmd.Value != *i.StopCmd {
			process["stop_cmd"] = *t.StopCmd.Value
			changed = true
		}
	}

	if IsAsDefaultValue(t.RestartCmd.AsDefaultValue) {
		if t.RestartCmd.Value == nil && i.RestartCmd != nil {
			process["restart_cmd"] = nil
			changed = true
		} else if t.RestartCmd.Value != nil && i.RestartCmd == nil {
			process["restart_cmd"] = *t.RestartCmd.Value
			changed = true
		} else if t.RestartCmd.Value != nil && i.RestartCmd != nil && *t.RestartCmd.Value != *i.RestartCmd {
			process["restart_cmd"] = *t.RestartCmd.Value
			changed = true
		}
	}

	if IsAsDefaultValue(t.ForceStopCmd.AsDefaultValue) {
		if t.ForceStopCmd.Value == nil && i.ForceStopCmd != nil {
			process["face_stop_cmd"] = nil
			changed = true
		} else if t.ForceStopCmd.Value != nil && i.ForceStopCmd == nil {
			process["face_stop_cmd"] = *t.ForceStopCmd.Value
			changed = true
		} else if t.ForceStopCmd.Value != nil && i.ForceStopCmd != nil && *t.ForceStopCmd.Value != *i.ForceStopCmd {
			process["face_stop_cmd"] = *t.ForceStopCmd.Value
			changed = true
		}
	}

	if IsAsDefaultValue(t.FuncName.AsDefaultValue) {
		if t.FuncName.Value == nil && i.FuncName != nil {
			process["bk_func_name"] = nil
			changed = true
		} else if t.FuncName.Value != nil && i.FuncName == nil {
			process["bk_func_name"] = *t.FuncName.Value
			changed = true
		} else if t.FuncName.Value != nil && i.FuncName != nil && *t.FuncName.Value != *i.FuncName {
			process["bk_func_name"] = *t.FuncName.Value
			changed = true
		}
	}

	if IsAsDefaultValue(t.WorkPath.AsDefaultValue) {
		if t.WorkPath.Value == nil && i.WorkPath != nil {
			process["work_path"] = nil
			changed = true
		} else if t.WorkPath.Value != nil && i.WorkPath == nil {
			process["work_path"] = *t.WorkPath.Value
			changed = true
		} else if t.WorkPath.Value != nil && i.WorkPath != nil && *t.WorkPath.Value != *i.WorkPath {
			process["work_path"] = *t.WorkPath.Value
			changed = true
		}
	}

	if IsAsDefaultValue(t.BindIP.AsDefaultValue) {
		if t.BindIP.Value == nil && i.BindIP != nil {
			process["bind_ip"] = nil
			changed = true
		} else if t.BindIP.Value != nil && i.BindIP == nil {
			process["bind_ip"] = *t.BindIP.Value
			changed = true
		} else if t.BindIP.Value != nil && i.BindIP != nil && t.BindIP.Value.IP() != *i.BindIP {
			process["bind_ip"] = t.BindIP.Value.IP()
			changed = true
		}
	}

	if IsAsDefaultValue(t.Priority.AsDefaultValue) {
		if t.Priority.Value != nil && i.Priority == nil {
			process["priority"] = *t.Priority.Value
			changed = true
		} else if t.Priority.Value == nil && i.Priority != nil {
			process["priority"] = nil
			changed = true
		} else if t.Priority.Value != nil && i.Priority != nil && *t.Priority.Value != *i.Priority {
			process["priority"] = *t.Priority.Value
			changed = true
		}
	}

	if IsAsDefaultValue(t.ReloadCmd.AsDefaultValue) {
		if t.ReloadCmd.Value == nil && i.ReloadCmd != nil {
			process["reload_cmd"] = nil
			changed = true
		} else if t.ReloadCmd.Value != nil && i.ReloadCmd == nil {
			process["reload_cmd"] = *t.ReloadCmd.Value
			changed = true
		} else if t.ReloadCmd.Value != nil && i.ReloadCmd != nil && *t.ReloadCmd.Value != *i.ReloadCmd {
			process["reload_cmd"] = *t.ReloadCmd.Value
			changed = true
		}
	}

	if IsAsDefaultValue(t.ProcessName.AsDefaultValue) {
		if t.ProcessName.Value == nil && i.ProcessName != nil {
			process["bk_process_name"] = nil
			changed = true
		} else if t.ProcessName.Value != nil && i.ProcessName == nil {
			process["bk_process_name"] = *t.ProcessName.Value
			changed = true
		} else if t.ProcessName.Value != nil && i.ProcessName != nil && *t.ProcessName.Value != *i.ProcessName {
			process["bk_process_name"] = *t.ProcessName.Value
			changed = true
		}
	}

	if IsAsDefaultValue(t.Port.AsDefaultValue) {
		if t.Port.Value == nil && i.Port != nil {
			process["port"] = nil
			changed = true
		} else if t.Port.Value != nil && i.Port == nil {
			process["port"] = *t.Port.Value
			changed = true
		} else if t.Port.Value != nil && i.Port != nil && *t.Port.Value != *i.Port {
			process["port"] = *t.Port.Value
			changed = true
		}
	}

	if IsAsDefaultValue(t.PidFile.AsDefaultValue) {
		if t.PidFile.Value == nil && i.PidFile != nil {
			process["pid_file"] = nil
			changed = true
		} else if t.PidFile.Value != nil && i.PidFile == nil {
			process["pid_file"] = *t.PidFile.Value
			changed = true
		} else if t.PidFile.Value != nil && i.PidFile != nil && *t.PidFile.Value != *i.PidFile {
			process["pid_file"] = *t.PidFile.Value
			changed = true
		}
	}

	if IsAsDefaultValue(t.AutoStart.AsDefaultValue) {
		if t.AutoStart.Value == nil && i.AutoStart != nil {
			process["auto_start"] = nil
			changed = true
		} else if t.AutoStart.Value != nil && i.AutoStart == nil {
			process["auto_start"] = *t.AutoStart.Value
			changed = true
		} else if t.AutoStart.Value != nil && i.AutoStart != nil && *t.AutoStart.Value != *i.AutoStart {
			process["auto_start"] = *t.AutoStart.Value
			changed = true
		}
	}

	if IsAsDefaultValue(t.AutoTimeGapSeconds.AsDefaultValue) {
		if t.AutoTimeGapSeconds.Value != nil && i.AutoTimeGap == nil {
			process["auto_time_gap"] = *t.AutoTimeGapSeconds.Value
			changed = true
		} else if t.AutoTimeGapSeconds.Value == nil && i.AutoTimeGap != nil {
			process["auto_time_gap"] = *t.AutoTimeGapSeconds.Value
			changed = true
		} else if t.AutoTimeGapSeconds.Value != nil && i.AutoTimeGap != nil && *t.AutoTimeGapSeconds.Value != *i.AutoTimeGap {
			process["auto_time_gap"] = *t.AutoTimeGapSeconds.Value
			changed = true
		}
	}

	if IsAsDefaultValue(t.StartCmd.AsDefaultValue) {
		if t.StartCmd.Value == nil && i.StartCmd != nil {
			process["start_cmd"] = nil
			changed = true
		} else if t.StartCmd.Value != nil && i.StartCmd == nil {
			process["start_cmd"] = *t.StartCmd.Value
			changed = true
		} else if t.StartCmd.Value != nil && i.StartCmd != nil && *t.StartCmd.Value != *i.StartCmd {
			process["start_cmd"] = *t.StartCmd.Value
			changed = true
		}
	}

	if IsAsDefaultValue(t.FuncID.AsDefaultValue) {
		if t.FuncID.Value == nil && i.FuncID != nil {
			process["bk_func_id"] = nil
			changed = true
		} else if t.FuncID.Value != nil && i.FuncID == nil {
			process["bk_func_id"] = *t.FuncID.Value
			changed = true
		} else if t.FuncID.Value != nil && i.FuncID != nil && *t.FuncID.Value != *i.FuncID {
			process["bk_func_id"] = *t.FuncID.Value
			changed = true
		}
	}

	if IsAsDefaultValue(t.User.AsDefaultValue) {
		if t.User.Value == nil && i.User != nil {
			process["user"] = nil
			changed = true
		} else if t.User.Value != nil && i.User == nil {
			process["user"] = *t.User.Value
			changed = true
		} else if t.User.Value != nil && i.User != nil && *t.User.Value != *i.User {
			process["user"] = *t.User.Value
			changed = true
		}
	}

	if IsAsDefaultValue(t.TimeoutSeconds.AsDefaultValue) {
		if t.TimeoutSeconds.Value != nil && i.TimeoutSeconds == nil {
			process["timeout"] = *t.TimeoutSeconds.Value
			changed = true
		} else if t.TimeoutSeconds.Value == nil && i.TimeoutSeconds != nil {
			process["timeout"] = nil
			changed = true
		} else if t.TimeoutSeconds.Value != nil && i.TimeoutSeconds != nil && *t.TimeoutSeconds.Value != *i.TimeoutSeconds {
			process["timeout"] = *t.TimeoutSeconds.Value
			changed = true
		}
	}

	if IsAsDefaultValue(t.Protocol.AsDefaultValue) {
		if t.Protocol.Value == nil && i.Protocol != nil {
			process["protocol"] = nil
			changed = true
		} else if t.Protocol.Value != nil && i.Protocol == nil {
			process["protocol"] = *t.Protocol.Value
			changed = true
		} else if t.Protocol.Value != nil && i.Protocol != nil && *t.Protocol.Value != *i.Protocol {
			process["protocol"] = *t.Protocol.Value
			changed = true
		}
	}

	if IsAsDefaultValue(t.Description.AsDefaultValue) {
		if t.Description.Value == nil && i.Description != nil {
			process["description"] = nil
			changed = true
		} else if t.Description.Value != nil && i.Description == nil {
			process["description"] = *t.Description.Value
			changed = true
		} else if t.Description.Value != nil && i.Description != nil && *t.Description.Value != *i.Description {
			process["description"] = *t.Description.Value
			changed = true
		}
	}

	if IsAsDefaultValue(t.StartParamRegex.AsDefaultValue) {
		if t.StartParamRegex.Value == nil && i.StartParamRegex != nil {
			process["bk_start_param_regex"] = nil
			changed = true
		} else if t.StartParamRegex.Value != nil && i.StartParamRegex == nil {
			process["bk_start_param_regex"] = *t.StartParamRegex.Value
			changed = true
		} else if t.StartParamRegex.Value != nil && i.StartParamRegex != nil && *t.StartParamRegex.Value != *i.StartParamRegex {
			process["bk_start_param_regex"] = *t.StartParamRegex.Value
			changed = true
		}
	}

	return process, changed
}

// InstanceUpdate is used for update instance's value
func (pt *ProcessTemplate) ExtractInstanceUpdateData(input *Process) map[string]interface{} {
	data := make(map[string]interface{})
	property := pt.Property
	if IsAsDefaultValue(property.FuncName.AsDefaultValue) == false {
		if input.FuncName != nil {
			data["bk_func_name"] = *input.FuncName
		}
	}
	if IsAsDefaultValue(property.ProcessName.AsDefaultValue) == false {
		if input.ProcessName != nil {
			data["bk_process_name"] = *input.ProcessName
		}
	}
	if IsAsDefaultValue(property.FuncID.AsDefaultValue) == false {
		if input.FuncID != nil {
			data["bk_func_id"] = *input.FuncID
		}
	}
	if IsAsDefaultValue(property.StartParamRegex.AsDefaultValue) == false {
		if input.StartParamRegex != nil {
			data["bk_start_param_regex"] = *input.StartParamRegex
		}
	}
	if IsAsDefaultValue(property.AutoTimeGapSeconds.AsDefaultValue) == false {
		if input.AutoTimeGap != nil {
			data["auto_time_gap"] = *input.AutoTimeGap
		}
	}
	if IsAsDefaultValue(property.User.AsDefaultValue) == false {
		if input.User != nil {
			data["user"] = *input.User
		}
	}
	if IsAsDefaultValue(property.StopCmd.AsDefaultValue) == false {
		if input.StopCmd != nil {
			data["stop_cmd"] = *input.StopCmd
		}
	}
	if IsAsDefaultValue(property.ProcNum.AsDefaultValue) == false {
		if input.ProcNum != nil {
			data["proc_num"] = *input.ProcNum
		}
	}
	if IsAsDefaultValue(property.Port.AsDefaultValue) == false {
		if input.Port != nil {
			data["port"] = *input.Port
		}
	}
	if IsAsDefaultValue(property.Description.AsDefaultValue) == false {
		if input.Description != nil {
			data["description"] = *input.Description
		}
	}
	if IsAsDefaultValue(property.Protocol.AsDefaultValue) == false {
		if input.Protocol != nil {
			data["protocol"] = *input.Protocol
		}
	}
	if IsAsDefaultValue(property.TimeoutSeconds.AsDefaultValue) == false {
		if input.TimeoutSeconds != nil {
			data["timeout"] = *input.TimeoutSeconds
		}
	}
	if IsAsDefaultValue(property.AutoStart.AsDefaultValue) == false {
		if input.AutoStart != nil {
			data["auto_start"] = *input.AutoStart
		}
	}
	if IsAsDefaultValue(property.PidFile.AsDefaultValue) == false {
		if input.PidFile != nil {
			data["pid_file"] = *input.PidFile
		}
	}
	if IsAsDefaultValue(property.ReloadCmd.AsDefaultValue) == false {
		if input.ReloadCmd != nil {
			data["reload_cmd"] = *input.ReloadCmd
		}
	}
	if IsAsDefaultValue(property.RestartCmd.AsDefaultValue) == false {
		if input.RestartCmd != nil {
			data["restart_cmd"] = *input.RestartCmd
		}
	}
	if IsAsDefaultValue(property.ForceStopCmd.AsDefaultValue) == false {
		if input.ForceStopCmd != nil {
			data["face_stop_cmd"] = *input.ForceStopCmd
		}
	}
	if IsAsDefaultValue(property.WorkPath.AsDefaultValue) == false {
		if input.WorkPath != nil {
			data["work_path"] = *input.WorkPath
		}
	}
	if IsAsDefaultValue(property.BindIP.AsDefaultValue) == false {
		if input.BindIP != nil {
			data["bind_ip"] = *input.BindIP
		}
	}
	if IsAsDefaultValue(property.Priority.AsDefaultValue) == false {
		if input.Priority != nil {
			data["priority"] = *input.Priority
		}
	}
	if IsAsDefaultValue(property.StartCmd.AsDefaultValue) == false {
		if input.StartCmd != nil {
			data["start_cmd"] = *input.StartCmd
		}
	}
	return data
}

type ProcessProperty struct {
	ProcNum            PropertyInt64       `field:"proc_num" json:"proc_num,omitempty" bson:"proc_num,omitempty" validate:"max=10000,min=1"`
	StopCmd            PropertyString      `field:"stop_cmd" json:"stop_cmd,omitempty" bson:"stop_cmd,omitempty"`
	RestartCmd         PropertyString      `field:"restart_cmd" json:"restart_cmd,omitempty" bson:"restart_cmd,omitempty"`
	ForceStopCmd       PropertyString      `field:"face_stop_cmd" json:"face_stop_cmd,omitempty" bson:"face_stop_cmd,omitempty"`
	FuncName           PropertyString      `field:"bk_func_name" json:"bk_func_name,omitempty" bson:"bk_func_name,omitempty" validate:"required"`
	WorkPath           PropertyString      `field:"work_path" json:"work_path,omitempty" bson:"work_path,omitempty"`
	BindIP             PropertyBindIP      `field:"bind_ip" json:"bind_ip,omitempty" bson:"bind_ip,omitempty"`
	Priority           PropertyInt64       `field:"priority" json:"priority,omitempty" bson:"priority,omitempty" validate:"max=10000,min=1"`
	ReloadCmd          PropertyString      `field:"reload_cmd" json:"reload_cmd,omitempty" bson:"reload_cmd,omitempty"`
	ProcessName        PropertyString      `field:"bk_process_name" json:"bk_process_name,omitempty" bson:"bk_process_name,omitempty" validate:"required"`
	Port               PropertyPort        `field:"port" json:"port,omitempty" bson:"port,omitempty"`
	PidFile            PropertyString      `field:"pid_file" json:"pid_file,omitempty" bson:"pid_file,omitempty"`
	AutoStart          PropertyBool        `field:"auto_start" json:"auto_start,omitempty" bson:"auto_start,omitempty"`
	AutoTimeGapSeconds PropertyInt64       `field:"auto_time_gap" json:"auto_time_gap,omitempty" bson:"auto_time_gap,omitempty" validate:"max=10000,min=1"`
	StartCmd           PropertyString      `field:"start_cmd" json:"start_cmd,omitempty" bson:"start_cmd,omitempty"`
	FuncID             PropertyInt64String `field:"bk_func_id" json:"bk_func_id,omitempty" bson:"bk_func_id,omitempty"`
	User               PropertyString      `field:"user" json:"user,omitempty" bson:"user,omitempty"`
	TimeoutSeconds     PropertyInt64       `field:"timeout" json:"timeout,omitempty" bson:"timeout,omitempty" validate:"max=10000,min=1"`
	Protocol           PropertyProtocol    `field:"protocol" json:"protocol,omitempty" bson:"protocol,omitempty"`
	Description        PropertyString      `field:"description" json:"description,omitempty" bson:"description,omitempty"`
	StartParamRegex    PropertyString      `field:"bk_start_param_regex" json:"bk_start_param_regex,omitempty" bson:"bk_start_param_regex,omitempty"`
}

func (pt *ProcessProperty) Validate() (field string, err error) {
	// call all field's Validate method one by one
	propertyInterfaceType := reflect.TypeOf((*ProcessPropertyInterface)(nil)).Elem()
	selfVal := reflect.ValueOf(pt).Elem()
	selfType := reflect.TypeOf(pt).Elem()
	fieldCount := selfVal.NumField()
	for fieldIdx := 0; fieldIdx < fieldCount; fieldIdx++ {
		field := selfType.Field(fieldIdx)
		fieldVal := selfVal.Field(fieldIdx)

		// check implements interface
		fieldValType := fieldVal.Addr().Type()
		if !fieldValType.Implements(propertyInterfaceType) {
			msg := fmt.Sprintf("field %s of type: %s should implements %s", field.Name, fieldVal.Type().Elem().Name(), propertyInterfaceType.Name())
			panic(msg)
		}

		// call validate method by interface
		checkResult := fieldVal.Addr().MethodByName("Validate").Call([]reflect.Value{})
		out := checkResult[0]
		if !out.IsNil() {
			err := out.Interface().(error)
			tag := field.Tag.Get("json")
			fieldName := strings.Split(tag, ",")[0]
			return fieldName, err
		}
	}

	if pt.ProcessName.Value == nil || len(*pt.ProcessName.Value) == 0 {
		return "bk_process_name", fmt.Errorf("field [%s] is required", "bk_process_name")
	}
	if pt.FuncName.Value == nil || len(*pt.FuncName.Value) == 0 {
		return "bk_func_name", fmt.Errorf("field [%s] is required", "bk_func_name")
	}
	if pt.AutoTimeGapSeconds.Value != nil {
		if *pt.AutoTimeGapSeconds.Value < 1 || *pt.AutoTimeGapSeconds.Value > 10000 {
			return "auto_time_gap", fmt.Errorf("field %s value must in range [1, 10000]", "auto_time_gap")
		}
	}
	if pt.TimeoutSeconds.Value != nil {
		if *pt.TimeoutSeconds.Value < 1 || *pt.TimeoutSeconds.Value > 10000 {
			return "timeout", fmt.Errorf("field %s value must in range [1, 10000]", "timeout")
		}
	}
	if pt.ProcNum.Value != nil {
		if *pt.ProcNum.Value < 1 || *pt.ProcNum.Value > 10000 {
			return "proc_num", fmt.Errorf("field %s value must in range [1, 10000]", "proc_num")
		}
	}
	if pt.Priority.Value != nil {
		if *pt.Priority.Value < 1 || *pt.Priority.Value > 10000 {
			return "priority", fmt.Errorf("field %s value must in range [1, 10000]", "priority")
		}
	}
	return "", nil
}

// Update all not nil field from input to pt
func (pt *ProcessProperty) Update(input ProcessProperty) {
	selfType := reflect.TypeOf(pt).Elem()
	selfVal := reflect.ValueOf(pt).Elem()
	inputVal := reflect.ValueOf(input)
	fieldCount := selfVal.NumField()
	updateIgnoreField := []string{"FuncName", "ProcessName"}
	for fieldIdx := 0; fieldIdx < fieldCount; fieldIdx++ {
		fieldName := selfType.Field(fieldIdx).Name
		if util.InArray(fieldName, updateIgnoreField) == true {
			continue
		}
		inputField := inputVal.Field(fieldIdx)
		selfField := selfVal.Field(fieldIdx)
		subFieldCount := inputField.NumField()
		// subFields: Value, AsDefaultValue
		for subFieldIdx := 0; subFieldIdx < subFieldCount; subFieldIdx++ {
			inputFieldPtr := inputField.Field(subFieldIdx)
			if inputFieldPtr.IsNil() {
				continue
			}
			inputFieldValue := inputFieldPtr.Elem()

			selfFieldValuePtr := selfField.Field(subFieldIdx)
			if selfFieldValuePtr.Kind() == reflect.Ptr {
				if selfFieldValuePtr.IsNil() && selfFieldValuePtr.CanSet() {
					selfFieldValuePtr.Set(reflect.New(selfFieldValuePtr.Type().Elem()))
				}
			}

			selfFieldValue := selfFieldValuePtr.Elem()
			selfFieldValue.Set(inputFieldValue)
		}
	}
	return
}

type ProcessPropertyInterface interface {
	Validate() error
}

type PropertyInt64 struct {
	Value *int64 `field:"value" json:"value" bson:"value"`

	// AsDefaultValue records the relations between process instance's property and
	// whether it's used as a default value, the empty value can also be a default value.
	// If the property's value is used as a default value, then this property
	// can not be changed in all the process instance's created by this process
	// template. or, it can only be changed to this default value.
	AsDefaultValue *bool `field:"as_default_value" json:"as_default_value" bson:"as_default_value"`
}

func (ti *PropertyInt64) Validate() error {
	return nil
}

// PropertyInt64String is a string field that parse into int64
type PropertyInt64String struct {
	Value *string `field:"value" json:"value" bson:"value"`

	// AsDefaultValue records the relations between process instance's property and
	// whether it's used as a default value, the empty value can also be a default value.
	// If the property's value is used as a default value, then this property
	// can not be changed in all the process instance's created by this process
	// template. or, it can only be changed to this default value.
	AsDefaultValue *bool `field:"as_default_value" json:"as_default_value" bson:"as_default_value"`
}

func (ti *PropertyInt64String) Validate() error {
	if ti.Value != nil {
		// 兼容前端
		if *ti.Value == "" {
			ti.Value = nil
			return nil
		}

		if _, err := strconv.ParseInt(*ti.Value, 10, 64); err != nil {
			return err
		}
	}
	return nil
}

type PropertyBool struct {
	Value          *bool `field:"value" json:"value" bson:"value"`
	AsDefaultValue *bool `field:"as_default_value" json:"as_default_value" bson:"as_default_value"`
}

func (ti *PropertyBool) Validate() error {
	return nil
}

type PropertyString struct {
	Value          *string `field:"value" json:"value" bson:"value"`
	AsDefaultValue *bool   `field:"as_default_value" json:"as_default_value" bson:"as_default_value"`
}

func (ti *PropertyString) Validate() error {
	return nil
}

var (
	ProcessPortFormat = regexp.MustCompile(`^(((([1-9][0-9]{0,3})|([1-5][0-9]{4})|(6[0-4][0-9]{3})|(65[0-4][0-9]{2})|(655[0-2][0-9])|(6553[0-5]))-(([1-9][0-9]{0,3})|([1-5][0-9]{4})|(6[0-4][0-9]{3})|(65[0-4][0-9]{2})|(655[0-2][0-9])|(6553[0-5])))|((([1-9][0-9]{0,3})|([1-5][0-9]{4})|(6[0-4][0-9]{3})|(65[0-4][0-9]{2})|(655[0-2][0-9])|(6553[0-5]))))(,(((([1-9][0-9]{0,3})|([1-5][0-9]{4})|(6[0-4][0-9]{3})|(65[0-4][0-9]{2})|(655[0-2][0-9])|(6553[0-5])))|((([1-9][0-9]{0,3})|([1-5][0-9]{4})|(6[0-4][0-9]{3})|(65[0-4][0-9]{2})|(655[0-2][0-9])|(6553[0-5]))-(([1-9][0-9]{0,3})|([1-5][0-9]{4})|(6[0-4][0-9]{3})|(65[0-4][0-9]{2})|(655[0-2][0-9])|(6553[0-5])))))*$`)
)

type PropertyPort struct {
	Value          *string `field:"value" json:"value" bson:"value"`
	AsDefaultValue *bool   `field:"as_default_value" json:"as_default_value" bson:"as_default_value"`
}

func (ti *PropertyPort) Validate() error {
	if ti.Value != nil {
		if len(*ti.Value) == 0 {
			return nil
		}
		if matched := ProcessPortFormat.MatchString(*ti.Value); matched == false {
			return fmt.Errorf("port format invalid")
		}
	}
	return nil
}

type PropertyBindIP struct {
	Value          *SocketBindType `field:"value" json:"value" bson:"value"`
	AsDefaultValue *bool           `field:"as_default_value" json:"as_default_value" bson:"as_default_value"`
}

func (ti *PropertyBindIP) Validate() error {
	if ti.Value != nil && len(*ti.Value) != 0 {
		if err := ti.Value.Validate(); err != nil {
			return err
		}
	}
	return nil
}

type PropertyProtocol struct {
	Value          *ProtocolType `field:"value" json:"value" bson:"value"`
	AsDefaultValue *bool         `field:"as_default_value" json:"as_default_value" bson:"as_default_value"`
}

func (ti *PropertyProtocol) Validate() error {
	if ti.Value != nil {
		if err := ti.Value.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// ServiceInstance is a service, which created when a host binding with a service template.
type ServiceInstance struct {
	Metadata Metadata        `field:"metadata" json:"metadata" bson:"metadata"`
	ID       int64           `field:"id" json:"id,omitempty" bson:"id"`
	Name     string          `field:"name" json:"name,omitempty" bson:"name"`
	Labels   selector.Labels `field:"labels" json:"labels,omitempty" bson:"labels"`

	// the template id can not be updated, once the service is created.
	// it can be 0 when the service is not created with a service template.
	ServiceTemplateID int64  `field:"service_template_id" json:"service_template_id,omitempty" bson:"service_template_id"`
	HostID            int64  `field:"bk_host_id" json:"bk_host_id,omitempty" bson:"bk_host_id"`
	InnerIP           string `field:"bk_host_innerip" json:"bk_host_innerip,omitempty" bson:"bk_host_innerip"`

	// the module that this service belongs to.
	ModuleID int64 `field:"bk_module_id" json:"bk_module_id,omitempty" bson:"bk_module_id"`

	Creator         string    `field:"creator" json:"creator,omitempty" bson:"creator"`
	Modifier        string    `field:"modifier" json:"modifier,omitempty" bson:"modifier"`
	CreateTime      time.Time `field:"create_time" json:"create_time,omitempty" bson:"create_time"`
	LastTime        time.Time `field:"last_time" json:"last_time,omitempty" bson:"last_time"`
	SupplierAccount string    `field:"bk_supplier_account" json:"bk_supplier_account,omitempty" bson:"bk_supplier_account"`
}

func (si *ServiceInstance) Validate() (field string, err error) {
	/*
		if len(si.Name) == 0 {
			return "name", errors.New("name can't be empty")
		}
	*/

	if len(si.Name) > common.NameFieldMaxLength {
		return "name", fmt.Errorf("name too long, input: %d > max: %d", len(si.Name), common.NameFieldMaxLength)
	}
	return "", nil
}

type ServiceInstanceDetail struct {
	ServiceInstance
	ServiceCategoryID int64               `field:"service_category_id" json:"service_category_id" bson:"service_category_id"`
	ProcessInstances  []ProcessInstanceNG `field:"process_instances" json:"process_instances" bson:"process_instances"`
}

type ServiceInstanceWithTopoPath struct {
	ServiceInstance
	TopoPath []TopoInstanceNodeSimplify `field:"topo_path" json:"topo_path" bson:"topo_path"`
}

// ProcessInstanceRelation record which service instance and process template are current process binding, process identified by ProcessID
type ProcessInstanceRelation struct {
	Metadata Metadata `field:"metadata" json:"metadata" bson:"metadata"`

	// unique field, 1:1 mapping with ProcessInstance.
	ProcessID         int64 `field:"bk_process_id" json:"bk_process_id" bson:"bk_process_id"`
	ServiceInstanceID int64 `field:"service_instance_id" json:"service_instance_id" bson:"service_instance_id"`

	// ProcessTemplateID indicate which template are current process instantiate from.
	ProcessTemplateID int64 `field:"process_template_id" json:"process_template_id" bson:"process_template_id"`

	// redundant field for accelerating processes by HostID
	HostID          int64  `field:"bk_host_id" json:"bk_host_id" bson:"bk_host_id"`
	SupplierAccount string `field:"bk_supplier_account" json:"bk_supplier_account" bson:"bk_supplier_account"`
}

func (pir *ProcessInstanceRelation) Validate() (field string, err error) {
	return "", nil
}

type ProcessInstance struct {
	Property mapstr.MapStr           `json:"property"`
	Relation ProcessInstanceRelation `json:"relation"`
}

type ProcessInstanceNG struct {
	Process  Process                 `json:"process"`
	Relation ProcessInstanceRelation `json:"relation"`
}

type GetProc2ModuleOption struct {
	ProcessID int64 `json:"bk_process_id"`
	BizID     int64 `json:"bk_biz_id"`
}

type Proc2Module struct {
	BizID           int    `json:"bk_biz_id"`
	ModuleName      string `json:"bk_module_name"`
	ProcessID       int    `json:"bk_process_id"`
	SupplierAccount string `json:"bk_supplier_account"`
}

type LabelAggregationOption struct {
	Metadata Metadata `json:"metadata"`
	ModuleID *int64   `json:"bk_module_id" bson:"bk_module_id" field:"bk_module_id"`
}
