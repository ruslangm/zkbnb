/*
 * Copyright © 2021 ZkBNB Protocol
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
 *
 */

package nft

import (
	"github.com/bnb-chain/zkbnb/types"
	"gorm.io/gorm"
)

const (
	L2NftMetadataHistoryTableName = `l2_nft_metadata_history`
)

type (
	L2NftMetadataHistoryModel interface {
		CreateL2NftMetadataHistoryTable() error
		DropL2NftMetadataHistoryTable() error
		CreateL2NftMetadataHistoryInTransact(metadata *L2NftMetadataHistory) error
	}

	defaultL2NftMetadataHistoryModel struct {
		table string
		DB    *gorm.DB
	}

	L2NftMetadataHistory struct {
		gorm.Model
		NftIndex int64
		Cid      string
		IpnsId   string
		Mutable  string
	}
)

func NewL2NftMetadataHistoryModel(db *gorm.DB) L2NftMetadataHistoryModel {
	return &defaultL2NftMetadataHistoryModel{
		table: L2NftMetadataHistoryTableName,
		DB:    db,
	}
}

func (*L2NftMetadataHistory) TableName() string {
	return L2NftMetadataHistoryTableName
}

func (m *defaultL2NftMetadataHistoryModel) CreateL2NftMetadataHistoryTable() error {
	return m.DB.AutoMigrate(L2NftMetadataHistory{})
}

func (m *defaultL2NftMetadataHistoryModel) DropL2NftMetadataHistoryTable() error {
	return m.DB.Migrator().DropTable(m.table)
}
func (m *defaultL2NftMetadataHistoryModel) CreateL2NftMetadataHistoryInTransact(metadata *L2NftMetadataHistory) (err error) {
	dbTx := m.DB.Table(m.table).Create(metadata)
	if dbTx.Error != nil {
		return dbTx.Error
	}
	if dbTx.RowsAffected == 0 {
		return types.DbErrFailToCreateProof
	}
	return nil
}
