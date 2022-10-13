/*
   Copyright 2022 CESS scheduler authors

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

        http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package com

import (
	"encoding/json"
	"sync"
)

type MsgType byte

const (
	MsgInvalid MsgType = iota
	MsgHead
	MsgFile
	MsgEnd
	MsgNotify
	MsgClose
	MsgFillerHead
	MsgFillerHeadNotify
	MsgFiller
	MsgFillerNotify
)

type Status byte

const (
	Status_Ok Status = iota
	Status_Err
)

type Message struct {
	MsgType  MsgType `json:"msg_type"`
	FileName string  `json:"file_name"`
	FileHash string  `json:"file_hash"`
	FileSize uint64  `json:"file_size"`
	Pubkey   []byte  `json:"pub_key"`
	SignMsg  []byte  `json:"sign_msg"`
	Sign     []byte  `json:"sign"`
	Bytes    []byte  `json:"bytes"`
}

type Notify struct {
	Status byte
}

var (
	msgPool = sync.Pool{
		New: func() any {
			return &Message{}
		},
	}

	BytesPool = sync.Pool{
		New: func() any {
			return make([]byte, 40*1024)
		},
	}
)

func (m *Message) GC() {
	if m.MsgType == MsgFile {
		BytesPool.Put(m.Bytes[:cap(m.Bytes)])
	}
	m.reset()
	msgPool.Put(m)
}

func (m *Message) reset() {
	m.MsgType = MsgInvalid
	m.FileName = ""
	m.FileHash = ""
	m.FileSize = 0
	m.Pubkey = nil
	m.SignMsg = nil
	m.Sign = nil
	m.Bytes = nil
}

func (m *Message) String() string {
	bytes, _ := json.Marshal(m)
	return string(bytes)
}

// Decode will convert from bytes
func Decode(b []byte) (m *Message, err error) {
	m = msgPool.Get().(*Message)
	err = json.Unmarshal(b, &m)
	return
}

func NewNotifyMsg(fileName string, status Status) *Message {
	m := msgPool.Get().(*Message)
	m.MsgType = MsgNotify
	m.Bytes = []byte{byte(status)}
	m.FileName = fileName
	m.FileHash = ""
	m.Pubkey = nil
	m.SignMsg = nil
	m.Sign = nil
	return m
}

func NewHeadMsg(fileName, fileHash string, pubkey, signmsg, sign []byte) *Message {
	m := msgPool.Get().(*Message)
	m.MsgType = MsgHead
	m.FileName = fileName
	m.FileHash = fileHash
	m.Pubkey = pubkey
	m.SignMsg = signmsg
	m.Sign = sign
	return m
}

func NewFileMsg(fileName, fileHash string, buf []byte) *Message {
	m := msgPool.Get().(*Message)
	m.MsgType = MsgFile
	m.FileName = fileName
	m.FileHash = fileHash
	m.Bytes = buf
	return m
}

func NewEndMsg(fileName string, size uint64) *Message {
	m := msgPool.Get().(*Message)
	m.MsgType = MsgEnd
	m.FileName = fileName
	m.FileSize = size
	return m
}

func NewCloseMsg(fileName string, status Status) *Message {
	m := msgPool.Get().(*Message)
	m.MsgType = MsgClose
	m.Bytes = []byte{byte(status)}
	m.FileName = fileName
	return m
}
