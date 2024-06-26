/*
 * Copyright 2021 CloudWeGo
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"context"
	"time"

	"github.com/cloudwego/netpoll"
)

func main() {
	network, address := "tcp", ":8000"
	listener, _ := netpoll.CreateListener(network, address)

	eventLoop, _ := netpoll.NewEventLoop(
		handle,
		netpoll.WithOnPrepare(prepare),
		netpoll.WithOnConnect(connect),
		netpoll.WithReadTimeout(time.Second),
	)

	// start listen loop ...
	eventLoop.Serve(listener)
}

var _ netpoll.OnPrepare = prepare
var _ netpoll.OnConnect = connect
var _ netpoll.OnRequest = handle
var _ netpoll.CloseCallback = close

func prepare(connection netpoll.Connection) context.Context {
	return context.Background()
}

func close(connection netpoll.Connection) error {
	return nil
}

func connect(ctx context.Context, connection netpoll.Connection) context.Context {
	connection.AddCloseCallback(close)
	return ctx
}

func handle(ctx context.Context, connection netpoll.Connection) error {
	reader, writer := connection.Reader(), connection.Writer()
	defer reader.Release()

	msg, _ := reader.ReadString(reader.Len())

	writer.WriteString(msg)
	writer.Flush()

	return nil
}
