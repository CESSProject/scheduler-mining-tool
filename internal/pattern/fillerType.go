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

package pattern

import (
	"sync"

	"github.com/CESSProject/cess-scheduler/pkg/chain"
)

type Fillermetamap struct {
	lock        *sync.Mutex
	Fillermetas map[string][]chain.FillerMetaInfo
}

var FillerMap *Fillermetamap

func init() {
	FillerMap = new(Fillermetamap)
	FillerMap.Fillermetas = make(map[string][]chain.FillerMetaInfo)
	FillerMap.lock = new(sync.Mutex)
}

func (this *Fillermetamap) Add(pubkey string, data chain.FillerMetaInfo) {
	this.lock.Lock()
	defer this.lock.Unlock()
	_, ok := this.Fillermetas[pubkey]
	if !ok {
		this.Fillermetas[pubkey] = make([]chain.FillerMetaInfo, 0)
	}
	this.Fillermetas[pubkey] = append(this.Fillermetas[pubkey], data)
}

func (this *Fillermetamap) GetNum(pubkey string) int {
	this.lock.Lock()
	defer this.lock.Unlock()

	return len(this.Fillermetas[pubkey])
}

func (this *Fillermetamap) Delete(pubkey string) {
	this.lock.Lock()
	defer this.lock.Unlock()
	delete(this.Fillermetas, pubkey)
}

func (this *Fillermetamap) Lock() {
	this.lock.Lock()
}

func (this *Fillermetamap) UnLock() {
	this.lock.Unlock()
}
