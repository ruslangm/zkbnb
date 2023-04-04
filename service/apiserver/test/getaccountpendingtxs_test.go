package test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bnb-chain/zkbnb/service/apiserver/internal/types"
)

func (s *ApiServerSuite) TestGetAccountPoolTxs() {
	type args struct {
		by      string
		value   string
		txTypes []int64
	}

	type testcase struct {
		name     string
		args     args
		httpCode int
	}

	tests := []testcase{
		{"not found by index", args{"account_index", "9999999", nil}, 200},
		{"not found by name", args{"l1_address", "notexists.legend", nil}, 200},
		{"invalidby", args{"invalidby", "", nil}, 400},
	}

	statusCode, txs := GetPendingTxs(s, 0, 100)
	if statusCode == http.StatusOK && len(txs.Txs) > 0 {
		tx := txs.Txs[len(txs.Txs)-1]
		_, account := GetAccount(s, "l1_address", tx.ToL1Address)
		tests = append(tests, []testcase{
			{"found by index", args{"account_index", strconv.Itoa(int(account.Index)), nil}, 200},
			{"found by l1_address", args{"l1_address", account.L1Address, nil}, 200},
			{"found by index and type", args{"account_index", strconv.Itoa(int(account.Index)), []int64{tx.Type}}, 200},
			{"not found by index and type", args{"account_index", strconv.Itoa(int(account.Index)), []int64{10000}}, 200},
		}...)
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			httpCode, result := GetAccountPendingTxs(s, tt.args.by, tt.args.value, tt.args.txTypes)
			assert.Equal(t, tt.httpCode, httpCode)
			if httpCode == http.StatusOK {
				if result.Total > 0 {
					assert.True(t, len(result.Txs) > 0)
					assert.NotNil(t, result.Txs[0].BlockHeight)
					assert.NotNil(t, result.Txs[0].Hash)
					assert.NotNil(t, result.Txs[0].Type)
					assert.NotNil(t, result.Txs[0].StateRoot)
					assert.NotNil(t, result.Txs[0].Info)
					assert.NotNil(t, result.Txs[0].Status)
				}
				fmt.Printf("result: %+v \n", result)
			}
		})
	}

}

func GetAccountPendingTxs(s *ApiServerSuite, by, value string, txTypes []int64) (int, *types.Txs) {
	url := fmt.Sprintf("%s/api/v1/accountPendingTxs?by=%s&value=%s", s.url, by, value)
	if len(txTypes) > 0 {
		data, _ := json.Marshal(txTypes)
		url += fmt.Sprintf("&types=%s", string(data))
	}
	resp, err := http.Get(url)
	assert.NoError(s.T(), err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	assert.NoError(s.T(), err)

	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, nil
	}
	result := types.Txs{}
	_ = json.Unmarshal(body, &result)
	return resp.StatusCode, &result
}
