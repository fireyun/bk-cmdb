<template>
    <div class="authority-wrapper" v-bkloading="{ isLoading: $loading('searchUserPrivilege') }">
        <div class="authority-box">
            <div class="authority-group clearfix">
                <h2 class="authority-group-title">{{$t('系统权限')}}</h2>
                <div class="authority-group-content">
                    <div class="authority-type system clearfix"
                        v-for="(config, configId) in sysConfig"
                        :key="configId"
                        v-if="config.authorities.length">
                        <h3 class="system-title fl">{{$t(config.name)}}：</h3>
                        <ul class="system-list fl">
                            <li class="system-item fl"
                                v-for="(authority, index) in config.authorities"
                                :key="index">
                                <label class="cmdb-form-checkbox cmdb-checkbox-small"
                                    :for="'systemAuth-' + authority.id"
                                    :title="$t(authority.name)">
                                    <input type="checkbox"
                                        :id="'systemAuth-' + authority.id"
                                        :value="authority.id"
                                        v-model="config.selectedAuthorities">
                                    <span class="cmdb-checkbox-text">{{$t(authority.name)}}</span>
                                </label>
                            </li>
                        </ul>
                    </div>
                </div>
            </div>
            <div class="authority-group model clearfix">
                <h2 class="authority-group-title"><span>{{$t('模型权限')}}</span></h2>
                <div class="authority-group-content">
                    <div class="authority-type model" v-for="(classify,classifyIndex) in localClassifications"
                        :key="classifyIndex"
                        v-if="classify.models.length">
                        <h3 class="classify-name clearfix" :title="classify.name" @click="classify.open = !classify.open">
                            <i class="bk-icon icon-angle-down angle fl" :class="{ 'open': classify.open }"></i>
                            <span class="fl">{{classify.name}}</span>
                        </h3>
                        <transition name="slide">
                            <ul class="model-list" v-show="classify.open" :style="calcModelListStyle(classify.models.length)">
                                <li class="model-item clearfix" v-for="(model,modelIndex) in classify.models" :key="modelIndex">
                                    <h4 class="model-authority fl" :title="model['bk_obj_name']">{{model['bk_obj_name']}}：</h4>
                                    <span class="model-authority-checkbox fl first">
                                        <label class="cmdb-form-checkbox cmdb-checkbox-small"
                                            :for="'model-all-' + model['bk_obj_id']">
                                            <input type="checkbox"
                                                :id="'model-all-' + model['bk_obj_id']"
                                                :checked="model.selectedAuthorities.length === 3"
                                                @change="checkAllModelAuthorities(classifyIndex,modelIndex,$event)">
                                            <span class="cmdb-checkbox-text">{{$t('全选')}}</span>
                                        </label>
                                    </span>
                                    <span class="model-authority-checkbox fl">
                                        <label class="cmdb-form-checkbox cmdb-checkbox-small"
                                            :for="'model-search-' + model['bk_obj_id']">
                                            <input type="checkbox" value="search"
                                                :id="'model-search-' + model['bk_obj_id']"
                                                v-model="model.selectedAuthorities"
                                                @change="checkOtherAuthorities(classifyIndex,modelIndex,$event)">
                                            <span class="cmdb-checkbox-text">{{$t('查询')}}</span>
                                        </label>
                                    </span>
                                    <span class="model-authority-checkbox fl">
                                        <label class="cmdb-form-checkbox cmdb-checkbox-small"
                                            :for="'model-update-' + model['bk_obj_id']"
                                            :class="{ 'disabled': model.selectedAuthorities.indexOf('search') === -1 }">
                                            <input type="checkbox" value="update"
                                                :id="'model-update-' + model['bk_obj_id']"
                                                :disabled="model.selectedAuthorities.indexOf('search') === -1"
                                                v-model="model.selectedAuthorities">
                                            <span class="cmdb-checkbox-text">{{$t('编辑')}}</span>
                                        </label>
                                    </span>
                                    <span class="model-authority-checkbox fl">
                                        <label class="cmdb-form-checkbox cmdb-checkbox-small"
                                            :for="'model-delete-' + model['bk_obj_id']"
                                            :class="{ 'disabled': model.selectedAuthorities.indexOf('search') === -1 }">
                                            <input type="checkbox" value="delete"
                                                :id="'model-delete-' + model['bk_obj_id']"
                                                :disabled="model.selectedAuthorities.indexOf('search') === -1"
                                                v-model="model.selectedAuthorities">
                                            <span class="cmdb-checkbox-text">{{$t('删除')}}</span>
                                        </label>
                                    </span>
                                </li>
                            </ul>
                        </transition>
                    </div>
                </div>
            </div>
        </div>
        <footer class="footer">
            <bk-button theme="primary" :loading="$loading('updateGroupAuthorities')" @click="updateGroupAuthorities">
                {{$t('保存')}}
            </bk-button>
            <bk-button theme="default" @click="cancel">
                {{$t('取消')}}
            </bk-button>
        </footer>
    </div>
