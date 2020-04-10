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

package local

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/common/util"
	"configcenter/src/storage/dal"
	"configcenter/src/storage/dal/types"
	dtype "configcenter/src/storage/types"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"
	"gopkg.in/redis.v5"
)

type Mongo struct {
	dbc    *mongo.Client
	dbname string
	sess   mongo.Session
	tm     *TxnManager
}

var _ dal.DB = new(Mongo)

type MongoConf struct {
	TimeoutSeconds int
	MaxOpenConns   uint64
	MaxIdleConns   uint64
	URI            string
	RsName         string
}

// NewMgo returns new RDB
func NewMgo(config MongoConf, timeout time.Duration) (*Mongo, error) {
	connStr, err := connstring.Parse(config.URI)
	if nil != err {
		return nil, err
	}
	if config.RsName == "" {
		return nil, fmt.Errorf("rsName not set")
	}

	conOpt := options.ClientOptions{
		MaxPoolSize:    &config.MaxOpenConns,
		MinPoolSize:    &config.MaxIdleConns,
		ConnectTimeout: &timeout,
		ReplicaSet:     &config.RsName,
	}

	client, err := mongo.NewClient(options.Client().ApplyURI(config.URI), &conOpt)
	if nil != err {
		return nil, err
	}

	if err := client.Connect(context.TODO()); nil != err {
		return nil, err
	}

	return &Mongo{
		dbc:    client,
		dbname: connStr.Database,
		tm:     &TxnManager{},
	}, nil
}

// NewMgo returns new RDB
func (c *Mongo) InitTxnManager(r *redis.Client) error {
	return c.tm.InitTxnManager(r)
}

// Close replica client
func (c *Mongo) Close() error {
	c.dbc.Disconnect(context.TODO())
	return nil
}

// Ping replica client
func (c *Mongo) Ping() error {
	return c.dbc.Ping(context.TODO(), nil)
}

// IsDuplicatedError check duplicated error
func (c *Mongo) IsDuplicatedError(err error) bool {
	if err != nil {
		if strings.Contains(err.Error(), "The existing index") {
			return true
		}
		if strings.Contains(err.Error(), "There's already an index with name") {
			return true
		}
		if strings.Contains(err.Error(), "E11000 duplicate") {
			return true
		}
	}
	return err == types.ErrDuplicated
}

// IsNotFoundError check the not found error
func (c *Mongo) IsNotFoundError(err error) bool {
	return err == types.ErrDocumentNotFound
}

// Table collection operation
func (c *Mongo) Table(collName string) types.Table {
	col := Collection{}
	col.collName = collName
	col.Mongo = c
	return &col
}

// Collection implement client.Collection interface
type Collection struct {
	collName string // 集合名
	*Mongo
}

// Find 查询多个并反序列化到 Result
func (c *Collection) Find(filter types.Filter) types.Find {
	return &Find{
		Collection: c,
		filter:     filter,
		projection: map[string]int{"_id": 0},
	}
}

// Find define a find operation
type Find struct {
	*Collection

	projection map[string]int
	filter     types.Filter
	start      int64
	limit      int64
	sort       map[string]interface{}
}

// Fields 查询字段
func (f *Find) Fields(fields ...string) types.Find {
	for _, field := range fields {
		if len(field) <= 0 {
			continue
		}
		f.projection[field] = 1
	}
	return f
}

// Sort 查询排序
func (f *Find) Sort(sort string) types.Find {
	if sort != "" {
		sortArr := strings.Split(sort, ",")
		f.sort = make(map[string]interface{}, 0)
		for _, sortItem := range sortArr {
			sortItemArr := strings.Split(sortItem, ":")
			sortKey := strings.TrimLeft(sortItemArr[0], "+-")
			if len(sortItemArr) == 2 {
				sortDescFlag := strings.TrimSpace(sortItemArr[1])
				if sortDescFlag == "-1" {
					f.sort[sortKey] = -1
				} else {
					f.sort[sortKey] = 1
				}
			} else {
				if strings.HasPrefix(sortItemArr[0], "-") {
					f.sort[sortKey] = -1
				} else {
					f.sort[sortKey] = 1
				}
			}
		}
	}

	return f
}

