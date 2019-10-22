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

package connection

import(
	"context"
	"fmt"
	"io"
	"time"
	"bytes"

	"github.com/mongodb/mongo-go-driver/x/network/wiremessage"
)


var (
	SendChan chan []byte
	RecChan chan []byte
)

func init() {
	SendChan = make(chan []byte, 1)
	RecChan = make(chan []byte, 1)
}

type cmdbconn struct {
	*connection
}


func (c *cmdbconn) WriteWireMessage(ctx context.Context, wm wiremessage.WireMessage) error {
	var err error
	if c.connection.dead {
		return Error{
			ConnectionID: c.connection.id,
			message:      "connection is dead",
		}
	}

	select {
	case <-ctx.Done():
		return Error{
			ConnectionID: c.connection.id,
			Wrapped:      ctx.Err(),
			message:      "failed to write",
		}
	default:
	}

	deadline := time.Time{}
	if c.connection.writeTimeout != 0 {
		deadline = time.Now().Add(c.connection.writeTimeout)
	}

	if dl, ok := ctx.Deadline(); ok && (deadline.IsZero() || dl.Before(deadline)) {
		deadline = dl
	}

	if err := c.connection.conn.SetWriteDeadline(deadline); err != nil {
		return Error{
			ConnectionID: c.connection.id,
			Wrapped:      err,
			message:      "failed to set write deadline",
		}
	}

	// Truncate the write buffer
	c.connection.writeBuf = c.connection.writeBuf[:0]

	messageToWrite := wm
	// Compress if possible
	if c.connection.compressor != nil {
		compressed, err := c.connection.compressMessage(wm)
		if err != nil {
			return Error{
				ConnectionID: c.connection.id,
				Wrapped:      err,
				message:      "unable to compress wire message",
			}
		}
		messageToWrite = compressed
	}

	c.connection.writeBuf, err = messageToWrite.AppendWireMessage(c.connection.writeBuf)
	if err != nil {
		return Error{
			ConnectionID: c.connection.id,
			Wrapped:      err,
			message:      "unable to encode wire message",
		}
	}

	fmt.Printf("***** 111 the byte stream SendChan <- , size: %d ******\n", len(c.connection.writeBuf))

	SendChan <- c.connection.writeBuf
	

	// _, err = c.connection.conn.Write(c.connection.writeBuf)
	// if err != nil {
	// 	c.connection.Close()
	// 	return Error{
	// 		ConnectionID: c.connection.id,
	// 		Wrapped:      err,
	// 		message:      "unable to write wire message to network",
	// 	}
	// }

	c.connection.bumpIdleDeadline()
	err = c.connection.commandStartedEvent(ctx, wm)
	if err != nil {
		return err
	}
	return nil
}