</template>

<script>
    import { mapGetters, mapActions } from 'vuex'
    export default {
        props: {
            groupId: {
                type: String
            }
        },
        data () {
            return {
                sysConfig: {
                    global_busi: {
                        id: 'global_busi',
                        name: '全局业务',
                        authorities: [{
                            id: 'resource',
                            name: '资源池管理'
                        }],
                        selectedAuthorities: []
                    },
                    back_config: {
                        id: 'back_config',
                        name: '后台配置',
                        authorities: [{
                            id: 'event',
                            name: '事件推送配置'
                        }, {
                            id: 'audit',
                            name: '操作审计'
                        }],
                        selectedAuthorities: []
                    }
                },
                localClassifications: [],
                groupAuthorities: {},
                hideClassify: ['bk_host_manage', 'bk_biz_topo']
            }
        },
        computed: {
            ...mapGetters('userPrivilege', [
                'roles'
            ]),
            ...mapGetters('objectModelClassify', [
                'classifications'
            ]),
            updateParams () {
                const updateParams = {}
                for (const config in this.sysConfig) {
                    if (this.sysConfig[config].selectedAuthorities.length) {
                        updateParams.sys_config = updateParams.sys_config || {}
                        updateParams.sys_config[config] = this.sysConfig[config].selectedAuthorities
                    }
                }
                this.localClassifications.map((classify) => {
                    classify.models.map((model) => {
                        if (model.selectedAuthorities.length) {
                            updateParams['model_config'] = updateParams['model_config'] || {}
                            updateParams['model_config'][classify.id] = updateParams['model_config'][classify.id] || {}
                            updateParams['model_config'][classify.id][model['bk_obj_id']] = model.selectedAuthorities
                        }
                    })
                })
                return updateParams
            }
        },
        created () {
            this.getGroupAuthorities()
        },
        methods: {
            ...mapActions('userPrivilege', [
                'searchUserPrivilege',
                'updateGroupPrivilege'
            ]),
            cancel () {
                this.$emit('cancel')
            },
            async updateGroupAuthorities () {
                await this.updateGroupPrivilege({
                    bkGroupId: this.groupId,
                    params: this.updateParams,
                    config: {
                        requestId: 'updateGroupAuthorities'
                    }
                })
                this.$success(this.$t('保存成功'))
                this.$emit('cancel')
            },
            checkAllModelAuthorities (classifyIndex, modelIndex, event) {
                const model = this.localClassifications[classifyIndex]['models'][modelIndex]
                if (event.target.checked) {
                    model.selectedAuthorities = ['search', 'update', 'delete']
                } else {
                    model.selectedAuthorities = []
                }
            },
            async getGroupAuthorities () {
                const res = await this.searchUserPrivilege({
                    bkGroupId: this.groupId,
                    config: {
                        requestId: 'searchUserPrivilege'
                    }
                })
                this.groupAuthorities = res.privilege
                this.initSystemAuthorities()
                this.initClassifications()
            },
            calcModelListStyle (total) {
                return {
                    height: `${total * 32}px`
                }
            },
            /* 模型权限没有选择'查询'，则无'新增'、编辑'、删除'权限 */
            checkOtherAuthorities (classifyIndex, modelIndex, event) {
                const model = this.localClassifications[classifyIndex]['models'][modelIndex]
                if (!event.target.checked) {
                    model.selectedAuthorities = []
                }
            },
            initSystemAuthorities () {
                if (this.groupAuthorities.hasOwnProperty('sys_config')) {
                    for (const configId in this.sysConfig) {
                        if (this.groupAuthorities['sys_config'].hasOwnProperty(configId)) {
                            this.sysConfig[configId].selectedAuthorities = this.groupAuthorities['sys_config'][configId] || []
                        } else {
                            this.sysConfig[configId].selectedAuthorities = []
                        }
                    }
                } else {
                    for (const configId in this.sysConfig) {
                        this.sysConfig[configId].selectedAuthorities = []
                    }
                }
            },
            initClassifications () {
                const localClassifications = []
                // 1.去掉停用模型
                let activeClassifications = this.classifications.map(classification => {
                    const activeClassification = { ...classification }
                    activeClassification['bk_objects'] = activeClassification['bk_objects'].filter(model => !model['bk_ispaused'])
                    return activeClassification
                })
                // 2.去掉无启用模型的分类和不显示的分类
                activeClassifications = activeClassifications.filter(classification => {
                    const {
                        'bk_classification_id': bkClassificationId,
                        'bk_objects': bkObjects
                    } = classification
                    return !this.hideClassify.includes(bkClassificationId) && Array.isArray(bkObjects) && bkObjects.length
                })
                const authorities = this.groupAuthorities
                activeClassifications.map(classify => {
                    const models = []
                    const classifyId = classify['bk_classification_id']
                    if (this.hideClassify.indexOf(classifyId) === -1) {
                        classify['bk_objects'].forEach((model) => {
                            let selectedAuthorities = []
                            if (
                                authorities.hasOwnProperty('model_config')
                                && authorities['model_config'].hasOwnProperty(classifyId)
                                && authorities['model_config'][classifyId].hasOwnProperty(model['bk_obj_id'])
                            ) {
                                selectedAuthorities = authorities['model_config'][classifyId][model['bk_obj_id']]
                            }
                            models.push(Object.assign({}, model, { selectedAuthorities }))
                        })
                        localClassifications.push({
                            id: classify['bk_classification_id'],
                            name: classify['bk_classification_name'],
                            open: true,
                            models: models
                        })
                    }
                })
                this.localClassifications = localClassifications
            }
        }
    }