// Start 查询上标
func (f *Find) Start(start uint64) types.Find {
	// change to int64,后续改成int64
	dbStart := int64(start)
	f.start = dbStart
	return f
}

// Limit 查询限制
func (f *Find) Limit(limit uint64) types.Find {
	// change to int64,后续改成int64
	dbLimit := int64(limit)
	f.limit = dbLimit
	return f
}

// All 查询多个
func (f *Find) All(ctx context.Context, result interface{}) error {
	rid := ctx.Value(common.ContextRequestIDField)
	start := time.Now()
	defer func() {
		blog.V(4).InfoDepthf(2, "mongo find-all cost %dms, rid: %v", time.Since(start)/time.Millisecond, rid)
	}()

	// 设置ctx的Session对象,用来处理事务
	se := &mongo.SessionExposer{}

	if f.hasSession(ctx) {
		sess, err := f.getDistributedSession(ctx)
		if err != nil {
			return err
		}
		ctx = se.ContextWithSession(ctx, sess)
		defer f.tm.SaveSession(sess)
	} else if f.sess != nil {
		ctx = se.ContextWithSession(ctx, f.sess)
	}

	findOpts := &options.FindOptions{}
	if len(f.projection) != 0 {
		findOpts.Projection = f.projection
	}
	if f.start != 0 {
		findOpts.SetSkip(f.start)
	}
	if f.limit != 0 {
		findOpts.SetLimit(f.limit)
	}
	if len(f.sort) != 0 {
		findOpts.SetSort(f.sort)
	}
	// 查询条件为空时候，mongodb 不返回数据
	if f.filter == nil {
		f.filter = bson.M{}
	}

	cursor, err := f.dbc.Database(f.dbname).Collection(f.collName).Find(ctx, f.filter, findOpts)
	if err != nil {
		return err
	}
	return cursor.All(ctx, result)
}

// One 查询一个
func (f *Find) One(ctx context.Context, result interface{}) error {
	start := time.Now()
	rid := ctx.Value(common.ContextRequestIDField)
	defer func() {
		blog.V(4).InfoDepthf(2, "mongo find-one cost %dms, rid: %v", time.Since(start)/time.Millisecond, rid)
	}()

	// 设置ctx的Session对象,用来处理事务
	se := &mongo.SessionExposer{}
	if f.hasSession(ctx) {
		sess, err := f.getDistributedSession(ctx)
		if err != nil {
			return err
		}
		ctx = se.ContextWithSession(ctx, sess)
		defer f.tm.SaveSession(sess)
	} else if f.sess != nil {
		ctx = se.ContextWithSession(ctx, f.sess)
	}

	findOpts := &options.FindOptions{}
	if len(f.projection) != 0 {
		findOpts.Projection = f.projection
	}
	if f.start != 0 {
		findOpts.SetSkip(f.start)
	}
	if f.limit != 0 {
		findOpts.SetLimit(1)
	}
	if len(f.sort) != 0 {
		findOpts.SetSort(f.sort)
	}
	// 查询条件为空时候，mongodb panic
	if f.filter == nil {
		f.filter = bson.M{}
	}

	cursor, err := f.dbc.Database(f.dbname).Collection(f.collName).Find(ctx, f.filter, findOpts)
	if err != nil {
		return err
	}

	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		return cursor.Decode(result)
	}
	return types.ErrDocumentNotFound
}