func (c *cmdbconn) ReadWireMessage(ctx context.Context) (wiremessage.WireMessage, error) {
	if c.connection.dead {
		return nil, Error{
			ConnectionID: c.connection.id,
			message:      "connection is dead",
		}
	}

	select {
	case <-ctx.Done():
		// We close the connection because we don't know if there
		// is an unread message on the wire.
		c.connection.Close()
		return nil, Error{
			ConnectionID: c.connection.id,
			Wrapped:      ctx.Err(),
			message:      "failed to read",
		}
	default:
	}

	deadline := time.Time{}
	if c.connection.readTimeout != 0 {
		deadline = time.Now().Add(c.connection.readTimeout)
	}

	if ctxDL, ok := ctx.Deadline(); ok && (deadline.IsZero() || ctxDL.Before(deadline)) {
		deadline = ctxDL
	}

	if err := c.connection.conn.SetReadDeadline(deadline); err != nil {
		return nil, Error{
			ConnectionID: c.connection.id,
			Wrapped:      ctx.Err(),
			message:      "failed to set read deadline",
		}
	}

	//**************************
	// var sizeBuf [4]byte
	// _, err := io.ReadFull(c.connection.conn, sizeBuf[:])
	// if err != nil {
	// 	c.connection.Close()
	// 	return nil, Error{
	// 		ConnectionID: c.connection.id,
	// 		Wrapped:      err,
	// 		message:      "unable to decode message length",
	// 	}
	// }

	// size := readInt32(sizeBuf[:], 0)

	// // Isn't the best reuse, but resizing a []byte to be larger
	// // is difficult.
	// if cap(c.connection.readBuf) > int(size) {
	// 	c.connection.readBuf = c.connection.readBuf[:size]
	// } else {
	// 	c.connection.readBuf = make([]byte, size)
	// }

	// c.connection.readBuf[0], c.connection.readBuf[1], c.connection.readBuf[2], c.connection.readBuf[3] = sizeBuf[0], sizeBuf[1], sizeBuf[2], sizeBuf[3]

	// _, err = io.ReadFull(c.connection.conn, c.connection.readBuf[4:])
	// if err != nil {
	// 	c.connection.Close()
	// 	return nil, Error{
	// 		ConnectionID: c.connection.id,
	// 		Wrapped:      err,
	// 		message:      "unable to read full message",
	// 	}
	// }
	//*******************************

	respData := <- RecChan
	// fmt.Printf("***** 444 <- RecChan , size: ******:%d\n", len(respData))
	respReader := bytes.NewReader(respData)

	var sizeBuf [4]byte
	_, err := io.ReadFull(respReader, sizeBuf[:])
	if err != nil {
		c.connection.Close()
		return nil, Error{
			ConnectionID: c.connection.id,
			Wrapped:      err,
			message:      "unable to decode message length",
		}
	}

	size := readInt32(sizeBuf[:], 0)

	// Isn't the best reuse, but resizing a []byte to be larger
	// is difficult.
	if cap(c.connection.readBuf) > int(size) {
		c.connection.readBuf = c.connection.readBuf[:size]
	} else {
		c.connection.readBuf = make([]byte, size)
	}

	c.connection.readBuf[0], c.connection.readBuf[1], c.connection.readBuf[2], c.connection.readBuf[3] = sizeBuf[0], sizeBuf[1], sizeBuf[2], sizeBuf[3]

	_, err = io.ReadFull(respReader, c.connection.readBuf[4:])
	if err != nil {
		c.connection.Close()
		return nil, Error{
			ConnectionID: c.connection.id,
			Wrapped:      err,
			message:      "unable to read full message",
		}
	}

	hdr, err := wiremessage.ReadHeader(c.connection.readBuf, 0)
	if err != nil {
		c.connection.Close()
		return nil, Error{
			ConnectionID: c.connection.id,
			Wrapped:      err,
			message:      "unable to decode header",
		}
	}

	messageToDecode := c.connection.readBuf
	opcodeToCheck := hdr.OpCode

	if hdr.OpCode == wiremessage.OpCompressed {
		var compressed wiremessage.Compressed
		err := compressed.UnmarshalWireMessage(c.connection.readBuf)
		if err != nil {
			defer c.connection.Close()
			return nil, Error{
				ConnectionID: c.connection.id,
				Wrapped:      err,
				message:      "unable to decode OP_COMPRESSED",
			}
		}

		uncompressed, origOpcode, err := c.connection.uncompressMessage(compressed)
		if err != nil {
			defer c.connection.Close()
			return nil, Error{
				ConnectionID: c.connection.id,
				Wrapped:      err,
				message:      "unable to uncompress message",
			}
		}
		messageToDecode = uncompressed
		opcodeToCheck = origOpcode
	}

	var wm wiremessage.WireMessage
	switch opcodeToCheck {
	case wiremessage.OpReply:
		var r wiremessage.Reply
		err := r.UnmarshalWireMessage(messageToDecode)
		if err != nil {
			c.connection.Close()
			return nil, Error{
				ConnectionID: c.connection.id,
				Wrapped:      err,
				message:      "unable to decode OP_REPLY",
			}
		}
		wm = r
	case wiremessage.OpMsg:
		var reply wiremessage.Msg
		err := reply.UnmarshalWireMessage(messageToDecode)
		if err != nil {
			c.connection.Close()
			return nil, Error{
				ConnectionID: c.connection.id,
				Wrapped:      err,
				message:      "unable to decode OP_MSG",
			}
		}
		wm = reply
	default:
		c.connection.Close()
		return nil, Error{
			ConnectionID: c.connection.id,
			message:      fmt.Sprintf("opcode %s not implemented", hdr.OpCode),
		}
	}

	c.connection.bumpIdleDeadline()
	err = c.connection.commandFinishedEvent(ctx, wm)
	if err != nil {
		return nil, err // TODO: do we care if monitoring fails?
	}

	return wm, nil
}
