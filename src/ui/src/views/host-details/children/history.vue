<template>
    <div class="history">
        <div class="history-filter">
            <cmdb-form-date-range class="filter-item filter-range"
                v-model="dateRange"
                @input="handlePageChange(1)">
            </cmdb-form-date-range>
            <cmdb-form-objuser class="filter-item filter-user"
                v-model="operator"
                :exclude="false"
                :multiple="false"
                :palceholder="$t('操作账号')"
                @input="handlePageChange(1)">
            </cmdb-form-objuser>
        </div>
        <bk-table class="history-table"
            v-bkloading="{ isLoading: $loading('getHostAuditLog') }"
            :data="history"
            :pagination="pagination"
            @page-change="handlePageChange"
            @page-limit-change="handleSizeChange"
            @sort-change="handleSortChange"
            @row-click="handleRowClick">
            <bk-table-column prop="op_desc" :label="$t('变更内容')" sortable="custom"></bk-table-column>
            <bk-table-column prop="operator" :label="$t('操作账号')" sortable="custom"></bk-table-column>
            <bk-table-column prop="op_time" :label="$t('操作时间')" sortable="custom">
                <template slot-scope="{ row }">
                    {{$tools.formatTime(row['op_time'])}}
                </template>
            </bk-table-column>
        </bk-table>
        <bk-sideslider
            :is-show.sync="details.show"
            :width="800"
            :title="$t('操作详情')">
            <cmdb-host-history-details :details="details.data" slot="content" v-if="details.show"></cmdb-host-history-details>
        </bk-sideslider>
    </div>
</template>

<script>
    import cmdbHostHistoryDetails from '@/components/audit-history/details'
    export default {
        name: 'cmdb-host-history',
        components: {
            cmdbHostHistoryDetails
        },
        data () {
            return {
                dateRange: [],
                operator: '',
                history: [],
                pagination: {
                    count: 0,
                    current: 1,
                    limit: 10
                },
                sort: '-op_time',
                details: {
                    show: false,
                    data: null
                }
            }
        },
        computed: {
            id () {
                return parseInt(this.$route.params.id)
            }
        },
        created () {
            this.getHistory()
        },
        methods: {
            async getHistory () {
                try {
                    const condition = {
                        op_target: 'host',
                        inst_id: this.id
                    }
                    if (this.dateRange.length) {
                        condition.op_time = [this.dateRange[0] + ' 00:00:00', this.dateRange[1] + ' 23:59:59']
                    }
                    if (this.operator) {
                        condition.operator = this.operator
                    }
                    const data = await this.$http.post('object/host/audit/search', {
                        condition,
                        limit: this.pagination.limit,
                        sort: this.sort,
                        start: (this.pagination.current - 1) * this.pagination.limit
                    }, {
                        requestId: 'getHostAuditLog'
                    })
                    this.history = data.info
                    this.pagination.count = data.count
                } catch (e) {
                    console.log(e)
                    this.history = []
                    this.pagination.count = 0
                }
            },
            handlePageChange (page) {
                this.pagination.current = page
                this.getHistory()
            },
            handleSizeChange (size) {
                this.pagination.limit = size
                this.pagination.current = 1
                this.getHistory()
            },
            handleSortChange (sort) {
                this.sort = sort
                this.getHistory()
            },
            handleRowClick (item) {
                this.details.data = item
                this.details.show = true
            }
        }
    }
</script>

<style lang="scss" scoped>
    .history {
        height: 100%;
    }
    .history-filter {
        padding: 14px 0;
        .filter-item {
            display: inline-block;
            vertical-align: middle;
            &.filter-range {
                width: 300px !important;
                margin: 0 5px 0 0;
            }
            &.filter-user {
                width: 240px;
                height: 32px;
            }
        }
    }
</style>