// Count 统计数量(非事务)
func (f *Find) Count(ctx context.Context) (uint64, error) {
	rid := ctx.Value(common.ContextRequestIDField)
	start := time.Now()
	defer func() {
		blog.V(4).InfoDepthf(2, "mongo count cost %dms, rid: %v", time.Since(start)/time.Millisecond, rid)
	}()

	// 设置ctx的Session对象,用来处理事务
	se := &mongo.SessionExposer{}
	if f.hasSession(ctx) {
		sess, err := f.getDistributedSession(ctx)
		if err != nil {
			return uint64(0), err
		}
		ctx = se.ContextWithSession(ctx, sess)
		defer f.tm.SaveSession(sess)
	} else if f.sess != nil {
		ctx = se.ContextWithSession(ctx, f.sess)
	}

	if f.filter == nil {
		f.filter = bson.M{}
	}
	cnt, err := f.dbc.Database(f.dbname).Collection(f.collName).CountDocuments(ctx, f.filter)
	// 后续改成int64
	return uint64(cnt), err
}

// Insert 插入数据, docs 可以为 单个数据 或者 多个数据
func (c *Collection) Insert(ctx context.Context, docs interface{}) error {
	rid := ctx.Value(common.ContextRequestIDField)
	start := time.Now()
	defer func() {
		blog.V(4).InfoDepthf(2, "mongo insert cost %dms, rid: %v", time.Since(start)/time.Millisecond, rid)
	}()

	// 设置ctx的Session对象,用来处理事务
	se := &mongo.SessionExposer{}
	if c.hasSession(ctx) {
		sess, err := c.getDistributedSession(ctx)
		if err != nil {
			return err
		}
		ctx = se.ContextWithSession(ctx, sess)
		defer c.tm.SaveSession(sess)
	} else if c.sess != nil {
		ctx = se.ContextWithSession(ctx, c.sess)
	}

	rows := util.ConverToInterfaceSlice(docs)

	_, err := c.dbc.Database(c.dbname).Collection(c.collName).InsertMany(ctx, rows)
	return err
}

// Update 更新数据
func (c *Collection) Update(ctx context.Context, filter types.Filter, doc interface{}) error {
	rid := ctx.Value(common.ContextRequestIDField)
	start := time.Now()
	defer func() {
		blog.V(4).InfoDepthf(2, "mongo update cost %dms, rid: %v", time.Since(start)/time.Millisecond, rid)
	}()

	// 设置ctx的Session对象,用来处理事务
	se := &mongo.SessionExposer{}
	if c.hasSession(ctx) {
		sess, err := c.getDistributedSession(ctx)
		if err != nil {
			return err
		}
		ctx = se.ContextWithSession(ctx, sess)
		defer c.tm.SaveSession(sess)
	} else if c.sess != nil {
		ctx = se.ContextWithSession(ctx, c.sess)
	}

	if filter == nil {
		filter = bson.M{}
	}
	data := bson.M{"$set": doc}
	_, err := c.dbc.Database(c.dbname).Collection(c.collName).UpdateMany(ctx, filter, data)
	return err
}

// Upsert 数据存在更新数据，否则新加数据
func (c *Collection) Upsert(ctx context.Context, filter types.Filter, doc interface{}) error {
	rid := ctx.Value(common.ContextRequestIDField)
	start := time.Now()
	defer func() {
		blog.V(4).InfoDepthf(2, "mongo upsert cost %dms, rid: %v", time.Since(start)/time.Millisecond, rid)
	}()

	// 设置ctx的Session对象,用来处理事务
	se := &mongo.SessionExposer{}
	if c.hasSession(ctx) {
		sess, err := c.getDistributedSession(ctx)
		if err != nil {
			return err
		}
		ctx = se.ContextWithSession(ctx, sess)
		defer c.tm.SaveSession(sess)
	} else if c.sess != nil {
		ctx = se.ContextWithSession(ctx, c.sess)
	}

	// set upsert option
	upsert := true
	replaceOpt := &options.UpdateOptions{
		Upsert: &upsert,
	}
	data := bson.M{"$set": doc}

	_, err := c.dbc.Database(c.dbname).Collection(c.collName).UpdateOne(ctx, filter, data, replaceOpt)
	return err
}

