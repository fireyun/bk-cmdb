// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"

	"github.com/mongodb/mongo-go-driver/x/network/description"
	"github.com/mongodb/mongo-go-driver/x/network/connection"
)



func (c *Client) GetReadClientConn(ctx context.Context) (connection.Connection, error) {


	selector := description.CompositeSelector([]description.ServerSelector{
		description.ReadPrefSelector(c.readPreference),
		description.LatencySelector(c.localThreshold),
	})


	ss, err := c.topology.SelectServer(ctx, selector)
	if err != nil {
		return nil, err
	}

	conn, err := ss.Connection(ctx)
	if err != nil {
		return nil, err
	}
	return conn, nil
}



func (c *Client) GetWriteClientConn(ctx context.Context) (connection.Connection, error) {


	selector := description.CompositeSelector([]description.ServerSelector{
		description.WriteSelector(),
		description.LatencySelector(c.localThreshold),
	})


	ss, err := c.topology.SelectServer(ctx, selector)
	if err != nil {
		return nil, err
	}

	conn, err := ss.Connection(ctx)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
