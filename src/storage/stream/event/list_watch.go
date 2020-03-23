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

package event

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"configcenter/src/common/blog"
	"configcenter/src/common/json"
	"configcenter/src/storage/stream/types"
	"github.com/tidwall/gjson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (e *Event) ListWatch(ctx context.Context, opts *types.ListWatchOptions) (*types.Watcher, error) {
	if err := opts.CheckSetDefault(); err != nil {
		return nil, err
	}
	pipeline, streamOptions := generateOptions(&opts.Options)

	// TODO: should use the mongodb cluster timestamp, if the time is not synchronise with
	// mongodb cluster time, then we may have to lost some events.
	// A better way is to get the mongodb cluster timestamp and set it
	// as the "startTime".
	startAt := time.Now()
	streamOptions.StartAtOperationTime = &primitive.Timestamp{
		// normally, a unix time seconds is a int64 value,
		// but mongodb has a 32 bit T represent a unix seconds time.
		// calculate this: time.Duration(math.MaxInt32 - int32(time.Now().Unix()))/(time.Hour * 24 *365/time.Second))
		// the value is 17 years, it's okay for now.
		// reference: https://docs.mongodb.com/manual/reference/bson-types/#timestamps
		T: uint32(startAt.Unix()),
		I: 1,
	}

	// we watch the stream at first, so that we can know if we can watch success.
	// and, we do not read the event stream immediately, we wait until all the data
	// has been listed from database.
	stream, err := e.client.Database(e.database).
		Collection(opts.Collection).
		Watch(ctx, pipeline, streamOptions)
	if err != nil {
		blog.Errorf("mongodb watch failed with conf: %v, err: %v", *opts, err)
		return nil, fmt.Errorf("watch collection: %s failed, err: %v", opts.Collection, err)
	}

	// prepare for list all the data.
	totalCnt, err := e.client.Database(e.database).
		Collection(opts.Collection).
		CountDocuments(ctx, opts.Filter)
	if err != nil {
		// close the event stream.
		stream.Close(ctx)

		return nil, fmt.Errorf("count db %s, collection: %s with filter: %+v failed, err: %v",
			e.database, opts.Collection, opts.Filter, err)
	}

	eventChan := make(chan *types.Event, types.DefaultEventChanSize)
	go func() {
		// list all the data from the collection and send it as an event now.
		e.lister(ctx, totalCnt, opts, eventChan)

		select {
		case <-ctx.Done():
			blog.Errorf("received stopped watch signal, stop list db: %s, collection: %s, err: %v", e.database,
				opts.Collection, ctx.Err())
			return
		default:

		}

		// tell the user that the list operation has already done.
		// we only send for once.
		eventChan <- &types.Event{
			Oid:           "",
			Document:      reflect.New(reflect.TypeOf(opts.EventStruct)).Elem().Interface(),
			OperationType: types.ListDone,
		}

		// all the data has already listed and send the event.
		// now, it's time to watch the event stream.
		e.loopWatch(ctx, &opts.Options, streamOptions, stream, pipeline, eventChan)
	}()

	watcher := &types.Watcher{
		EventChan: eventChan,
	}
	return watcher, nil

}

func (e *Event) lister(ctx context.Context, cnt int64, opts *types.ListWatchOptions, ch chan *types.Event) {

	pageSize := *opts.PageSize
	for start := 0; start < int(cnt); start += pageSize {
		reset := func() {
			// sleep a while and retry later
			time.Sleep(3 * time.Second)
		}

		findOpts := new(options.FindOptions)
		findOpts.SetSkip(int64(start))
		findOpts.SetLimit(int64(pageSize))

	retry:
		cursor, err := e.client.Database(e.database).
			Collection(opts.Collection).
			Find(ctx, opts.Filter, findOpts)
		if err != nil {
			blog.Errorf("list watch operation, but list db: %s, collection: %s failed, will *retry later*, err: %v",
				e.database, opts.Collection, err)
			reset()
			continue
		}

		for cursor.Next(ctx) {
			select {
			case <-ctx.Done():
				blog.Errorf("received stopped lister signal, stop list db: %s, collection: %s, err: %v", e.database,
					opts.Collection, ctx.Err())
				return
			default:

			}

			// create a new event struct for use
			result := reflect.New(reflect.TypeOf(opts.EventStruct)).Elem()
			err := cursor.Decode(result.Addr().Interface())
			if err != nil {
				blog.Errorf("list watch operation, but list db: %s, collection: %s with cursor failed, will *retry later*, err: %v",
					e.database, opts.Collection, err)

				reset()
				cursor.Close(ctx)
				goto retry
			}

			byt, _ := json.Marshal(result.Addr().Interface())
			oid := gjson.GetBytes(byt, "_id").String()

			// send the event now
			ch <- &types.Event{
				Oid:           oid,
				Document:      result.Interface(),
				OperationType: types.Lister,
				DocBytes:      byt,
			}
		}

		if err := cursor.Err(); err != nil {
			blog.Errorf("list watch operation, but list db: %s, collection: %s with cursor failed, will *retry later*, err: %v",
				e.database, opts.Collection, err)
			reset()
			cursor.Close(ctx)
			goto retry
		}
		cursor.Close(ctx)
	}

}