// UpdateMultiModel 根据不同的操作符去更新数据
func (c *Collection) UpdateMultiModel(ctx context.Context, filter types.Filter, updateModel ...types.ModeUpdate) error {
	rid := ctx.Value(common.ContextRequestIDField)
	start := time.Now()
	defer func() {
		blog.V(4).InfoDepthf(2, "mongo update-multi-model cost %dms, rid: %v", time.Since(start)/time.Millisecond, rid)
	}()

	// 设置ctx的Session对象,用来处理事务
	se := &mongo.SessionExposer{}
	if c.hasSession(ctx) {
		sess, err := c.getDistributedSession(ctx)
		if err != nil {
			return err
		}
		ctx = se.ContextWithSession(ctx, sess)
		defer c.tm.SaveSession(sess)
	} else if c.sess != nil {
		ctx = se.ContextWithSession(ctx, c.sess)
	}

	data := bson.M{}
	for _, item := range updateModel {
		if _, ok := data[item.Op]; ok {
			return errors.New(item.Op + " appear multiple times")
		}
		data["$"+item.Op] = item.Doc
	}

	_, err := c.dbc.Database(c.dbname).Collection(c.collName).UpdateMany(ctx, filter, data)
	return err
}

// UpdateModifyCount 更新数据,返回更新的条数
func (c *Collection) UpdateModifyCount(ctx context.Context, filter types.Filter, doc interface{}) (int64, error) {
	rid := ctx.Value(common.ContextRequestIDField)
	start := time.Now()
	defer func() {
		blog.V(4).InfoDepthf(2, "mongo update-modify-count cost %dms, rid: %v", time.Since(start)/time.Millisecond, rid)
	}()

	// 设置ctx的Session对象,用来处理事务
	se := &mongo.SessionExposer{}
	if c.hasSession(ctx) {
		sess, err := c.getDistributedSession(ctx)
		if err != nil {
			return 0, err
		}
		ctx = se.ContextWithSession(ctx, sess)
		defer c.tm.SaveSession(sess)
	} else if c.sess != nil {
		ctx = se.ContextWithSession(ctx, c.sess)
	}

	data := bson.M{"$set": doc}
	result, err := c.dbc.Database(c.dbname).Collection(c.collName).UpdateMany(ctx, filter, data)
	if err != nil {
		return 0, nil
	}
	return result.ModifiedCount, nil
}

// Delete 删除数据
func (c *Collection) Delete(ctx context.Context, filter types.Filter) error {
	rid := ctx.Value(common.ContextRequestIDField)
	start := time.Now()
	defer func() {
		blog.V(4).InfoDepthf(2, "mongo delete cost %dms, rid: %v", time.Since(start)/time.Millisecond, rid)
	}()

	// 设置ctx的Session对象,用来处理事务
	se := &mongo.SessionExposer{}
	if c.hasSession(ctx) {
		sess, err := c.getDistributedSession(ctx)
		if err != nil {
			return err
		}
		ctx = se.ContextWithSession(ctx, sess)
		defer c.tm.SaveSession(sess)
	} else if c.sess != nil {
		ctx = se.ContextWithSession(ctx, c.sess)
	}

	_, err := c.dbc.Database(c.dbname).Collection(c.collName).DeleteMany(ctx, filter)
	return err
}

// NextSequence 获取新序列号(非事务), TODO test
func (c *Mongo) NextSequence(ctx context.Context, sequenceName string) (uint64, error) {
	rid := ctx.Value(common.ContextRequestIDField)
	start := time.Now()
	defer func() {
		blog.V(4).InfoDepthf(2, "mongo next-sequence cost %dms, rid: %v", time.Since(start)/time.Millisecond, rid)
	}()

	// 直接使用新的context，确保不会用到事务,不会因为context含有session而使用分布式事务，防止产生相同的序列号
	ctx = context.Background()

	coll := c.dbc.Database(c.dbname).Collection("cc_idgenerator")

	Update := bson.M{
		"$inc":         bson.M{"SequenceID": int64(1)},
		"$setOnInsert": bson.M{"create_time": time.Now()},
		"$set":         bson.M{"last_time": time.Now()},
	}
	filter := bson.M{"_id": sequenceName}
	upsert := true
	returnChange := options.After
	opt := &options.FindOneAndUpdateOptions{
		Upsert:         &upsert,
		ReturnDocument: &returnChange,
	}

	doc := Idgen{}
	err := coll.FindOneAndUpdate(ctx, filter, Update, opt).Decode(&doc)
	if err != nil {
		return 0, err
	}
	return doc.SequenceID, err
}

