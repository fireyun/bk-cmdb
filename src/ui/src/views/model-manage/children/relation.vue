<template>
    <div class="model-relation-wrapper">
        <div class="options">
            <span v-cursor="{
                active: !$isAuthorized($OPERATION.U_MODEL),
                auth: [$OPERATION.U_MODEL]
            }">
                <bk-button class="create-btn" theme="primary"
                    :disabled="isReadOnly || !updateAuth"
                    @click="createRelation">
                    {{$t('新建关联')}}
                </bk-button>
            </span>
        </div>
        <bk-table
            class="relation-table"
            v-bkloading="{ isLoading: $loading() }"
            :data="table.list"
            :max-height="$APP.height - 220"
            :row-style="{
                cursor: 'pointer'
            }"
            @cell-click="handleShowDetails">
            <bk-table-column prop="bk_obj_asst_id" :label="$t('唯一标识')" class-name="is-highlight">
                <template slot-scope="{ row }">
                    <span
                        v-if="row.ispre"
                        :class="['relation-pre', $i18n.locale]">
                        {{$t('内置')}}
                    </span>
                    <span class="relation-id">{{row['bk_obj_asst_id']}}</span>
                </template>
            </bk-table-column>
            <bk-table-column prop="bk_asst_name" :label="$t('关联类型')">
                <template slot-scope="{ row }">
                    {{getRelationName(row['bk_asst_id'])}}
                </template>
            </bk-table-column>
            <bk-table-column prop="mapping" :label="$t('源-目标约束')">
                <template slot-scope="{ row }">
                    {{mappingMap[row.mapping]}}
                </template>
            </bk-table-column>
            <bk-table-column prop="bk_obj_name" :label="$t('源模型')">
                <template slot-scope="{ row }">
                    {{getModelName(row['bk_obj_id'])}}
                </template>
            </bk-table-column>
            <bk-table-column prop="bk_asst_obj_name" :label="$t('目标模型')">
                <template slot-scope="{ row }">
                    {{getModelName(row['bk_asst_obj_id'])}}
                </template>
            </bk-table-column>
            <bk-table-column prop="operation" :label="$t('操作')" v-if="updateAuth">
                <template slot-scope="{ row }">
                    <button class="text-primary mr10 operation-btn"
                        :disabled="!isEditable(row)"
                        @click.stop="editRelation(row)">
                        {{$t('编辑')}}
                    </button>
                    <button class="text-primary operation-btn"
                        :disabled="!isEditable(row)"
                        @click.stop="deleteRelation(row)">
                        {{$t('删除')}}
                    </button>
                </template>
            </bk-table-column>
        </bk-table>
        <bk-sideslider
            :width="450"
            :title="slider.title"
            :is-show.sync="slider.isShow">
            <the-relation-detail
                class="slider-content"
                slot="content"
                v-if="slider.isShow"
                :is-read-only="isReadOnly || slider.isReadOnly"
                :is-edit="slider.isEdit"
                :relation="slider.relation"
                :relation-list="relationList"
                @save="saveRelation"
                @cancel="slider.isShow = false">
            </the-relation-detail>
        </bk-sideslider>
    </div>
</template>