</script>

<style lang="scss" scoped>
    .authority-wrapper{
        height: 100%;
        .authority-box {
            padding: 30px 20px 0 30px;
            max-height: calc(100% - 76px);
            @include scrollbar;
        }
    }
    .authority-group{
        font-size: 0;
        &.model{
            margin-top: 14px;
            .authority-group-content{
                padding: 0;
            }
        }
        .authority-group-title{
            font-weight: bold;
            font-size: 14px;
            line-height: 1;
        }
    }
    .authority-type.system{
        line-height: 32px;
        &:first-child{
            margin-top: 10px;
        }
        .system-title{
            width: 100px;
            margin: 0;
            font-size: 14px;
            font-weight: normal;
            text-align: right;
            @include ellipsis;
        }
        .system-list{
            white-space: nowrap;
            .system-item{
                min-width: 115px;
                height: 32px;
                margin: 0 0 0 5px;
            }
        }
    }
    .authority-type.model{
        line-height: 32px;
        .classify-name{
            font-size: 12px;
            cursor: pointer;
            margin-top: 10px;
            .icon-angle-down{
                font-size: 12px;
                margin: 9px 8px 0 0;
                font-weight: bold;
                transform: rotate(180deg);
                transition: transform .5s cubic-bezier(.23, 1, .23, 1);
                &.open{
                    transform: rotate(0);
                }
            }
        }
        .model-list{
            .model-item{
                &:first-child{
                    padding-top: 0;
                }
            }
        }
        .model-authority{
            width: 100px;
            margin: 0;
            font-size: 14px;
            font-weight: normal;
            text-align: right;
            @include ellipsis;
        }
        .model-authority-checkbox{
            width: 115px;
            height: 32px;
            margin: 0 0 0 5px;
            &.first {
                width: 115px;
            }
            &:last-child{
                width: auto;
            }
        }
    }
    label.cmdb-form-checkbox{
        line-height: 32px;
        padding: 0;
        margin: 0;
        overflow: hidden;
        text-overflow: ellipsis;
        cursor: pointer;
        border: none;
        &.disabled{
            cursor: not-allowed;
            color: #c3cdd7;
        }
        input[type='checkbox']{
            width: 14px;
            height: 14px;
            margin-right: 10px;
        }
    }
    .footer{
        padding: 20px 0 20px 135px;
        background: #fff;
        font-size: 0;
        .bk-button:first-child {
            margin-right: 10px;
        }
    }
    .slide-enter-active, .slide-leave-active{
        transition: height .5s cubic-bezier(.23, 1, .23, 1);
        overflow: hidden;
    }
    .slide-enter, .slide-leave-to{
        height: 0 !important;
    }
</style>