type Idgen struct {
	ID         string `bson:"_id"`
	SequenceID uint64 `bson:"SequenceID"`
}

// HasTable 判断是否存在集合
func (c *Mongo) HasTable(ctx context.Context, collName string) (bool, error) {
	cursor, err := c.dbc.Database(c.dbname).ListCollections(ctx, bson.M{"name": collName, "type": "collection"})
	if err != nil {
		return false, err
	}

	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		return true, nil
	}

	return false, nil
}

// DropTable 移除集合
func (c *Mongo) DropTable(ctx context.Context, collName string) error {
	return c.dbc.Database(c.dbname).Collection(collName).Drop(ctx)
}

// CreateTable 创建集合 TODO test
func (c *Mongo) CreateTable(ctx context.Context, collName string) error {
	return c.dbc.Database(c.dbname).RunCommand(ctx, map[string]interface{}{"create": collName}).Err()
}

// CreateIndex 创建索引
func (c *Collection) CreateIndex(ctx context.Context, index types.Index) error {
	createIndexOpt := &options.IndexOptions{
		Background: &index.Background,
		Unique:     &index.Unique,
	}
	if index.Name != "" {
		createIndexOpt.Name = &index.Name
	}

	createIndexInfo := mongo.IndexModel{
		Keys:    index.Keys,
		Options: createIndexOpt,
	}

	indexView := c.dbc.Database(c.dbname).Collection(c.collName).Indexes()
	_, err := indexView.CreateOne(ctx, createIndexInfo)
	return err
}

// DropIndex remove index by name
func (c *Collection) DropIndex(ctx context.Context, indexName string) error {
	indexView := c.dbc.Database(c.dbname).Collection(c.collName).Indexes()
	_, err := indexView.DropOne(ctx, indexName)
	return err
}

// Indexes get all indexes for the collection
func (c *Collection) Indexes(ctx context.Context) ([]types.Index, error) {
	indexView := c.dbc.Database(c.dbname).Collection(c.collName).Indexes()
	cursor, err := indexView.List(ctx)
	if nil != err {
		return nil, err
	}
	defer cursor.Close(ctx)
	var indexs []types.Index
	for cursor.Next(ctx) {
		idxResult := types.Index{}
		cursor.Decode(&idxResult)
		indexs = append(indexs, idxResult)
	}

	return indexs, nil
}

// AddColumn add a new column for the collection
func (c *Collection) AddColumn(ctx context.Context, column string, value interface{}) error {
	rid := ctx.Value(common.ContextRequestIDField)
	start := time.Now()
	defer func() {
		blog.V(4).InfoDepthf(2, "mongo add-column cost: %sms, rid: %s", time.Since(start)/time.Millisecond, rid)
	}()

	selector := dtype.Document{column: dtype.Document{"$exists": false}}
	datac := dtype.Document{"$set": dtype.Document{column: value}}
	_, err := c.dbc.Database(c.dbname).Collection(c.collName).UpdateMany(ctx, selector, datac)
	return err
}

// RenameColumn rename a column for the collection
func (c *Collection) RenameColumn(ctx context.Context, oldName, newColumn string) error {
	rid := ctx.Value(common.ContextRequestIDField)
	start := time.Now()
	defer func() {
		blog.V(4).InfoDepthf(2, "mongo rename-column cost: %sms, rid: %s", time.Since(start)/time.Millisecond, rid)
	}()

	datac := dtype.Document{"$rename": dtype.Document{oldName: newColumn}}
	_, err := c.dbc.Database(c.dbname).Collection(c.collName).UpdateMany(ctx, dtype.Document{}, datac)
	return err
}