<script>
    import theRelationDetail from './relation-detail'
    import { mapGetters, mapActions } from 'vuex'
    export default {
        components: {
            theRelationDetail
        },
        data () {
            return {
                slider: {
                    isShow: false,
                    isEdit: false,
                    title: this.$t('新建关联'),
                    relation: {}
                },
                relationList: [],
                table: {
                    list: [],
                    defaultSort: '-op_time',
                    sort: '-op_time'
                },
                mappingMap: {
                    '1:1': '1-1',
                    '1:n': '1-N',
                    'n:n': 'N-N'
                }
            }
        },
        computed: {
            ...mapGetters(['isAdminView', 'isBusinessSelected']),
            ...mapGetters('objectModel', [
                'activeModel',
                'isInjectable'
            ]),
            ...mapGetters('objectModelClassify', ['models']),
            isReadOnly () {
                if (this.activeModel) {
                    return this.activeModel['bk_ispaused']
                }
                return false
            },
            updateAuth () {
                const cantEdit = ['process', 'plat']
                if (cantEdit.includes(this.$route.params.modelId)) {
                    return false
                }
                const editable = this.isAdminView || (this.isBusinessSelected && this.isInjectable)
                return editable && this.$isAuthorized(this.$OPERATION.U_MODEL)
            }
        },
        created () {
            this.searchRelationList()
            this.initRelationList()
        },
        methods: {
            ...mapActions('objectAssociation', [
                'searchObjectAssociation',
                'deleteObjectAssociation',
                'searchAssociationType'
            ]),
            isEditable (item) {
                if (item.ispre || item['bk_asst_id'] === 'bk_mainline' || this.isReadOnly) {
                    return false
                }
                if (!this.isAdminView) {
                    return !!this.$tools.getMetadataBiz(item)
                }
                return true
            },
            getRelationName (id) {
                const relation = this.relationList.find(item => item.id === id)
                if (relation) {
                    return relation.name
                }
            },
            async initRelationList () {
                const data = await this.searchAssociationType({
                    params: {},
                    config: {
                        requestId: 'post_searchAssociationType',
                        fromCache: true
                    }
                })
                this.relationList = data.info.map(({ bk_asst_id: asstId, bk_asst_name: asstName }) => {
                    if (asstName.length) {
                        return {
                            id: asstId,
                            name: `${asstId}(${asstName})`
                        }
                    }
                    return {
                        id: asstId,
                        name: asstId
                    }
                })
            },
            getModelName (objId) {
                const model = this.models.find(model => model['bk_obj_id'] === objId)
                if (model) {
                    return model['bk_obj_name']
                }
                return ''
            },
            createRelation () {
                this.slider.isEdit = false
                this.slider.isReadOnly = false
                this.slider.relation = {}
                this.slider.title = this.$t('新建关联')
                this.slider.isShow = true
            },
            editRelation (item) {
                this.slider.isEdit = true
                this.slider.isReadOnly = false
                this.slider.relation = item
                this.slider.title = this.$t('编辑关联')
                this.slider.isShow = true
            },
            deleteRelation (relation) {
                this.$bkInfo({
                    title: this.$t('确定删除关联关系?'),
                    confirmFn: async () => {
                        await this.deleteObjectAssociation({
                            id: relation.id,
                            config: {
                                data: this.$injectMetadata({}, { inject: this.isInjectable }),
                                requestId: 'deleteObjectAssociation'
                            }
                        }).then(() => {
                            this.$http.cancel(`post_searchObjectAssociation_${this.activeModel['bk_obj_id']}`)
                        })
                        this.searchRelationList()
                    }
                })
            },
            async searchRelationList () {
                const [source, dest] = await Promise.all([this.searchAsSource(), this.searchAsDest()])
                this.table.list = [...source, ...dest.filter(des => !source.some(src => src.id === des.id))]
            },
            searchAsSource () {
                return this.searchObjectAssociation({
                    params: this.$injectMetadata({
                        condition: {
                            'bk_obj_id': this.activeModel['bk_obj_id']
                        }
                    }, {
                        inject: this.isInjectable
                    })
                })
            },
            searchAsDest () {
                return this.searchObjectAssociation({
                    params: this.$injectMetadata({
                        condition: {
                            'bk_asst_obj_id': this.activeModel['bk_obj_id']
                        }
                    }, {
                        inject: this.isInjectable
                    })
                })
            },
            saveRelation () {
                this.slider.isShow = false
                this.searchRelationList()
            },
            handleShowDetails (row, column, cell) {
                if (column.property === 'operation') return
                this.slider.isEdit = true
                this.slider.isReadOnly = true
                this.slider.relation = row
                this.slider.title = this.$t('查看关联')
                this.slider.isShow = true
            }
        }
    }
</script>

<style lang="scss" scoped>
    .options {
        padding: 20px 0 14px;
    }
    .relation-pre {
        display: inline-block;
        margin-right: -26px;
        padding: 0 6px;
        vertical-align: middle;
        line-height: 32px;
        border-radius: 4px;
        background-color: #a4aab3;
        color: #fff;
        font-size: 20px;
        transform: scale(0.5);
        transform-origin: left center;
        opacity: 0.4;
        &.en {
            margin-right: -40px;
        }
    }
    .relation-id {
        vertical-align: middle;
    }
    .text-primary {
        cursor: pointer;
    }
    .operation-btn[disabled] {
        color: #dcdee5 !important;
        opacity: 1 !important;
    }
</style>
