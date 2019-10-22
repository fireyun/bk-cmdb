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
package driver_test

import (
	"context"
	"fmt"
	"io"
	"testing"

	"configcenter/src/storage/mongodb"
	"configcenter/src/storage/mongodb/driver"

	"github.com/mongodb/mongo-go-driver/x/network/connection"
	"github.com/stretchr/testify/require"
)

func init() {
	go clientPassMsg()
}

func createConnection() mongodb.CommonClient {
	//return driver.NewClient("mongodb://cc:cc@localhost:27011,localhost:27012,localhost:27013,localhost:27014/cmdb")
	return driver.NewClient("mongodb://cc:cc@localhost:27014/cmdb")
}

func executeCommand(t *testing.T, callback func(dbclient mongodb.CommonClient)) {
	dbClient := createConnection()
	require.NoError(t, dbClient.Open())

	callback(dbClient)

	require.NoError(t, dbClient.Close())
}

func aTestDatabaseName(t *testing.T) {

	executeCommand(t, func(dbClient mongodb.CommonClient) {
		t.Log("database name:", dbClient.Database().Name())
		require.Equal(t, "cmdb", dbClient.Database().Name())
	})
}

func aTestDatabaseHasCollection(t *testing.T) {

	executeCommand(t, func(dbClient mongodb.CommonClient) {
		exists, err := dbClient.Database().HasCollection("cc_tmp")
		require.Equal(t, true, exists)
		require.NoError(t, err)
	})
}

func aTestDatabaseDropCollection(t *testing.T) {
	executeCommand(t, func(dbClient mongodb.CommonClient) {
		require.NoError(t, dbClient.Database().DropCollection("tmptest"))
	})
}

func TestDatabaseGetCollectionNames(t *testing.T) {
	executeCommand(t, func(dbClient mongodb.CommonClient) {
		collNames, err := dbClient.Database().GetCollectionNames()
		require.NoError(t, err)
		for _, name := range collNames {
			t.Log("colloction:", name)
		}
	})
}


func aaTestTwoClients(t *testing.T) {
	go clientPassMsg()
	clientRequest(t)

}

func clientRequest(t *testing.T) {
	c1 := createConnection()
	require.NoError(t, c1.Open())
	collNames, err := c1.Database().GetCollectionNames()
	require.NoError(t, err)
	for _, name := range collNames {
		fmt.Println("colloction:", name)
		//t.Log("colloction:", name)
	}
	require.NoError(t, c1.Database().DropCollection("tmptest"))
	collNames2, err := c1.Database().GetCollectionNames()
	require.NoError(t, err)
	for _, name := range collNames2 {
		fmt.Println("colloction:", name)
		//t.Log("colloction:", name)
	}
}

func clientPassMsg() {
	ports := []string{"27011","27012","27013","27014"}
	i := 0
	for{
	port := ports[i%4]
	fmt.Println("connect port is:", port)
	c2 := driver.NewClient("mongodb://cc:cc@localhost:"+port+"/cmdb")
	i++
	// c2 := createConnection()
	err := c2.Open()
	if err != nil {
		panic("open err")
	}
	// for {
		// conn, err := c2.GetInnerClient().GetWriteClientConn(context.Background())
		conn, err := c2.GetInnerClient().GetReadClientConn(context.Background())
		if err != nil {
			fmt.Printf("GetReadClientConn err:%#v", err)
			// return Error{
			// 	ConnectionID: c.c.id,
			// 	Wrapped:      err,
			// 	message:      "unable to write wire message to network",
			// }
		}
		defer conn.Close()
		opData := <-connection.SendChan
		// fmt.Printf("***** 222 <- SendChan ,opData size: %d bytes*****\n", len(opData))
		netconn := conn.GetConn()
		_, err = netconn.Write(opData)
		if err != nil {
			netconn.Close()
			fmt.Printf("conn write err:%#v", err)
			// return Error{
			// 	ConnectionID: c.c.id,
			// 	Wrapped:      err,
			// 	message:      "unable to write wire message to network",
			// }
		}

		var sizeBuf [4]byte
		_, err = io.ReadFull(netconn, sizeBuf[:])
		if err != nil {
			netconn.Close()
			// return nil, Error{
			// 	ConnectionID: c.c.id,
			// 	Wrapped:      err,
			// 	message:      "unable to decode message length",
			// }
		}

		size := readInt32(sizeBuf[:], 0)

		// readBuf := make([]byte, 256)
		// // Isn't the best reuse, but resizing a []byte to be larger
		// // is difficult.
		// if cap(readBuf) > int(size) {
		// 	readBuf = readBuf[:size]
		// } else {
		readBuf := make([]byte, size)
		// }

		readBuf[0], readBuf[1], readBuf[2], readBuf[3] = sizeBuf[0], sizeBuf[1], sizeBuf[2], sizeBuf[3]

		_, err = io.ReadFull(netconn, readBuf[4:])
		if err != nil {
			netconn.Close()
			// return nil, Error{
			// 	ConnectionID: c.c.id,
			// 	Wrapped:      err,
			// 	message:      "unable to read full message",
			// }
		}
		fmt.Printf("***** 333 RecChan<- %d bytes *****\n", len(readBuf))
		connection.RecChan <- readBuf
	}
}

func readInt32(b []byte, pos int32) int32 {
	return (int32(b[pos+0])) | (int32(b[pos+1]) << 8) | (int32(b[pos+2]) << 16) | (int32(b[pos+3]) << 24)
}