// DropColumn remove a column by the name
func (c *Collection) DropColumn(ctx context.Context, field string) error {
	rid := ctx.Value(common.ContextRequestIDField)
	start := time.Now()
	defer func() {
		blog.V(4).InfoDepthf(2, "mongo drop-column cost: %sms, rid: %s", time.Since(start)/time.Millisecond, rid)
	}()

	datac := dtype.Document{"$unset": dtype.Document{field: ""}}
	_, err := c.dbc.Database(c.dbname).Collection(c.collName).UpdateMany(ctx, dtype.Document{}, datac)
	return err
}

// DropColumns remove many columns by the name
func (c *Collection) DropColumns(ctx context.Context, filter types.Filter, fields []string) error {
	unsetFields := make(map[string]interface{})
	for _, field := range fields {
		unsetFields[field] = ""
	}
	datac := dtype.Document{"$unset": unsetFields}
	_, err := c.dbc.Database(c.dbname).Collection(c.collName).UpdateMany(ctx, filter, datac)
	return err
}

// AggregateAll aggregate all operation
func (c *Collection) AggregateAll(ctx context.Context, pipeline interface{}, result interface{}) error {
	rid := ctx.Value(common.ContextRequestIDField)
	start := time.Now()
	defer func() {
		blog.V(4).InfoDepthf(2, "mongo aggregate-all cost %dms, rid: %v", time.Since(start)/time.Millisecond, rid)
	}()

	// 设置ctx的Session对象,用来处理事务
	se := &mongo.SessionExposer{}
	if c.hasSession(ctx) {
		sess, err := c.getDistributedSession(ctx)
		if err != nil {
			return err
		}
		ctx = se.ContextWithSession(ctx, sess)
		defer c.tm.SaveSession(sess)
	} else if c.sess != nil {
		ctx = se.ContextWithSession(ctx, c.sess)
	}

	cursor, err := c.dbc.Database(c.dbname).Collection(c.collName).Aggregate(ctx, pipeline)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)
	return decodeCusorIntoSlice(ctx, cursor, result)
}

// AggregateOne aggregate one operation
func (c *Collection) AggregateOne(ctx context.Context, pipeline interface{}, result interface{}) error {
	rid := ctx.Value(common.ContextRequestIDField)
	start := time.Now()
	defer func() {
		blog.V(4).InfoDepthf(2, "mongo aggregate-one cost %dms, rid: %v", time.Since(start)/time.Millisecond, rid)
	}()

	// 设置ctx的Session对象,用来处理事务
	se := &mongo.SessionExposer{}
	if c.hasSession(ctx) {
		sess, err := c.getDistributedSession(ctx)
		if err != nil {
			return err
		}
		ctx = se.ContextWithSession(ctx, sess)
		defer c.tm.SaveSession(sess)
	} else if c.sess != nil {
		ctx = se.ContextWithSession(ctx, c.sess)
	}

	cursor, err := c.dbc.Database(c.dbname).Collection(c.collName).Aggregate(ctx, pipeline)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		return cursor.Decode(result)
	}
	return types.ErrDocumentNotFound
}

func decodeCusorIntoSlice(ctx context.Context, cursor *mongo.Cursor, result interface{}) error {
	resultv := reflect.ValueOf(result)
	if resultv.Kind() != reflect.Ptr || resultv.Elem().Kind() != reflect.Slice {
		return errors.New("result argument must be a slice address")
	}

	elemt := resultv.Elem().Type().Elem()
	slice := reflect.MakeSlice(resultv.Elem().Type(), 0, 10)
	for cursor.Next(ctx) {
		elemp := reflect.New(elemt)
		if err := cursor.Decode(elemp.Interface()); nil != err {
			return err
		}
		slice = reflect.Append(slice, elemp.Elem())
	}
	if err := cursor.Err(); err != nil {
		return err
	}

	resultv.Elem().Set(slice)
	return nil
}
