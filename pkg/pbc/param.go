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

package pbc

const (
	Success            = 200
	Error              = 201
	ErrorParam         = 202
	ErrorParamNotFound = 203
	ErrorInternal      = 204
)

//————————————————————————————————————————————————————————————————Implement KeyGen()————————————————————————————————————————————————————————————————

type PBCKeyPair struct {
	Spk          []byte
	Ssk          []byte
	SharedParams string
	SharedG      []byte
	ZrLength     uint
}

//————————————————————————————————————————————————————————————————Implement SigGen()————————————————————————————————————————————————————————————————

// Sigma is σ
type Sigma = []byte

// T be the file tag for F
type T struct {
	Tag
	SigAbove []byte
}

// Tag belongs to T
type Tag struct {
	Name []byte `json:"name"`
	N    int64  `json:"n"`
	U    []byte `json:"u"`
}

// SigGenResponse is result of SigGen() step
type SigGenResponse struct {
	T           T         `json:"t"`
	Phi         []Sigma   `json:"phi"`           //Φ = {σi}
	SigRootHash []byte    `json:"sig_root_hash"` //BLS
	StatueMsg   StatueMsg `json:"statue_msg"`
}

type StatueMsg struct {
	StatusCode int    `json:"status"`
	Msg        string `json:"msg"`
}

//————————————————————————————————————————————————————————————————Implement ChalGen()————————————————————————————————————————————————————————————————

type QElement struct {
	I int64  `json:"i"`
	V []byte `json:"v"`
}

//————————————————————————————————————————————————————————————————Implement GenProof()————————————————————————————————————————————————————————————————

type GenProofResponse struct {
	Sigma Sigma  `json:"sigmas"`
	MU    []byte `json:"mu"`
	MHTInfo
	SigRootHash []byte    `json:"sig_root_hash"`
	StatueMsg   StatueMsg `json:"statue_msg"`
}

type MHTInfo struct {
	HashMi [][]byte `json:"hash_mi"`
	Omega  []byte   `json:"omega"`
}

var PbcKey = PBCKeyPair{
	Ssk:          []byte{55, 32, 220, 181, 208, 19, 253, 239, 98, 230, 99, 252, 121, 44, 39, 145, 251, 44, 7, 84},
	Spk:          []byte{10, 220, 75, 195, 174, 36, 186, 176, 59, 223, 170, 199, 177, 143, 223, 147, 220, 84, 132, 101, 54, 112, 120, 144, 219, 28, 230, 129, 240, 127, 161, 4, 193, 25, 118, 181, 98, 3, 34, 200, 217, 50, 125, 125, 26, 120, 139, 11, 63, 0, 223, 99, 217, 72, 24, 157, 225, 79, 157, 168, 219, 170, 73, 134, 74, 223, 196, 139, 171, 223, 110, 21, 54, 36, 247, 187, 95, 40, 251, 4, 11, 92, 93, 105, 206, 67, 21, 31, 255, 227, 9, 166, 11, 194, 117, 81, 227, 225, 25, 170, 140, 120, 254, 100, 174, 110, 180, 158, 45, 0, 197, 150, 193, 71, 30, 34, 233, 90, 5, 64, 37, 163, 246, 121, 176, 26, 201, 174},
	SharedParams: string([]byte{116, 121, 112, 101, 32, 97, 10, 113, 32, 54, 53, 56, 50, 55, 51, 49, 54, 52, 56, 53, 50, 52, 55, 48, 50, 57, 55, 56, 52, 54, 54, 48, 54, 50, 53, 51, 48, 57, 53, 56, 57, 50, 49, 51, 56, 53, 57, 50, 55, 57, 54, 55, 48, 54, 55, 48, 50, 51, 56, 48, 53, 51, 55, 51, 53, 49, 49, 51, 51, 51, 55, 55, 51, 56, 51, 57, 49, 55, 57, 49, 55, 56, 56, 53, 54, 48, 52, 49, 55, 51, 51, 54, 48, 51, 57, 55, 56, 51, 50, 55, 51, 48, 50, 48, 56, 54, 49, 57, 52, 50, 56, 49, 56, 53, 50, 51, 49, 48, 49, 57, 56, 54, 48, 52, 48, 55, 48, 48, 48, 50, 51, 55, 49, 57, 54, 57, 56, 50, 55, 50, 56, 57, 50, 49, 56, 57, 53, 53, 55, 56, 52, 57, 49, 52, 51, 57, 51, 49, 52, 56, 48, 56, 51, 10, 104, 32, 57, 48, 48, 56, 49, 55, 53, 53, 51, 55, 50, 52, 54, 49, 49, 52, 54, 50, 53, 55, 51, 55, 56, 56, 57, 52, 50, 57, 52, 53, 53, 49, 57, 48, 57, 48, 57, 48, 48, 49, 52, 53, 48, 49, 55, 51, 56, 54, 52, 49, 54, 56, 54, 56, 52, 48, 48, 53, 54, 53, 50, 54, 55, 52, 48, 49, 49, 56, 49, 51, 55, 54, 51, 56, 57, 49, 56, 57, 49, 56, 57, 55, 55, 54, 49, 55, 50, 55, 49, 52, 53, 56, 53, 56, 55, 55, 50, 49, 56, 54, 49, 50, 53, 54, 52, 52, 10, 114, 32, 55, 51, 48, 55, 53, 48, 56, 49, 56, 54, 54, 53, 52, 53, 50, 55, 53, 55, 49, 55, 54, 48, 53, 55, 48, 53, 48, 48, 54, 53, 48, 52, 56, 54, 52, 50, 52, 53, 50, 48, 52, 56, 53, 55, 54, 53, 49, 49, 10, 101, 120, 112, 50, 32, 49, 53, 57, 10, 101, 120, 112, 49, 32, 49, 49, 48, 10, 115, 105, 103, 110, 49, 32, 49, 10, 115, 105, 103, 110, 48, 32, 45, 49, 10}),
	SharedG:      []byte{6, 82, 21, 158, 104, 141, 100, 78, 98, 180, 126, 135, 86, 92, 214, 75, 221, 27, 157, 4, 92, 203, 235, 234, 39, 170, 30, 218, 100, 100, 155, 185, 152, 19, 67, 73, 171, 46, 16, 231, 150, 190, 83, 175, 106, 104, 182, 175, 58, 112, 114, 96, 155, 77, 179, 139, 236, 226, 12, 9, 236, 20, 191, 94, 103, 130, 95, 226, 185, 125, 59, 33, 243, 126, 130, 246, 152, 60, 57, 144, 29, 40, 248, 89, 176, 174, 34, 187, 149, 8, 186, 232, 192, 164, 130, 21, 17, 145, 25, 151, 165, 105, 78, 11, 210, 212, 85, 243, 54, 83, 190, 179, 6, 67, 145, 56, 123, 208, 75, 19, 183, 220, 98, 129, 37, 7, 81, 243},
	ZrLength:     1024 * 1024,
}
